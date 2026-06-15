package document

import (
	"regexp"
	"strings"
	"unicode"
)

var searchStopWords = map[string]struct{}{
	"apa": {}, "yang": {}, "dan": {}, "di": {}, "ke": {}, "dari": {}, "untuk": {},
	"dengan": {}, "pada": {}, "adalah": {}, "ini": {}, "itu": {}, "saya": {}, "anda": {},
	"tolong": {}, "bisa": {}, "berapa": {}, "bagaimana": {}, "mengapa": {}, "kenapa": {},
	"the": {}, "a": {}, "an": {}, "is": {}, "are": {}, "of": {}, "to": {}, "in": {},
}

var yearPattern = regexp.MustCompile(`\b(19|20)\d{2}(/\d{4})?\b`)

func extractSearchTerms(query string) []string {
	seen := make(map[string]struct{})
	terms := make([]string, 0, 8)

	addTerm := func(term string) {
		term = strings.TrimSpace(strings.ToLower(term))
		if len(term) < 2 {
			return
		}
		if _, skip := searchStopWords[term]; skip {
			return
		}
		if _, exists := seen[term]; exists {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}

	for _, match := range yearPattern.FindAllString(query, -1) {
		addTerm(match)
	}

	var current strings.Builder
	flushWord := func() {
		if current.Len() == 0 {
			return
		}
		addTerm(current.String())
		current.Reset()
	}

	for _, r := range strings.ToLower(query) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' {
			current.WriteRune(r)
			continue
		}
		flushWord()
	}
	flushWord()

	return terms
}
