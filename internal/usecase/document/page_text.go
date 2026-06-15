package document

import "strings"

type PageText struct {
	PageNumber int
	Text       string
}

func joinPageTexts(pages []PageText) string {
	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		text := strings.TrimSpace(page.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func (c *Chunker) ChunkPages(pages []PageText) (contents []string, pageNumbers []int) {
	for _, page := range pages {
		text := strings.TrimSpace(CleanReadableContent(page.Text))
		if text == "" {
			continue
		}

		for _, chunk := range c.ChunkText(text) {
			contents = append(contents, chunk)
			pageNumbers = append(pageNumbers, page.PageNumber)
		}
	}

	return contents, pageNumbers
}
