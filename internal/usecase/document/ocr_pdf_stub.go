//go:build !cgo || nocgo

package document

import "fmt"

// PDFOCRAvailable reports whether this binary can render PDF pages for OCR.
// Without CGO, go-fitz is excluded so Windows builds can start without
// libmupdf.dll / a C toolchain. Plain PDF text extraction still works;
// scanned PDFs need a CGO build (see scripts/setup-windows.ps1).
func PDFOCRAvailable() bool { return false }

func (o *OCRExtractor) extractPagesFromPDFSelective(
	_ []byte,
	_ func(pageNumber int) bool,
) ([]PageText, error) {
	return nil, fmt.Errorf(
		"PDF OCR requires a CGO-enabled build (install MinGW/gcc, then build with CGO_ENABLED=1)",
	)
}
