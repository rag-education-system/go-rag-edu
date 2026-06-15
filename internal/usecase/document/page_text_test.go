package document

import "testing"

func TestChunkPagesPreservesPageNumbers(t *testing.T) {
	chunker := NewChunker(100, 20)
	pages := []PageText{
		{PageNumber: 1, Text: "Halaman satu dengan cukup teks agar terpecah menjadi chunk."},
		{PageNumber: 5, Text: "Halaman lima berisi data IPK rata-rata ganjil 2016/2017 untuk beberapa prodi."},
	}

	contents, pageNumbers := chunker.ChunkPages(pages)
	if len(contents) == 0 {
		t.Fatal("expected chunks")
	}
	if len(contents) != len(pageNumbers) {
		t.Fatalf("chunk/page mismatch: %d vs %d", len(contents), len(pageNumbers))
	}

	foundPageOne := false
	foundPageFive := false
	for _, page := range pageNumbers {
		if page == 1 {
			foundPageOne = true
		}
		if page == 5 {
			foundPageFive = true
		}
	}
	if !foundPageOne || !foundPageFive {
		t.Fatalf("expected chunks on pages 1 and 5, got %v", pageNumbers)
	}
}
