package document

import "testing"

func TestChunkText_shortPageProducesSingleChunk(t *testing.T) {
	chunker := NewChunker(1000, 200)
	text := "0823-8811-8474 Pendidikan Teknik Informatika & Komputer Universitas Bung Hatta Keunggulan Fasilitas Labor multimedia."

	chunks := chunker.ChunkText(text)
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1 for short poster-like page", len(chunks))
	}
}

func TestChunkText_noSingleCharacterSlidingWindow(t *testing.T) {
	chunker := NewChunker(1000, 200)
	text := "0852-6305-8769 Pendidikan Bahasa Inggris Universitas Bung Hatta Keunggulan Mata Kuliah Intensive Course melatih dasar-dasar Bahasa Inggris Small Classroom Peluang beasiswa KIP, LLDIKTI, BI, VDMI, dll siap kerja Dosen berkualifikasi Master dan Doktor dari dalam dan luar negeri Ayo Daftar Sekarang spmb.bunghatta.ac.id"

	chunks := chunker.ChunkText(text)
	if len(chunks) > 3 {
		t.Fatalf("len(chunks) = %d, want at most 3 without 1-char sliding duplicates", len(chunks))
	}
}

func TestChunkPages_oneChunkPerShortPosterPage(t *testing.T) {
	chunker := NewChunker(1000, 200)
	pages := []PageText{
		{PageNumber: 1, Text: "0852-6305-8769 Pendidikan Bahasa Inggris Universitas Bung Hatta Keunggulan Mata Kuliah Intensive Course."},
		{PageNumber: 2, Text: "0823-8811-8474 Pendidikan Teknik Informatika & Komputer Universitas Bung Hatta Keunggulan Fasilitas Labor multimedia, komputer, jaringan."},
	}

	contents, pageNumbers := chunker.ChunkPages(pages)
	if len(contents) != 2 {
		t.Fatalf("len(contents) = %d, want 2 (one chunk per poster page)", len(contents))
	}
	if pageNumbers[0] != 1 || pageNumbers[1] != 2 {
		t.Fatalf("pageNumbers = %v, want [1 2]", pageNumbers)
	}
}
