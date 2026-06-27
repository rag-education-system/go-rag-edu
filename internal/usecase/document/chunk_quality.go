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
	minRelevantSimilarity = 0.45
	garbledRatioThreshold = 0.35
	minChunkLength        = 50
	minWordCount          = 5
)

var googleDocsPattern = regexp.MustCompile(`(?i)\b(google\s+docs|gdocs)\b`)
var docsOnlyPattern = regexp.MustCompile(`(?i)\bdocs\b`)
var wordPattern = regexp.MustCompile(`(?i)\b(microsoft\s+word|ms\s+word|\bword\b)`)
var prodiFooterPattern = regexp.MustCompile(`(?i)pendidikan teknik informatika dan komputer\s+\d+\s*$`)
var programStudyQueryPattern = regexp.MustCompile(`(?i)\b(prodi|jurusan|program studi|keunggulan|kuliah)\b`)

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

func documentLabel(chunk entity.SimilarChunk) string {
	if name := strings.TrimSpace(chunk.DocumentName); name != "" {
		return name
	}
	if chunk.DocumentID != "" {
		return chunk.DocumentID
	}
	return "dokumen"
}

func buildRAGContext(chunks []entity.SimilarChunk) (string, []entity.SimilarChunk) {
	var contextBuilder strings.Builder
	displaySources := make([]entity.SimilarChunk, 0, len(chunks))

	docIndex := 0
	for _, chunk := range chunks {
		if chunk.Similarity < minRelevantSimilarity {
			continue
		}

		docIndex++
		qualityNote := ""
		if isGarbledText(chunk.Content) {
			qualityNote = " - Catatan: sebagian teks (terutama rumus/simbol) mungkin rusak setelah ekstraksi dokumen"
		}

		contextBuilder.WriteString(fmt.Sprintf(
			"[Dokumen: %s - Similarity: %.2f%s]\n%s\n\n",
			documentLabel(chunk),
			chunk.Similarity,
			qualityNote,
			CleanReadableContent(chunk.Content),
		))

		if !isChunkLowQuality(chunk) {
			displaySources = append(displaySources, chunk)
		}
	}

	if docIndex == 0 {
		return "", nil
	}

	return strings.TrimSpace(contextBuilder.String()), displaySources
}

type scoredChunk struct {
	chunk entity.SimilarChunk
	score float64
}

func BuildRAGContextWithQuery(query string, chunks []entity.SimilarChunk) (string, []entity.SimilarChunk) {
	queryKeywords := extractQueryKeywords(query)

	scored := make([]scoredChunk, 0, len(chunks))
	for _, chunk := range chunks {
		score := scoreChunkRelevanceWithQuery(query, chunk, queryKeywords)
		scored = append(scored, scoredChunk{chunk: chunk, score: score})
	}

	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	var contextBuilder strings.Builder
	displaySources := make([]entity.SimilarChunk, 0, len(chunks))

	docIndex := 0
	for _, sc := range scored {
		isRelevant := sc.score >= minRelevantSimilarity

		if !isRelevant {
			continue
		}

		docIndex++
		qualityNote := ""
		if isGarbledText(sc.chunk.Content) {
			qualityNote = " - Catatan: sebagian teks (terutama rumus/simbol) mungkin rusak setelah ekstraksi dokumen"
		}

		contextBuilder.WriteString(fmt.Sprintf(
			"[Dokumen: %s - Score: %.2f%s]\n%s\n\n",
			documentLabel(sc.chunk),
			sc.score,
			qualityNote,
			CleanReadableContent(sc.chunk.Content),
		))

		shouldDisplay := sc.score >= minDisplaySimilarity && !isGarbledText(sc.chunk.Content)

		if shouldDisplay {
			displaySources = append(displaySources, sc.chunk)
		}
	}

	if docIndex == 0 {
		return "", nil
	}

	return strings.TrimSpace(contextBuilder.String()), AggregateSourcesByDocument(displaySources)
}

func AggregateSourcesByDocument(chunks []entity.SimilarChunk) []entity.SimilarChunk {
	if len(chunks) == 0 {
		return nil
	}

	bestByDoc := make(map[string]entity.SimilarChunk)
	for _, chunk := range chunks {
		if chunk.DocumentID == "" {
			continue
		}

		existing, ok := bestByDoc[chunk.DocumentID]
		if !ok || chunk.Similarity > existing.Similarity {
			bestByDoc[chunk.DocumentID] = chunk
		}
	}

	result := make([]entity.SimilarChunk, 0, len(bestByDoc))
	for _, chunk := range bestByDoc {
		result = append(result, chunk)
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Similarity > result[i].Similarity {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func extractQueryKeywords(query string) []string {
	query = strings.ToLower(query)
	words := strings.Fields(query)

	stopWords := map[string]bool{
		"yang": true, "dan": true, "di": true, "ke": true, "dari": true,
		"untuk": true, "dengan": true, "pada": true, "adalah": true,
		"ini": true, "itu": true, "atau": true, "juga": true, "saya": true,
		"apa": true, "bagaimana": true, "cara": true, "bisa": true, "dapat": true,
		"ada": true, "tidak": true, "akan": true, "sudah": true, "belum": true,
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"how": true, "what": true, "to": true, "in": true, "of": true,
		"membuat": true, "membuka": true, "menggunakan": true, "ingin": true,
		"mencari": true, "tolong": true, "mohon": true, "jelaskan": true,
	}

	var keywords []string
	for _, word := range words {
		word = strings.Trim(word, ".,?!\"'()[]{}:;")
		if len(word) >= 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func hasKeywordMatch(content string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}

	contentLower := strings.ToLower(content)
	matchCount := 0

	for _, keyword := range keywords {
		if strings.Contains(contentLower, keyword) {
			matchCount++
		}
	}

	return matchCount >= 1 && float64(matchCount)/float64(len(keywords)) >= 0.3
}

func isExplanatoryContent(content string) bool {
	contentLower := strings.ToLower(content)

	explanatoryPatterns := []string{
		"adalah", "merupakan", "berfungsi", "digunakan untuk",
		"berisi", "terdiri dari", "yaitu", "yakni",
		"pengertian", "definisi", "penjelasan",
		"menu", "tab", "fitur", "tombol",
	}

	matchCount := 0
	for _, pattern := range explanatoryPatterns {
		if strings.Contains(contentLower, pattern) {
			matchCount++
		}
	}

	return matchCount >= 2
}

func isExerciseContent(content string) bool {
	contentLower := strings.ToLower(content)

	exercisePatterns := []string{
		"penugasan", "tugas", "latihan", "praktikum",
		"buatlah", "kerjakan", "jawablah",
		"soal", "pertanyaan",
	}

	for _, pattern := range exercisePatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}

	return false
}

func isProdiFooterBoilerplate(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > 250 {
		return false
	}
	return prodiFooterPattern.MatchString(trimmed)
}

func isProgramStudyQuery(query string) bool {
	return programStudyQueryPattern.MatchString(query)
}

func hasProgramAdvantageContent(content string) bool {
	contentLower := strings.ToLower(content)
	if !strings.Contains(contentLower, "keunggulan") {
		return false
	}
	programHints := []string{
		"ptik", "pendidikan teknik informatika", "program studi",
		"universitas", "prodi", "jurusan", "spmb",
	}
	for _, hint := range programHints {
		if strings.Contains(contentLower, hint) {
			return true
		}
	}
	return false
}

func scoreChunkRelevance(chunk entity.SimilarChunk, queryKeywords []string) float64 {
	score := chunk.Similarity

	if hasKeywordMatch(chunk.Content, queryKeywords) {
		score += 0.10
	}

	if isExplanatoryContent(chunk.Content) {
		score += 0.08
	}

	if isExerciseContent(chunk.Content) {
		score -= 0.05
	}

	if isProdiFooterBoilerplate(chunk.Content) {
		score -= 0.20
	}

	if isGarbledText(chunk.Content) {
		score -= 0.15
	}

	garbledRat := garbledRatio(chunk.Content)
	if garbledRat > 0.2 {
		score -= garbledRat * 0.2
	}

	return score
}

func scoreChunkRelevanceWithQuery(query string, chunk entity.SimilarChunk, queryKeywords []string) float64 {
	score := scoreChunkRelevance(chunk, queryKeywords)

	if isProgramStudyQuery(query) && hasProgramAdvantageContent(chunk.Content) {
		score += 0.15
	}

	return score
}

func composeDocContext(query string, chunks []entity.SimilarChunk) (string, []entity.SimilarChunk) {
	contextText, displaySources := BuildRAGContextWithQuery(query, chunks)

	var builder strings.Builder
	if hint := detectApplicationHint(query); hint != "" {
		builder.WriteString(hint)
		builder.WriteString("\n\n")
	}
	builder.WriteString(contextText)

	return strings.TrimSpace(builder.String()), displaySources
}

func IsChunkWorthEmbedding(content string) bool {
	content = strings.TrimSpace(content)

	if len(content) < minChunkLength {
		return false
	}

	words := strings.Fields(content)
	if len(words) < minWordCount {
		return false
	}

	if garbledRatio(content) > 0.5 {
		return false
	}

	letterCount := 0
	for _, r := range content {
		if unicode.IsLetter(r) {
			letterCount++
		}
	}
	if float64(letterCount)/float64(len(content)) < 0.3 {
		return false
	}

	return true
}

func FilterChunksForEmbedding(chunks []string) (filtered []string, indices []int) {
	for i, chunk := range chunks {
		if IsChunkWorthEmbedding(chunk) {
			filtered = append(filtered, chunk)
			indices = append(indices, i)
		}
	}
	return filtered, indices
}
