package document

import "strings"

const maxHistoryForSearch = 6
const maxAssistantContextChars = 500

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
		"solusi", "mengatasi", "menurut", "penulis", "peneliti",
		"bagaimana", "mengapa", "hal yang", "apa hal", "masalah tersebut",
		"masalah itu", "topik tersebut", "topik itu", "jurnal tersebut", "jurnal itu",
	}
	for _, hint := range followUpHints {
		if strings.Contains(q, hint) {
			return true
		}
	}

	prefixes := []string{"lalu ", "kemudian ", "terus ", "maka ", "dan "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(q, prefix) {
			return true
		}
	}

	return false
}

func truncateForSearch(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
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

	if len(userMessages) > 2 {
		userMessages = userMessages[len(userMessages)-2:]
	}

	var builder strings.Builder
	for _, message := range userMessages {
		builder.WriteString(message)
		builder.WriteString("\n")
	}

	// Sertakan jawaban asisten terakhir agar kata "tersebut"/"masalah itu" ter-resolve.
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			builder.WriteString(truncateForSearch(history[i].Content, maxAssistantContextChars))
			builder.WriteString("\n")
			break
		}
	}

	builder.WriteString(query)

	return strings.TrimSpace(builder.String())
}
