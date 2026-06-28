package document

import (
	"strings"
	"testing"
)

func TestIsSummaryQuery(t *testing.T) {
	cases := map[string]bool{
		"buatkan rangkuman dari dokumen ini": true,
		"ringkasan materi":                   true,
		"siapa penulis buku ini":             false,
		"apa yang dibahas":                   false,
	}

	for query, want := range cases {
		if got := isSummaryQuery(query); got != want {
			t.Fatalf("isSummaryQuery(%q) = %v, want %v", query, got, want)
		}
	}
}

func TestScopedDocumentInstruction(t *testing.T) {
	instruction := scopedDocumentInstruction("media.pdf")
	if instruction == "" {
		t.Fatal("expected non-empty instruction")
	}
	for _, part := range []string{"media.pdf", "HANYA", "dokumen lain"} {
		if !strings.Contains(instruction, part) {
			t.Fatalf("instruction missing %q: %s", part, instruction)
		}
	}
}
