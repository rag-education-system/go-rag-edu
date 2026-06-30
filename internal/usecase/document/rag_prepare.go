package document

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"
)

var summaryQueryPattern = regexp.MustCompile(`(?i)\b(rangkuman|ringkasan|ringkas|summary|simpulan|kesimpulan|hafal|mempelajari|outline|poin\s+poin|gambarkan\s+isi|isi\s+dokumen)\b`)

func isSummaryQuery(query string) bool {
	return summaryQueryPattern.MatchString(strings.TrimSpace(query))
}

func scopedDocumentInstruction(docName string) string {
	name := strings.TrimSpace(docName)
	if name == "" {
		name = "dokumen ini"
	}
	return fmt.Sprintf(
		"[MODE DOKUMEN TUNGGAL: Jawab HANYA berdasarkan dokumen \"%s\" di bawah. "+
			"Jangan menyebut, merangkum, atau membandingkan dengan dokumen lain. "+
			"Jika pengguna meminta rangkuman, rangkum hanya isi dokumen ini.]\n\n",
		name,
	)
}

// defaultFullDocMaxChars caps the size of a document that may be sent in full.
// ~200k characters is roughly 50k tokens, comfortably within a 128k context
// window once the prompt and answer budget are accounted for.
const defaultFullDocMaxChars = 200000

// RAGResult holds retrieval output before LLM generation.
type RAGResult struct {
	Answer            string
	Chunks            []entity.SimilarChunk
	DisplaySources    []entity.SimilarChunk
	DocContext        string
	ReformulatedQuery string
	SearchType        string
}

// PrepareRAG runs reformulation → hybrid/vector search → context building (AI-Hukum-BE pipeline phase 2).
func (uc *DocumentUsecase) PrepareRAG(
	ctx context.Context,
	access docaccess.Context,
	query string,
	history []ChatMessage,
) (*RAGResult, error) {
	query = strings.TrimSpace(query)
	history = normalizeHistory(history)

	if isGreeting(query) && len(history) == 0 {
		return &RAGResult{
			Answer: "Halo! Saya siap membantu Anda. Silakan tanyakan apa saja tentang dokumen yang telah Anda upload.",
		}, nil
	}

	searchQuery := query
	reformulated := ""
	if uc.reformulator != nil && uc.reformulator.Enabled() && NeedsQueryReformulation(query, history) {
		if r, err := uc.reformulator.ReformulateQuery(ctx, query, history); err == nil && r != "" && r != query {
			reformulated = r
			searchQuery = r
		}
	}

	chunks, searchType, err := uc.searchRelevantChunks(ctx, access, query, searchQuery, history, "")
	if err != nil {
		return nil, err
	}

	uc.enrichChunkDocumentNames(ctx, chunks)

	docContext, displaySources := composeDocContext(query, chunks)

	return &RAGResult{
		Chunks:            chunks,
		DisplaySources:    displaySources,
		DocContext:        docContext,
		ReformulatedQuery: reformulated,
		SearchType:        searchType,
	}, nil
}

// PrepareRAGForDocument scopes retrieval to a single document. When the
// document fits within the full-document budget, its entire text is used as
// context so the model sees everything. Otherwise (or when documentID is
// empty) it falls back to the standard top-k retrieval pipeline.
func (uc *DocumentUsecase) PrepareRAGForDocument(
	ctx context.Context,
	access docaccess.Context,
	documentID string,
	query string,
	history []ChatMessage,
) (*RAGResult, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return uc.PrepareRAG(ctx, access, query, history)
	}

	query = strings.TrimSpace(query)
	history = normalizeHistory(history)

	if isGreeting(query) && len(history) == 0 {
		return uc.PrepareRAG(ctx, access, query, history)
	}

	// Selalu izinkan konteks parsial agar dokumen besar tidak jatuh ke RAG global.
	allowPartial := true
	docContext, sources, ok := uc.buildFullDocumentContext(ctx, access, documentID, allowPartial)
	if ok {
		return &RAGResult{
			Chunks:         sources,
			DisplaySources: AggregateSourcesByDocument(sources),
			DocContext:     docContext,
			SearchType:     "full_document",
		}, nil
	}

	return uc.prepareScopedRAG(ctx, access, documentID, query, history)
}

func (uc *DocumentUsecase) prepareScopedRAG(
	ctx context.Context,
	access docaccess.Context,
	documentID string,
	query string,
	history []ChatMessage,
) (*RAGResult, error) {
	query = strings.TrimSpace(query)
	history = normalizeHistory(history)

	searchQuery := query
	reformulated := ""
	if uc.reformulator != nil && uc.reformulator.Enabled() && NeedsQueryReformulation(query, history) {
		if r, err := uc.reformulator.ReformulateQuery(ctx, query, history); err == nil && r != "" && r != query {
			reformulated = r
			searchQuery = r
		}
	}

	chunks, searchType, err := uc.searchRelevantChunks(ctx, access, query, searchQuery, history, documentID)
	if err != nil {
		return nil, err
	}

	uc.enrichChunkDocumentNames(ctx, chunks)

	docName := uc.GetDocumentOriginalName(ctx, documentID)
	docContext, displaySources := composeDocContext(query, chunks)
	if docContext != "" {
		docContext = scopedDocumentInstruction(docName) + docContext
	}

	return &RAGResult{
		Chunks:            chunks,
		DisplaySources:    displaySources,
		DocContext:        docContext,
		ReformulatedQuery: reformulated,
		SearchType:        searchType + "_scoped",
	}, nil
}

// buildFullDocumentContext assembles the entire document text from its stored
// chunks (in order). It returns ok=false when the document is inaccessible,
// has no chunks, or exceeds the configured character budget — signalling the
// caller to fall back to retrieval.
func (uc *DocumentUsecase) buildFullDocumentContext(
	ctx context.Context,
	access docaccess.Context,
	documentID string,
	allowPartial bool,
) (string, []entity.SimilarChunk, bool) {
	doc, err := uc.docRepo.FindByIDWithAccess(ctx, documentID, access)
	if err != nil || doc == nil {
		return "", nil, false
	}

	chunks, err := uc.chunkRepo.FindByDocumentID(ctx, documentID)
	if err != nil || len(chunks) == 0 {
		return "", nil, false
	}

	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].ChunkIndex < chunks[j].ChunkIndex
	})

	maxChars := uc.fullDocMaxChars
	if maxChars <= 0 {
		maxChars = defaultFullDocMaxChars
	}

	var builder strings.Builder
	total := 0
	truncated := false
	similar := make([]entity.SimilarChunk, 0, len(chunks))

	for _, chunk := range chunks {
		content := strings.TrimSpace(CleanReadableContent(chunk.Content))
		if content == "" {
			continue
		}

		nextTotal := total + len(content) + 2
		if nextTotal > maxChars {
			if !allowPartial || total == 0 {
				return "", nil, false
			}
			truncated = true
			break
		}

		total = nextTotal
		builder.WriteString(content)
		builder.WriteString("\n\n")

		similar = append(similar, entity.SimilarChunk{
			DocumentChunk: chunk,
			Similarity:    1.0,
			DocumentName:  doc.OriginalName,
		})
	}

	body := strings.TrimSpace(builder.String())
	if body == "" {
		return "", nil, false
	}

	if truncated {
		body += "\n\n[Catatan: dokumen panjang; yang ditampilkan hanya sebagian awal.]"
	}

	label := strings.TrimSpace(doc.OriginalName)
	if label == "" {
		label = documentID
	}
	docContext := scopedDocumentInstruction(label) + fmt.Sprintf("[Dokumen lengkap: %s]\n%s", label, body)

	return docContext, similar, true
}

func (uc *DocumentUsecase) enrichChunkDocumentNames(ctx context.Context, chunks []entity.SimilarChunk) {
	names := make(map[string]string)
	for i := range chunks {
		docID := chunks[i].DocumentID
		if docID == "" || chunks[i].DocumentName != "" {
			continue
		}
		if name, ok := names[docID]; ok {
			chunks[i].DocumentName = name
			continue
		}
		name := uc.GetDocumentOriginalName(ctx, docID)
		names[docID] = name
		chunks[i].DocumentName = name
	}
}
