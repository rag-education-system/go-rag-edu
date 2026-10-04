//go:build cgo && !nocgo

package document

import (
	"fmt"
	"strings"

	"github.com/gen2brain/go-fitz"
)

// PDFOCRAvailable reports whether this binary can render PDF pages for OCR.
func PDFOCRAvailable() bool { return true }

// extractPagesFromPDFSelective renders and OCRs only the pages for which
// shouldOCR(pageNumber) returns true. Page numbers are 1-indexed. Pages that
// produce no text are skipped. A nil shouldOCR means OCR every page.
func (o *OCRExtractor) extractPagesFromPDFSelective(
	data []byte,
	shouldOCR func(pageNumber int) bool,
) ([]PageText, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF for OCR: %w", err)
	}
	defer doc.Close()

	var pages []PageText
	for i := 0; i < doc.NumPage(); i++ {
		pageNumber := i + 1
		if shouldOCR != nil && !shouldOCR(pageNumber) {
			continue
		}

		imgData, err := doc.ImagePNG(i, o.dpi)
		if err != nil {
			continue
		}

		text, err := o.runTesseract(imgData)
		if err != nil {
			continue
		}

		text = strings.TrimSpace(text)
		if text != "" {
			pages = append(pages, PageText{
				PageNumber: pageNumber,
				Text:       text,
			})
		}
	}

	return pages, nil
}
