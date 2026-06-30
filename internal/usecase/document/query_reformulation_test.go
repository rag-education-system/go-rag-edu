package document

import "testing"

func TestNeedsQueryReformulation(t *testing.T) {
	history := []ChatMessage{{Role: "user", Content: "Apa itu fotosintesis?"}}

	cases := map[string]bool{
		"Apa itu algoritma sorting?": false,
		"jelaskan lebih lanjut":       true,
		"bagaimana?":                  true,
		"Apa perbedaan merge sort dan quick sort?": true,
	}

	for query, want := range cases {
		h := history
		if query == "Apa itu algoritma sorting?" {
			h = nil
		}
		if got := NeedsQueryReformulation(query, h); got != want {
			t.Fatalf("NeedsQueryReformulation(%q) = %v, want %v", query, got, want)
		}
	}
}
