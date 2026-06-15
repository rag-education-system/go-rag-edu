package document

import (
	"context"
	"strings"

	"rag-api/internal/domain/entity"
)

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
	userID string,
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
	if uc.reformulator != nil && uc.reformulator.Enabled() {
		if r, err := uc.reformulator.ReformulateQuery(ctx, query, history); err == nil && r != "" && r != query {
			reformulated = r
			searchQuery = r
		}
	}

	chunks, searchType, err := uc.searchRelevantChunks(ctx, userID, query, searchQuery, history)
	if err != nil {
		return nil, err
	}

	docContext, displaySources := composeDocContext(query, chunks)

	return &RAGResult{
		Chunks:            chunks,
		DisplaySources:    displaySources,
		DocContext:        docContext,
		ReformulatedQuery: reformulated,
		SearchType:        searchType,
	}, nil
}
