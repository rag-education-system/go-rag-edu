package document

import (
	"bytes"
	"fmt"
	"image/png"
	"os/exec"
	"strings"

	"github.com/gen2brain/go-fitz"
)

type OCRExtractor struct {
	lang string
}

func NewOCRExtractor(lang string) *OCRExtractor {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = "eng"
	}

	return &OCRExtractor{lang: lang}
}

func (o *OCRExtractor) ExtractFromImage(data []byte) (string, error) {
	return o.runTesseract(data)
}

func (o *OCRExtractor) ExtractFromPDF(data []byte) (string, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF for OCR: %w", err)
	}
	defer doc.Close()

	var pageTexts []string
	for i := 0; i < doc.NumPage(); i++ {
		img, err := doc.Image(i)
		if err != nil {
			continue
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			continue
		}

		text, err := o.runTesseract(buf.Bytes())
		if err != nil {
			continue
		}

		if text != "" {
			pageTexts = append(pageTexts, text)
		}
	}

	if len(pageTexts) == 0 {
		return "", fmt.Errorf("no text detected by OCR")
	}

	return strings.Join(pageTexts, "\n\n"), nil
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
