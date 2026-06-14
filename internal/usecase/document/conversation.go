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

func buildSearchQuery(query string, history []ChatMessage) string {
	query = strings.TrimSpace(query)
	if len(history) == 0 {
		return query
	}

	start := 0
	if len(history) > maxHistoryForSearch {
		start = len(history) - maxHistoryForSearch
	}

	var builder strings.Builder
	for _, message := range history[start:] {
		builder.WriteString(message.Content)
		builder.WriteString("\n")
	}
	builder.WriteString(query)

	return strings.TrimSpace(builder.String())
}
