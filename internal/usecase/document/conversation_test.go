package document

import (
	"strings"
	"testing"
)

func TestBuildSearchQuery(t *testing.T) {
	history := []ChatMessage{
		{Role: "user", Content: "Apa topik utama dokumen machine learning?"},
		{Role: "assistant", Content: "Topik utamanya adalah supervised learning dan neural networks."},
	}

	got := buildSearchQuery("jelaskan poin ketiganya", history)
	if !strings.Contains(got, "machine learning") || !strings.Contains(got, "jelaskan poin ketiganya") {
		t.Fatalf("buildSearchQuery() = %q, expected contextual query", got)
	}
}

func TestBuildSearchQueryIncludesAssistantContextForSolutionFollowUp(t *testing.T) {
	history := []ChatMessage{
		{Role: "user", Content: "apa masalah peneliti pada jurnal IoT sistem sampah?"},
		{Role: "assistant", Content: "Masalahnya meliputi kurangnya kesadaran masyarakat dan tempat sampah konvensional yang dianggap kotor pada sistem tempat sampah otomatis berbasis IoT."},
	}

	got := buildSearchQuery("lalu menurut penulis apa solusi untuk mengatasi masalah tersebut", history)
	if !strings.Contains(strings.ToLower(got), "iot") {
		t.Fatalf("buildSearchQuery() = %q, expected IoT context from assistant answer", got)
	}
	if !strings.Contains(strings.ToLower(got), "solusi") {
		t.Fatalf("buildSearchQuery() = %q, expected follow-up query", got)
	}
}

func TestExpandSolutionSynonyms(t *testing.T) {
	terms := expandSolutionSynonyms("apa solusi untuk mengatasi masalah tersebut", []string{"jurnal", "iot"})
	found := false
	for _, term := range terms {
		if term == "metode" || term == "rancang" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("terms = %v, expected solution-related synonyms", terms)
	}
}

func TestBuildSearchQueryIgnoresPollutedHistoryForDirectQuestion(t *testing.T) {
	history := []ChatMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "Maaf, saya tidak menemukan informasi yang relevan dalam dokumen"},
	}

	got := buildSearchQuery("berapa rata2 ipk 2016/2017", history)
	if got != "berapa rata2 ipk 2016/2017" {
		t.Fatalf("buildSearchQuery() = %q, expected direct query only", got)
	}
}

func TestExtractSearchTermsForIPKQuery(t *testing.T) {
	terms := extractSearchTerms("berapa rata2 ipk 2016/2017")
	if len(terms) == 0 {
		t.Fatal("expected search terms")
	}

	found := false
	for _, term := range terms {
		if term == "ipk" || term == "2016/2017" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ipk or year term, got %v", terms)
	}
}
