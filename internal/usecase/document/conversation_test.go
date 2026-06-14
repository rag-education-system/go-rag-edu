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
