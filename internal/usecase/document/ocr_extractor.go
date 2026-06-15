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

func (o *OCRExtractor) ExtractPagesFromPDF(data []byte) ([]PageText, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF for OCR: %w", err)
	}
	defer doc.Close()

	var pages []PageText
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

		text = strings.TrimSpace(text)
		if text != "" {
			pages = append(pages, PageText{
				PageNumber: i + 1,
				Text:       text,
			})
		}
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
