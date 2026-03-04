package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Locale controls which spelling variants are preferred.
type Locale string

const (
	EnGB Locale = "en-gb"
	EnUS Locale = "en-us"
)

// Corrector fixes typos in text using a three-stage pipeline:
//  1. Semicolon preprocessing (don;t → don't when result is a known word)
//  2. Autocorrect map (contractions, known misspellings)
//  3. Spell checker (transpose/delete, unambiguous only, skips capitalised words)
//  4. Sentence capitalisation
type Corrector struct {
	locale      Locale
	autocorrect map[string]string
	spell       *SpellChecker
}

func newCorrector(locale Locale) *Corrector {
	return &Corrector{
		locale:      locale,
		autocorrect: buildAutocorrect(locale),
		spell:       loadSpellChecker(),
	}
}

// Correct fixes typos in text, preserving whitespace and punctuation.
func (c *Corrector) Correct(text string) string {
	text = c.preprocessSemicolons(text)
	text = fixPunctuation(text)
	tokens := tokenize(text)

	// Only apply sentence capitalisation if the text already contains at least
	// one capitalised word — avoids imposing caps on intentionally lowercase text.
	applyCapitalize := hasCapitalizedWord(tokens)

	atBoundary := true          // true for first word and after .!?
	capitalize := applyCapitalize // capitalise first word if applyCapitalize

	for i, tok := range tokens {
		if !isWordToken(tok) {
			if isSentenceEnd(tok) {
				atBoundary = true
				if applyCapitalize {
					capitalize = true
				}
			}
			continue
		}
		tokens[i] = c.correctWord(tok, capitalize, atBoundary, applyCapitalize)
		atBoundary = false
		capitalize = false
	}
	return strings.Join(tokens, "")
}

// hasCapitalizedWord reports whether any word token starts with an uppercase letter.
func hasCapitalizedWord(tokens []string) bool {
	for _, tok := range tokens {
		if isWordToken(tok) && startsWithUpper(tok) {
			return true
		}
	}
	return false
}


// preprocessSemicolons replaces ';' with '\'' between letters when the
// resulting word is a known valid word or contraction (e.g. don;t → don't).
// This avoids blindly replacing every semicolon between letters.
func (c *Corrector) preprocessSemicolons(text string) string {
	runes := []rune(text)
	out := make([]rune, len(runes))
	copy(out, runes)

	for i, r := range runes {
		if r != ';' || i == 0 || i == len(runes)-1 {
			continue
		}
		if !unicode.IsLetter(runes[i-1]) || !unicode.IsLetter(runes[i+1]) {
			continue
		}
		// Find the word boundaries around this ';'.
		start := i - 1
		for start > 0 && (unicode.IsLetter(runes[start-1]) || runes[start-1] == '\'') {
			start--
		}
		end := i + 1
		for end < len(runes) && (unicode.IsLetter(runes[end]) || runes[end] == '\'') {
			end++
		}
		// Build candidate word with ';' → '\''.
		candidate := make([]rune, end-start)
		copy(candidate, runes[start:end])
		candidate[i-start] = '\''
		if c.spell.isValid(string(candidate)) {
			out[i] = '\''
		}
	}
	return string(out)
}

func (c *Corrector) correctWord(word string, capitalize bool, atBoundary bool, applyCapitalize bool) string {
	lower := strings.ToLower(word)

	// Stage 1: autocorrect map.
	if replacement, ok := c.autocorrect[lower]; ok {
		if !applyCapitalize {
			replacement = lowercaseFirst(replacement)
		}
		if capitalize {
			return capitalizeFirst(replacement)
		}
		return replacement
	}

	// Stage 2: spell check.
	// Skip mid-sentence capitalised words — likely proper nouns.
	// At a sentence boundary (start of text or after .!?) the capital may be
	// grammatical, so we let the dictionary decide: proper nouns won't be in
	// the word lists and will simply return unchanged.
	isProperNoun := startsWithUpper(word) && !atBoundary
	if !isProperNoun && !c.spell.isValid(lower) {
		if split := c.spell.splitCorrect(lower); split != "" {
			if capitalize {
				return capitalizeFirst(split)
			}
			return split
		}
		if suggestion := c.spell.correct(lower); suggestion != lower {
			if capitalize {
				return capitalizeFirst(suggestion)
			}
			return suggestion
		}
	}

	// Stage 3: sentence capitalisation only.
	if capitalize {
		return capitalizeFirst(word)
	}
	return word
}

func startsWithUpper(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// tokenize splits text into alternating word / non-word tokens.
// Word characters are letters and apostrophes.
func tokenize(text string) []string {
	var tokens []string
	var cur strings.Builder
	inWord := false

	for _, r := range text {
		isWordRune := unicode.IsLetter(r) || r == '\''
		if isWordRune != inWord {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			inWord = isWordRune
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func isWordToken(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLetter(r)
}

func isSentenceEnd(s string) bool {
	return strings.ContainsAny(s, ".!?")
}

func capitalizeFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

func lowercaseFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}

// sameChars reports whether a and b contain the same multiset of runes.
func sameChars(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[rune]int, len(a))
	for _, r := range a {
		counts[r]++
	}
	for _, r := range b {
		counts[r]--
	}
	for _, v := range counts {
		if v != 0 {
			return false
		}
	}
	return true
}

// fixPunctuation fixes common punctuation spacing errors:
//   - removes space(s) before , ! ? ; : (e.g. "word ," → "word,")
//   - adds a space after , ! ? ; : when directly followed by a letter/digit
//     (e.g. "word,next" → "word, next")
func fixPunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)
	n := len(runes)
	for i := 0; i < n; i++ {
		r := runes[i]
		// Remove spaces immediately before these punctuation marks.
		if r == ' ' && i+1 < n && strings.ContainsRune(",!?;:", runes[i+1]) {
			continue
		}
		b.WriteRune(r)
		// Add a space after punctuation if directly followed by a letter or digit.
		if strings.ContainsRune(",!?;:", r) && i+1 < n && unicode.IsLetter(runes[i+1]) {
			b.WriteRune(' ')
		}
	}
	return b.String()
}
