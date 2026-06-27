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

	terms = expandProgramAcronyms(terms)
	terms = expandSolutionSynonyms(query, terms)

	return terms
}

var solutionQueryPattern = regexp.MustCompile(`(?i)\b(solusi|mengatasi|penanganan|upaya|tujuan|metode|rancang)\b`)

func expandSolutionSynonyms(query string, terms []string) []string {
	if !solutionQueryPattern.MatchString(query) {
		return terms
	}

	seen := make(map[string]struct{}, len(terms)+8)
	out := make([]string, 0, len(terms)+8)
	add := func(term string) {
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	for _, term := range terms {
		add(term)
	}
	for _, synonym := range []string{
		"solusi", "metode", "tujuan", "rancang", "implementasi", "sistem", "desain", "penelitian",
	} {
		add(synonym)
	}
	return out
}

var programAcronymExpansions = map[string][]string{
	"ptik": {"pendidikan teknik informatika", "teknik informatika"},
	"ptm":  {"pendidikan teknik mesin"},
	"ptb":  {"pendidikan teknik bangunan"},
}

func expandProgramAcronyms(terms []string) []string {
	seen := make(map[string]struct{}, len(terms)*3)
	out := make([]string, 0, len(terms)*2)
	add := func(term string) {
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	for _, term := range terms {
		add(term)
		if expansions, ok := programAcronymExpansions[term]; ok {
			for _, expansion := range expansions {
				add(expansion)
			}
		}
	}
	return out
}
