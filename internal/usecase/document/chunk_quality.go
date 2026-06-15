package document

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"rag-api/internal/domain/entity"
)

const (
	minDisplaySimilarity  = 0.50
	garbledRatioThreshold = 0.35
)

var googleDocsPattern = regexp.MustCompile(`(?i)\b(google\s+docs|gdocs)\b`)
var docsOnlyPattern = regexp.MustCompile(`(?i)\bdocs\b`)
var wordPattern = regexp.MustCompile(`(?i)\b(microsoft\s+word|ms\s+word|\bword\b)`)

func garbledRatio(text string) float64 {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	isolatedLetters := 0
	for _, word := range words {
		runes := []rune(word)
		if len(runes) == 1 && unicode.IsLetter(runes[0]) {
			isolatedLetters++
		}
	}

	return float64(isolatedLetters) / float64(len(words))
}

func isGarbledText(text string) bool {
	return garbledRatio(text) >= garbledRatioThreshold
}

// IsGarbledChunkContent reports whether extracted chunk text likely contains broken symbols or equations.
func IsGarbledChunkContent(text string) bool {
	return isGarbledText(text)
}

// IsLowConfidenceSource reports whether a retrieved chunk should be shown as low confidence in the UI.
func IsLowConfidenceSource(similarity float64, content string) bool {
	return similarity < minDisplaySimilarity || isGarbledText(content)
}

func isChunkLowQuality(chunk entity.SimilarChunk) bool {
	return chunk.Similarity < minDisplaySimilarity || isGarbledText(chunk.Content)
}

func detectApplicationHint(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return ""
	}

	mentionsWord := wordPattern.MatchString(q)
	mentionsGoogleDocs := googleDocsPattern.MatchString(q) ||
		(docsOnlyPattern.MatchString(q) && !mentionsWord)

	if !mentionsGoogleDocs {
		return ""
	}

	return `PETUNJUK KONTEKS:
Pengguna menanyakan Google Docs. Dokumen yang diindeks kemungkinan membahas Microsoft Word (misalnya fitur Equation atau SmartArt).
Jika konteks merujuk Word, jelaskan secara eksplisit bahwa dokumen membahas Microsoft Word, bukan Google Docs.
Jangan memberikan langkah untuk Google Docs atau pengetahuan umum di luar konteks dokumen.`
}

func buildRAGContext(chunks []entity.SimilarChunk) (string, []entity.SimilarChunk) {
	var contextBuilder strings.Builder
	displaySources := make([]entity.SimilarChunk, 0, len(chunks))

	for i, chunk := range chunks {
		qualityNote := ""
		if isGarbledText(chunk.Content) {
			qualityNote = " - Catatan: sebagian teks (terutama rumus/simbol) mungkin rusak setelah ekstraksi dokumen"
		}

		contextBuilder.WriteString(fmt.Sprintf(
			"[Dokumen %d - Similarity: %.2f%s]\n%s\n\n",
			i+1,
			chunk.Similarity,
			qualityNote,
			CleanReadableContent(chunk.Content),
		))

		if !isChunkLowQuality(chunk) {
			displaySources = append(displaySources, chunk)
		}
	}

	if len(displaySources) == 0 && len(chunks) > 0 {
		limit := 2
		if len(chunks) < limit {
			limit = len(chunks)
		}
		displaySources = append(displaySources, chunks[:limit]...)
	}

	return strings.TrimSpace(contextBuilder.String()), displaySources
}

func composeDocContext(query string, chunks []entity.SimilarChunk) (string, []entity.SimilarChunk) {
	contextText, displaySources := buildRAGContext(chunks)

	var builder strings.Builder
	if hint := detectApplicationHint(query); hint != "" {
		builder.WriteString(hint)
		builder.WriteString("\n\n")
	}
	builder.WriteString(contextText)

	return strings.TrimSpace(builder.String()), displaySources
}
