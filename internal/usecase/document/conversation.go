package document

import "strings"

const maxHistoryForSearch = 6

type ChatMessage struct {
	Role    string
	Content string
}

func normalizeHistory(history []ChatMessage) []ChatMessage {
	normalized := make([]ChatMessage, 0, len(history))
	for _, message := range history {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role != "user" && role != "assistant" {
			continue
		}

		normalized = append(normalized, ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	if len(normalized) > 20 {
		normalized = normalized[len(normalized)-20:]
	}

	return normalized
}

func needsHistoryContext(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	followUpHints := []string{
		"jelaskan", "poin", "maksud", "yang tadi", "tersebut", "lanjut",
		"ketiga", "kedua", "pertama", "keempat", "nomor", "pada bagian",
		"itu", "tersebutnya", "tadi", "sebelumnya",
	}
	for _, hint := range followUpHints {
		if strings.Contains(q, hint) {
			return true
		}
	}
	return false
}

func buildSearchQuery(query string, history []ChatMessage) string {
	query = strings.TrimSpace(query)
	if len(history) == 0 || !needsHistoryContext(query) {
		return query
	}

	start := 0
	if len(history) > maxHistoryForSearch {
		start = len(history) - maxHistoryForSearch
	}

	var userMessages []string
	for _, message := range history[start:] {
		if message.Role != "user" {
			continue
		}
		userMessages = append(userMessages, message.Content)
	}

	if len(userMessages) == 0 {
		return query
	}

	if len(userMessages) > 2 {
		userMessages = userMessages[len(userMessages)-2:]
	}

	var builder strings.Builder
	for _, message := range userMessages {
		builder.WriteString(message)
		builder.WriteString("\n")
	}
	builder.WriteString(query)

	return strings.TrimSpace(builder.String())
}
