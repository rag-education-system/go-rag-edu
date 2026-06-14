package document

import "strings"

const (
	greetingMaxLen  = 25
	greetingMaxDist = 2
)

var greetingWords = []string{"halo", "hai", "hello", "hi", "hey", "helo", "hallo"}
var greetingPhrases = []string{"selamat pagi", "selamat siang", "selamat malam", "selamat sore", "selamat"}

func isGreeting(message string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(message))
	trimmed = strings.Trim(trimmed, "!?.,")
	if trimmed == "" || len(trimmed) > greetingMaxLen {
		return false
	}

	for _, phrase := range greetingPhrases {
		if trimmed == phrase || strings.HasPrefix(trimmed, phrase+" ") {
			return true
		}
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return false
	}

	return matchesGreetingWord(words[0])
}

func matchesGreetingWord(word string) bool {
	for _, greeting := range greetingWords {
		if word == greeting {
			return true
		}
		// Fuzzy match only for longer greetings to avoid false positives like "this" -> "hi".
		if len(greeting) >= 4 && abs(len(word)-len(greeting)) <= 1 &&
			levenshtein(word, greeting) <= greetingMaxDist {
			return true
		}
	}
	return false
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				curr[j-1]+1,
				prev[j]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
