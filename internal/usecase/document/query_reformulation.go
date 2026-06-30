package document

import "strings"

// NeedsQueryReformulation skips LLM reformulation on the first message (saves
// latency) but keeps it for all follow-ups where conversation context matters.
func NeedsQueryReformulation(query string, history []ChatMessage) bool {
	return len(history) > 0 && strings.TrimSpace(query) != ""
}
