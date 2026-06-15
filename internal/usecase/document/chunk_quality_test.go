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

func TestBuildRAGContext_fallbackSourcesWhenAllLowQuality(t *testing.T) {
	chunks := []entity.SimilarChunk{
		{
			DocumentChunk: entity.DocumentChunk{Content: "Equation a. D = B A B B A A P P J B J B - = -"},
			Similarity:    0.58,
		},
		{
			DocumentChunk: entity.DocumentChunk{Content: "Equation b. x = Y Z Z Y Y"},
			Similarity:    0.57,
		},
	}

	_, sources := buildRAGContext(chunks)
	if len(sources) != 2 {
		t.Fatalf("len(sources) = %d, want fallback top 2 sources", len(sources))
	}
}
