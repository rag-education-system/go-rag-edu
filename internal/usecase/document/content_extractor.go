package document

import (
	"fmt"
	"sort"
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
	// Plain extraction may legitimately return no text (e.g. fully scanned PDF).
	// Treat any error as "no plain text" and let OCR take over instead of failing.
	plainPages, _ := ce.plain.ExtractPagesFromPDF(data)
	plainByPage := make(map[int]string, len(plainPages))
	for _, page := range plainPages {
		if text := strings.TrimSpace(page.Text); text != "" {
			plainByPage[page.PageNumber] = text
		}
	}
	plainText := strings.TrimSpace(joinPageTexts(plainPages))

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

	// Decide OCR per page: a page needs OCR when its text layer is too short or
	// looks garbled. Pages missing from the text layer (image-only pages) get an
	// empty string here and are therefore always OCR'd.
	needsPageOCR := func(pageNumber int) bool {
		text := plainByPage[pageNumber]
		return len(text) < ce.minTextLength || isGarbledText(text)
	}

	ocrPages, ocrErr := ce.ocr.extractPagesFromPDFSelective(data, needsPageOCR)
	ocrByPage := make(map[int]string, len(ocrPages))
	for _, page := range ocrPages {
		if text := strings.TrimSpace(page.Text); text != "" {
			ocrByPage[page.PageNumber] = text
		}
	}

	if ocrErr != nil && len(ocrByPage) == 0 {
		if plainText != "" {
			return ExtractionResult{
				Text:   CleanReadableContent(plainText),
				Source: "text",
				Pages:  plainPages,
			}, nil
		}
		return ExtractionResult{}, fmt.Errorf("OCR fallback failed: %w", ocrErr)
	}

	pages, usedPlain, usedOCR := mergePageTexts(plainByPage, ocrByPage)
	if len(pages) == 0 {
		return ExtractionResult{}, fmt.Errorf("no text extracted from document")
	}

	source := "text"
	switch {
	case usedPlain && usedOCR:
		source = "mixed"
	case usedOCR:
		source = "ocr"
	}

	return ExtractionResult{
		Text:   strings.TrimSpace(joinPageTexts(pages)),
		Source: source,
		Pages:  pages,
	}, nil
}

// mergePageTexts combines the plain text layer and OCR output page by page so
// that no page's content is dropped. For pages where both sources have text the
// better/combined text is kept; otherwise whichever source produced text wins.
func mergePageTexts(plainByPage, ocrByPage map[int]string) (pages []PageText, usedPlain, usedOCR bool) {
	pageNumbers := unionPageNumbers(plainByPage, ocrByPage)

	for _, pageNumber := range pageNumbers {
		plain := plainByPage[pageNumber]
		ocr := ocrByPage[pageNumber]

		var text string
		switch {
		case plain != "" && ocr != "":
			text = pickBetterExtractedText(plain, ocr)
			usedPlain = true
			usedOCR = true
		case ocr != "":
			text = ocr
			usedOCR = true
		default:
			text = plain
			usedPlain = true
		}

		text = CleanReadableContent(text)
		if text != "" {
			pages = append(pages, PageText{PageNumber: pageNumber, Text: text})
		}
	}

	return pages, usedPlain, usedOCR
}

func unionPageNumbers(a, b map[int]string) []int {
	set := make(map[int]struct{}, len(a)+len(b))
	for pageNumber := range a {
		set[pageNumber] = struct{}{}
	}
	for pageNumber := range b {
		set[pageNumber] = struct{}{}
	}

	numbers := make([]int, 0, len(set))
	for pageNumber := range set {
		numbers = append(numbers, pageNumber)
	}
	sort.Ints(numbers)
	return numbers
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
