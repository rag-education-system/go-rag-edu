package document

import "testing"

func TestNormalizeMimeType(t *testing.T) {
	tests := map[string]string{
		"image/jpg":  "image/jpeg",
		"image/JPEG": "image/jpeg",
		"application/pdf": "application/pdf",
	}

	for input, want := range tests {
		if got := normalizeMimeType(input); got != want {
			t.Fatalf("normalizeMimeType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsImageMimeType(t *testing.T) {
	if !isImageMimeType("image/png") {
		t.Fatal("expected png to be image mime")
	}
	if isImageMimeType("application/pdf") {
		t.Fatal("expected pdf not to be image mime")
	}
}
