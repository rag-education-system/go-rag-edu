package document

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var garbledEquationPattern = regexp.MustCompile(`(?i)(?:\b[a-z]\s*=\s*)?(?:\b[A-Za-z]\s+){4,}[-=+\s]*(?:\b[A-Za-z]\s+)*[-=]*`)

func pickBetterExtractedText(primary, secondary string) string {
	primary = strings.TrimSpace(primary)
	secondary = strings.TrimSpace(secondary)

	if primary == "" {
		return secondary
	}
	if secondary == "" {
		return primary
	}

	primaryRatio := garbledRatio(primary)
	secondaryRatio := garbledRatio(secondary)

	switch {
	case secondaryRatio+0.08 < primaryRatio:
		return mergeExtractedTexts(secondary, primary)
	case primaryRatio+0.08 < secondaryRatio:
		return mergeExtractedTexts(primary, secondary)
	default:
		return mergeExtractedTexts(primary, secondary)
	}
}

func mergeExtractedTexts(parts ...string) string {
	seen := make(map[string]struct{})
	merged := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, part)
	}

	return strings.Join(merged, "\n\n")
}

func SanitizeUTF8(text string) string {
	if utf8.ValidString(text) {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))

	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == utf8.RuneError && size == 1 {
			i++
			continue
		}
		result.WriteRune(r)
		i += size
	}

	return result.String()
}

func CleanReadableContent(text string) string {
	text = SanitizeUTF8(text)
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}

	text = garbledEquationPattern.ReplaceAllString(text, " [rumus/simbol tidak terbaca] ")

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if looksLikeTableRow(originalLine) {
			line = normalizeTableRow(line)
		} else {
			line = cleanMultipleSpaces(line)
		}

		if isGarbledText(line) {
			if readable := extractReadablePrefix(line); readable != "" {
				cleaned = append(cleaned, readable)
			}
			continue
		}

		cleaned = append(cleaned, line)
	}

	result := strings.TrimSpace(strings.Join(cleaned, "\n"))
	if result == "" {
		return text
	}

	return result
}

func looksLikeTableRow(line string) bool {
	spaceCount := 0
	consecutiveSpaces := 0
	maxConsecutive := 0

	for _, r := range line {
		if r == ' ' {
			spaceCount++
			consecutiveSpaces++
			if consecutiveSpaces > maxConsecutive {
				maxConsecutive = consecutiveSpaces
			}
		} else {
			consecutiveSpaces = 0
		}
	}

	return maxConsecutive >= 3 && spaceCount > len(line)/4
}

func extractReadablePrefix(line string) string {
	lower := strings.ToLower(line)
	keywords := []string{
		"menggunakan fitur equation",
		"fitur equation",
		"menggunakan fitur smartart",
		"fitur smartart",
		"buatlah rumus",
	}

	for _, keyword := range keywords {
		idx := strings.Index(lower, keyword)
		if idx < 0 {
			continue
		}

		end := idx + len(keyword)
		prefix := strings.TrimSpace(line[:end])
		if prefix == "" {
			continue
		}

		return prefix + ". [Contoh rumus/simbol di dokumen tidak terbaca setelah ekstraksi]"
	}

	return ""
}

func cleanMultipleSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func normalizeTableRow(line string) string {
	parts := strings.Fields(line)
	if len(parts) <= 1 {
		return strings.TrimSpace(line)
	}

	var result []string
	for i, part := range parts {
		result = append(result, part)
		if i < len(parts)-1 {
			nextPart := parts[i+1]
			if looksLikeValue(nextPart) && looksLikeLabel(part) {
				result = append(result, ":")
			}
		}
	}

	return strings.Join(result, " ")
}

func looksLikeLabel(s string) bool {
	s = strings.ToUpper(s)
	labels := []string{"IPK", "RATA", "NILAI", "JUMLAH", "TOTAL", "SKS", "INDEX", "PRESTASI", "SEMESTER", "TAHUN"}
	for _, label := range labels {
		if strings.Contains(s, label) {
			return true
		}
	}
	return false
}

func looksLikeValue(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	for _, r := range s {
		if r == '.' || r == ',' || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}
