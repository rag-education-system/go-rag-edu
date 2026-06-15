package document

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

type TextExtractor struct{}

func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

func (te *TextExtractor) ExtractFromPDF(data []byte) (string, error) {
	pages, err := te.ExtractPagesFromPDF(data)
	if err != nil {
		return "", err
	}
	return joinPageTexts(pages), nil
}

func (te *TextExtractor) ExtractPagesFromPDF(data []byte) ([]PageText, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var pages []PageText
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		pages = append(pages, PageText{
			PageNumber: i,
			Text:       text,
		})
	}

	return pages, nil
}
