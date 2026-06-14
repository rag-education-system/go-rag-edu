package document

import (
	"fmt"
	"strings"
)

type ExtractionResult struct {
	Text   string
	Source string
}

type ContentExtractor struct {
	minTextLength int
	ocrEnabled    bool
	plain         *TextExtractor
	ocr           *OCRExtractor
}

func NewContentExtractor(ocrEnabled bool, ocrLang string, minTextLength int) *ContentExtractor {
	if minTextLength <= 0 {
		minTextLength = 80
	}

	return &ContentExtractor{
		minTextLength: minTextLength,
		ocrEnabled:    ocrEnabled,
		plain:         NewTextExtractor(),
		ocr:           NewOCRExtractor(ocrLang),
	}
}

func normalizeMimeType(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch mimeType {
	case "image/jpg":
		return "image/jpeg"
	default:
		return mimeType
	}
}

func isImageMimeType(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp":
		return true
	default:
		return false
	}
}

func (ce *ContentExtractor) Extract(data []byte, mimeType string) (ExtractionResult, error) {
	mimeType = normalizeMimeType(mimeType)

	switch mimeType {
	case "application/pdf":
		return ce.extractPDF(data)
	case "image/png", "image/jpeg", "image/webp":
		return ce.extractImage(data)
	default:
		return ExtractionResult{}, fmt.Errorf("unsupported file type: %s", mimeType)
	}
}

func (ce *ContentExtractor) extractPDF(data []byte) (ExtractionResult, error) {
	plainText, err := ce.plain.ExtractFromPDF(data)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("failed to extract PDF text: %w", err)
	}
	plainText = strings.TrimSpace(plainText)

	if len(plainText) >= ce.minTextLength {
		return ExtractionResult{Text: plainText, Source: "text"}, nil
	}

	if !ce.ocrEnabled {
		if plainText == "" {
			return ExtractionResult{}, fmt.Errorf("no text extracted from document")
		}
		return ExtractionResult{Text: plainText, Source: "text"}, nil
	}

	ocrText, ocrErr := ce.ocr.ExtractFromPDF(data)
	ocrText = strings.TrimSpace(ocrText)

	if ocrErr != nil {
		if plainText != "" {
			return ExtractionResult{Text: plainText, Source: "text"}, nil
		}
		return ExtractionResult{}, fmt.Errorf("OCR fallback failed: %w", ocrErr)
	}

	if plainText != "" && ocrText != "" {
		return ExtractionResult{
			Text:   plainText + "\n\n" + ocrText,
			Source: "mixed",
		}, nil
	}

	if ocrText != "" {
		return ExtractionResult{Text: ocrText, Source: "ocr"}, nil
	}

	if plainText != "" {
		return ExtractionResult{Text: plainText, Source: "text"}, nil
	}

	return ExtractionResult{}, fmt.Errorf("no text extracted from document")
}

func (ce *ContentExtractor) extractImage(data []byte) (ExtractionResult, error) {
	if !ce.ocrEnabled {
		return ExtractionResult{}, fmt.Errorf("OCR is disabled; image uploads require OCR")
	}

	text, err := ce.ocr.ExtractFromImage(data)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("failed to OCR image: %w", err)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return ExtractionResult{}, fmt.Errorf("no text detected in image")
	}

	return ExtractionResult{Text: text, Source: "ocr"}, nil
}
