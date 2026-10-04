package document

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// defaultOCRDPI controls the render resolution before running Tesseract.
// Higher DPI yields noticeably better accuracy on scanned documents and
// small fonts than the go-fitz default (~72 DPI).
const defaultOCRDPI = 300

type OCRExtractor struct {
	lang string
	dpi  float64
}

func NewOCRExtractor(lang string) *OCRExtractor {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = "eng"
	}

	return &OCRExtractor{lang: lang, dpi: defaultOCRDPI}
}

func (o *OCRExtractor) ExtractFromImage(data []byte) (string, error) {
	text, err := o.runTesseract(data)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func (o *OCRExtractor) ExtractPagesFromImage(data []byte) ([]PageText, error) {
	text, err := o.ExtractFromImage(data)
	if err != nil {
		return nil, err
	}
	if text == "" {
		return nil, fmt.Errorf("no text detected in image")
	}
	return []PageText{{PageNumber: 1, Text: text}}, nil
}

func (o *OCRExtractor) ExtractFromPDF(data []byte) (string, error) {
	pages, err := o.ExtractPagesFromPDF(data)
	if err != nil {
		return "", err
	}
	return joinPageTexts(pages), nil
}

// ExtractPagesFromPDF runs OCR on every page of the PDF.
func (o *OCRExtractor) ExtractPagesFromPDF(data []byte) ([]PageText, error) {
	pages, err := o.extractPagesFromPDFSelective(data, func(int) bool { return true })
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no text detected by OCR")
	}
	return pages, nil
}

func (o *OCRExtractor) runTesseract(imageData []byte) (string, error) {
	cmd := exec.Command("tesseract", "stdin", "stdout", "-l", o.lang)
	cmd.Stdin = bytes.NewReader(imageData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}
