package document

import (
	"regexp"
	"strings"
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

func CleanReadableContent(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}

	text = garbledEquationPattern.ReplaceAllString(text, " [rumus/simbol tidak terbaca] ")

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(cleanMultipleSpaces(line))
		if line == "" {
			continue
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

	return cleanMultipleSpaces(result)
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
