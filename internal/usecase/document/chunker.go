package document

import (
	"strings"
	"unicode"
)

type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewChunker creates a new chunker
func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

var abbreviations = map[string]bool{
	"dr": true, "mr": true, "mrs": true, "ms": true, "prof": true,
	"sr": true, "jr": true, "inc": true, "ltd": true, "co": true,
	"vs": true, "etc": true, "vol": true, "no": true, "hal": true,
	"fig": true, "st": true, "ave": true, "dept": true, "univ": true,
	"jan": true, "feb": true, "mar": true, "apr": true, "jun": true,
	"jul": true, "aug": true, "sep": true, "oct": true, "nov": true, "dec": true,
	"e.g": true, "i.e": true, "p.s": true, "a.m": true, "p.m": true,
	"ml": true, "kg": true, "cm": true, "mm": true, "km": true,
}

func (c *Chunker) ChunkText(text string) []string {
	text = strings.TrimSpace(text)
	text = cleanText(text)

	if len(text) == 0 {
		return []string{}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + c.chunkSize
		if end > len(text) {
			end = len(text)
		}

		if end < len(text) {
			bestBreak := c.findBestBreakPoint(text, start, end)
			if bestBreak > start {
				end = bestBreak
			}
		}

		chunk := strings.TrimSpace(text[start:end])
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		newStart := c.findOverlapStart(text, end)
		if newStart <= start {
			newStart = start + 1
		}
		start = newStart

		if start >= len(text) {
			break
		}
	}

	return chunks
}

func (c *Chunker) findBestBreakPoint(text string, start, end int) int {
	minBreak := start + c.chunkSize/2

	for i := end; i > minBreak; i-- {
		if i >= len(text) {
			continue
		}

		char := text[i]
		if char == '\n' {
			return i + 1
		}

		if (char == '.' || char == '!' || char == '?') && c.isSentenceEnd(text, i) {
			return i + 1
		}
	}

	for i := end; i > minBreak; i-- {
		if i >= len(text) {
			continue
		}
		if text[i] == ';' || text[i] == ':' {
			return i + 1
		}
	}

	for i := end; i > minBreak; i-- {
		if i >= len(text) {
			continue
		}
		if text[i] == ',' {
			return i + 1
		}
	}

	return end
}

func (c *Chunker) isSentenceEnd(text string, dotIndex int) bool {
	if dotIndex+1 < len(text) {
		nextChar := text[dotIndex+1]
		if !unicode.IsSpace(rune(nextChar)) && nextChar != '"' && nextChar != '\'' && nextChar != ')' {
			return false
		}
	}

	wordStart := dotIndex - 1
	for wordStart >= 0 && (unicode.IsLetter(rune(text[wordStart])) || text[wordStart] == '.') {
		wordStart--
	}
	wordStart++

	if wordStart < dotIndex {
		word := strings.ToLower(text[wordStart:dotIndex])
		if abbreviations[word] {
			return false
		}
	}

	if dotIndex > 0 {
		prevChar := text[dotIndex-1]
		if unicode.IsDigit(rune(prevChar)) {
			if dotIndex+1 < len(text) && unicode.IsDigit(rune(text[dotIndex+1])) {
				return false
			}
		}
	}

	return true
}

func (c *Chunker) findOverlapStart(text string, end int) int {
	targetStart := end - c.chunkOverlap
	if targetStart <= 0 {
		return 0
	}

	for i := targetStart; i < end && i < len(text); i++ {
		if unicode.IsSpace(rune(text[i])) {
			return i + 1
		}
	}

	for i := targetStart; i > 0 && i > targetStart-50; i-- {
		if unicode.IsSpace(rune(text[i])) {
			return i + 1
		}
	}

	return targetStart
}

func cleanText(text string) string {
	var result strings.Builder
	prevSpace := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		} else {
			result.WriteRune(r)
			prevSpace = false
		}
	}

	return result.String()
}
