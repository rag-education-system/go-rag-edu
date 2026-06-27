package document

import (
	"strings"
	"testing"

	"rag-api/internal/domain/entity"
)

func TestGarbledRatio_detectsBrokenEquationText(t *testing.T) {
	text := "SmartArt 3. Buatlah rumus berikut menggunakan fitur Equation a. D = B A B B A A P P J B J B - = -"
	ratio := garbledRatio(text)
	if ratio < garbledRatioThreshold {
		t.Fatalf("garbledRatio() = %v, expected >= %v for broken equation text", ratio, garbledRatioThreshold)
	}
}

func TestGarbledRatio_normalText(t *testing.T) {
	text := "Untuk membuat rumus, buka tab Insert lalu pilih Equation."
	ratio := garbledRatio(text)
	if ratio >= garbledRatioThreshold {
		t.Fatalf("garbledRatio() = %v, expected readable text to stay below threshold", ratio)
	}
}

func TestDetectApplicationHint_googleDocs(t *testing.T) {
	hint := detectApplicationHint("saya ingin membuat rumus matematika di docs, beritahu saya caranya")
	if hint == "" {
		t.Fatal("expected application hint for google docs query")
	}
	if !strings.Contains(strings.ToLower(hint), "google docs") {
		t.Fatalf("hint = %q, expected google docs guidance", hint)
	}
}

func TestDetectApplicationHint_wordQuery(t *testing.T) {
	hint := detectApplicationHint("saya ingin membuat rumus matematika di word")
	if hint != "" {
		t.Fatalf("detectApplicationHint() = %q, want empty hint for word query", hint)
	}
}

func TestBuildRAGContext_usesDocumentName(t *testing.T) {
	chunks := []entity.SimilarChunk{
		{
			DocumentChunk: entity.DocumentChunk{Content: "Konten tentang IoT dan sistem sampah."},
			Similarity:    0.81,
			DocumentName:  "jurnal-iot.pdf",
		},
	}

	contextText, _ := buildRAGContext(chunks)
	if !strings.Contains(contextText, "[Dokumen: jurnal-iot.pdf") {
		t.Fatalf("context = %q, expected document name in label", contextText)
	}
}

func TestBuildRAGContext_filtersLowQualitySources(t *testing.T) {
	chunks := []entity.SimilarChunk{
		{
			DocumentChunk: entity.DocumentChunk{Content: "SmartArt 3. Buatlah rumus berikut menggunakan fitur Equation a. D = B A B B A A P P J B J B - = -"},
			Similarity:    0.59,
		},
		{
			DocumentChunk: entity.DocumentChunk{Content: "Buka tab Insert lalu pilih Equation untuk membuat rumus matematika."},
			Similarity:    0.81,
		},
	}

	contextText, sources := buildRAGContext(chunks)
	if contextText == "" {
		t.Fatal("expected non-empty context")
	}
	if !strings.Contains(contextText, "rusak setelah ekstraksi dokumen") {
		t.Fatal("expected garbled quality note in context")
	}
	if len(sources) != 1 {
		t.Fatalf("len(sources) = %d, want 1 high-quality source", len(sources))
	}
	if sources[0].Similarity != 0.81 {
		t.Fatalf("sources[0].Similarity = %v, want 0.81", sources[0].Similarity)
	}
}

func TestScoreChunkRelevance_prefersPosterOverModuleFooter(t *testing.T) {
	query := "apa saja keunggulan dari prodi ptik"
	keywords := extractQueryKeywords(query)

	poster := entity.SimilarChunk{
		DocumentChunk: entity.DocumentChunk{
			Content: "Pendidikan Teknik Informatika & Komputer Universitas Bung Hatta Keunggulan Fasilitas Labor multimedia, komputer, jaringan, dan mikro-teaching.",
		},
		Similarity: 0.62,
	}
	modulFooter := entity.SimilarChunk{
		DocumentChunk: entity.DocumentChunk{
			Content: "Dengan microsoft word dapat memudahkan kerja manusia dalam melakukan pengetikan surat maupun dokumen lainnya. Pendidikan Teknik Informatika dan Komputer 1",
		},
		Similarity: 0.70,
	}

	posterScore := scoreChunkRelevanceWithQuery(query, poster, keywords)
	modulScore := scoreChunkRelevanceWithQuery(query, modulFooter, keywords)
	if posterScore <= modulScore {
		t.Fatalf("posterScore=%v modulScore=%v, expected poster chunk to outrank module footer", posterScore, modulScore)
	}
}

func TestExpandProgramAcronyms_includesPTIK(t *testing.T) {
	terms := expandProgramAcronyms([]string{"ptik", "keunggulan"})
	found := false
	for _, term := range terms {
		if term == "pendidikan teknik informatika" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("terms = %v, expected expanded PTIK acronym", terms)
	}
}

func TestBuildRAGContext_fallbackSourcesWhenAllLowQuality(t *testing.T) {
	chunks := []entity.SimilarChunk{
		{
			DocumentChunk: entity.DocumentChunk{DocumentID: "doc-1", Content: "Equation a. D = B A B B A A P P J B J B - = -"},
			Similarity:    0.58,
		},
		{
			DocumentChunk: entity.DocumentChunk{DocumentID: "doc-1", Content: "Equation b. x = Y Z Z Y Y"},
			Similarity:    0.57,
		},
	}

	_, sources := buildRAGContext(chunks)
	if len(sources) != 0 {
		t.Fatalf("len(sources) = %d, want 0 low-quality sources", len(sources))
	}
}

func TestAggregateSourcesByDocument_groupsByDocumentID(t *testing.T) {
	chunks := []entity.SimilarChunk{
		{
			DocumentChunk: entity.DocumentChunk{DocumentID: "doc-1", Content: "Chunk A"},
			Similarity:    0.72,
		},
		{
			DocumentChunk: entity.DocumentChunk{DocumentID: "doc-1", Content: "Chunk B"},
			Similarity:    0.81,
		},
		{
			DocumentChunk: entity.DocumentChunk{DocumentID: "doc-2", Content: "Chunk C"},
			Similarity:    0.76,
		},
	}

	aggregated := AggregateSourcesByDocument(chunks)
	if len(aggregated) != 2 {
		t.Fatalf("len(aggregated) = %d, want 2 documents", len(aggregated))
	}
	if aggregated[0].DocumentID != "doc-1" || aggregated[0].Similarity != 0.81 {
		t.Fatalf("aggregated[0] = %+v, want doc-1 with similarity 0.81", aggregated[0])
	}
}
