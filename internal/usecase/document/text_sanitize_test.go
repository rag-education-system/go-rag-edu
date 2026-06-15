package document

import (
	"strings"
	"testing"
)

func TestCleanReadableContent_removesGarbledEquationTail(t *testing.T) {
	input := "SmartArt 3. Buatlah rumus berikut menggunakan fitur Equation a. D = B A B B A A P P J B J B - = - b. x = Y Z Z"
	got := CleanReadableContent(input)

	if got == input {
		t.Fatalf("CleanReadableContent() did not modify garbled input")
	}
	if containsGarbledEquation(got) {
		t.Fatalf("CleanReadableContent() = %q, still contains garbled equation", got)
	}
	if !containsAll(got, "fitur Equation", "Buatlah rumus") {
		t.Fatalf("CleanReadableContent() = %q, expected readable prefix to remain", got)
	}
}

func TestCleanReadableContent_preservesReadableText(t *testing.T) {
	input := "Langkah 1. Buka tab Insert. Langkah 2. Pilih Equation."
	got := CleanReadableContent(input)
	if got != input {
		t.Fatalf("CleanReadableContent() = %q, want unchanged readable text", got)
	}
}

func TestPickBetterExtractedText_prefersOCR(t *testing.T) {
	plain := "SmartArt 3. Buatlah rumus berikut menggunakan fitur Equation a. D = B A B B A A P P J B J B - = -"
	ocr := "3. Buatlah rumus berikut menggunakan fitur Equation. Insert > Equation > Insert New Equation."

	got := pickBetterExtractedText(plain, ocr)
	if !containsAll(got, "Insert", "Equation") {
		t.Fatalf("pickBetterExtractedText() = %q, expected OCR content to be preferred", got)
	}
}

func containsGarbledEquation(text string) bool {
	return garbledRatio(text) >= garbledRatioThreshold
}

func containsAll(text string, parts ...string) bool {
	lowerText := strings.ToLower(text)
	for _, part := range parts {
		if !strings.Contains(lowerText, strings.ToLower(part)) {
			return false
		}
	}
	return true
}
