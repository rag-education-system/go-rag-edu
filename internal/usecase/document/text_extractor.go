package document

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	pages, err := te.extractWithPdftotext(data)
	if err == nil && len(pages) > 0 {
		return pages, nil
	}

	return te.extractWithGoLibrary(data)
}

func (te *TextExtractor) extractWithPdftotext(data []byte) ([]PageText, error) {
	tmpFile, err := os.CreateTemp("", "pdf-extract-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF for page count: %w", err)
	}
	numPages := reader.NumPage()

	var pages []PageText
	for i := 1; i <= numPages; i++ {
		text, err := te.extractPageWithPdftotext(tmpFile.Name(), i)
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

	if len(pages) == 0 {
		return nil, fmt.Errorf("no text extracted with pdftotext")
	}

	return pages, nil
}

func (te *TextExtractor) extractPageWithPdftotext(pdfPath string, pageNum int) (string, error) {
	pageStr := strconv.Itoa(pageNum)

	cmd := exec.Command("pdftotext",
		"-layout",
		"-f", pageStr,
		"-l", pageStr,
		pdfPath,
		"-",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

func (te *TextExtractor) extractWithGoLibrary(data []byte) ([]PageText, error) {
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
