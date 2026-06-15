package document

import (
	"fmt"
	"strings"
)

type ExtractionResult struct {
	Text   string
	Source string
	Pages  []PageText
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
	plainPages, err := ce.plain.ExtractPagesFromPDF(data)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("failed to extract PDF text: %w", err)
	}
	plainText := strings.TrimSpace(joinPageTexts(plainPages))
	plainGarbled := isGarbledText(plainText)
	needsOCR := len(plainText) < ce.minTextLength || plainGarbled

	if !needsOCR {
		return ExtractionResult{
			Text:   CleanReadableContent(plainText),
			Source: "text",
			Pages:  plainPages,
		}, nil
	}

	if !ce.ocrEnabled {
		if plainText == "" {
			return ExtractionResult{}, fmt.Errorf("no text extracted from document")
		}
		return ExtractionResult{
			Text:   CleanReadableContent(plainText),
			Source: "text",
			Pages:  plainPages,
		}, nil
	}

	ocrPages, ocrErr := ce.ocr.ExtractPagesFromPDF(data)
	ocrText := strings.TrimSpace(joinPageTexts(ocrPages))

	if ocrErr != nil {
		if plainText != "" {
			return ExtractionResult{
				Text:   CleanReadableContent(plainText),
				Source: "text",
				Pages:  plainPages,
			}, nil
		}
		return ExtractionResult{}, fmt.Errorf("OCR fallback failed: %w", ocrErr)
	}

	source := "ocr"
	selectedPages := ocrPages
	combined := ocrText
	if plainText != "" && ocrText != "" {
		source = "mixed"
		combined = pickBetterExtractedText(plainText, ocrText)
		if combined == plainText {
			selectedPages = plainPages
		}
	} else if plainText != "" {
		source = "text"
		combined = plainText
		selectedPages = plainPages
	}

	combined = CleanReadableContent(combined)
	if combined == "" {
		return ExtractionResult{}, fmt.Errorf("no text extracted from document")
	}

	return ExtractionResult{Text: combined, Source: source, Pages: selectedPages}, nil
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

	return ExtractionResult{
		Text:   text,
		Source: "ocr",
		Pages:  []PageText{{PageNumber: 1, Text: text}},
	}, nil
}
