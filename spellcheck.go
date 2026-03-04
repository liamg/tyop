package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strconv"
	"strings"
)

//go:embed freq_en.txt.gz
var freqListGz []byte

//go:embed words_en.txt.gz
var wordListGz []byte

// shortWords are common words absent from the freq dict that must never be "corrected".
var shortWords = map[string]bool{
	"oh": true, "ah": true, "eh": true, "uh": true, "ok": true,
	"ow": true, "ew": true, "aw": true, "ay": true, "oy": true,
	"hi": true, "yo": true, "go": true, "no": true, "so": true,
	"my": true, "by": true, "he": true, "we": true, "me": true,
	"be": true, "do": true, "to": true, "of": true, "or": true,
	"if": true, "it": true, "in": true, "is": true, "at": true,
	"an": true, "as": true, "up": true, "us": true, "ox": true,
	"mm": true, "hm": true,
}


var knownContractions = map[string]bool{
	"don't": true, "won't": true, "can't": true, "it's": true,
	"i'm": true, "i've": true, "i'd": true, "i'll": true,
	"you're": true, "you've": true, "you'd": true, "you'll": true,
	"he's": true, "she's": true, "they're": true, "we're": true,
	"isn't": true, "aren't": true, "wasn't": true, "weren't": true,
	"didn't": true, "doesn't": true, "hadn't": true, "hasn't": true,
	"wouldn't": true, "couldn't": true, "shouldn't": true,
	"that's": true, "there's": true, "here's": true, "let's": true,
	"what's": true, "who's": true, "where's": true, "how's": true,
	"they've": true, "we've": true, "we'd": true, "they'd": true,
	"they'll": true, "we'll": true, "he'd": true, "she'd": true,
	"he'll": true, "she'll": true, "who'd": true, "who'll": true,
}

type SpellChecker struct {
	freq    map[string]int64 // 82k common words with frequency scores
	fullDict map[string]bool  // 370k word fallback
}

func loadSpellChecker() *SpellChecker {
	sc := &SpellChecker{
		freq:     make(map[string]int64, 90000),
		fullDict: make(map[string]bool, 380000),
	}

	// Load frequency dictionary.
	if gr, err := gzip.NewReader(bytes.NewReader(freqListGz)); err == nil {
		scanner := bufio.NewScanner(gr)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) == 2 {
				if n, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					sc.freq[strings.ToLower(parts[0])] = n
				}
			}
		}
		gr.Close()
	}

	// Load full word list.
	if gr, err := gzip.NewReader(bytes.NewReader(wordListGz)); err == nil {
		scanner := bufio.NewScanner(gr)
		for scanner.Scan() {
			if w := strings.TrimSpace(scanner.Text()); w != "" {
				sc.fullDict[strings.ToLower(w)] = true
			}
		}
		gr.Close()
	}

	return sc
}

// isValid reports whether word is a correctly-spelled English word
// (either common or in the full dictionary).
func (sc *SpellChecker) isValid(word string) bool {
	lower := strings.ToLower(word)
	_, inFreq := sc.freq[lower]
	return inFreq || shortWords[lower] || knownContractions[lower]
}

// correct returns the best correction for word, or word unchanged if no
// confident correction exists.
//
// splitCorrect checks whether inserting a space at any position produces two
// words that are both in the freq dict. Requires both parts to be ≥2 chars
// to avoid splitting into single letters. Returns "left right" or "".
func (sc *SpellChecker) splitCorrect(word string) string {
	for i := 2; i < len(word); i++ {
		left, right := word[:i], word[i:]
		if _, ok := sc.freq[left]; !ok {
			continue
		}
		if _, ok := sc.freq[right]; !ok {
			continue
		}
		return left + " " + right
	}
	return ""
}

// Priority order:
//  1. Transposes scored by freq dict (swap two adjacent chars — most common typo)
//  2. Deletions + insertions scored by freq dict together (missing/extra char)
//  3. ED2: all ED1 operations applied to each ED1 candidate, freq dict only
//  4. All ED1 against full dict — only if exactly one match
func (sc *SpellChecker) correct(word string) string {
	lower := strings.ToLower(word)
	if sc.isValid(lower) {
		return word
	}

	trans := sc.transposes(lower)
	ins := sc.insertions(lower)
	dels := sc.deletions(lower)
	ed1All := sc.dedupe(append(append(trans, ins...), dels...))

	if result := sc.bestFreqCandidate(trans); result != "" {
		return result
	}
	if result := sc.bestFreqCandidate(ins); result != "" {
		return result
	}
	if result := sc.bestFreqCandidate(dels); result != "" {
		return result
	}

	// ED2: prefer pure rearrangements (same chars, different order) over substitutions.
	ed2 := sc.ed2candidates(lower, ed1All)
	var ed2Rearrange, ed2Other []string
	for _, c := range ed2 {
		if sameChars(lower, c) {
			ed2Rearrange = append(ed2Rearrange, c)
		} else {
			ed2Other = append(ed2Other, c)
		}
	}
	if result := sc.bestFreqCandidate(ed2Rearrange); result != "" {
		return result
	}
	if result := sc.bestFreqCandidate(ed2Other); result != "" {
		return result
	}

	// Full-dict fallback for ED1: unambiguous only.
	if result := sc.bestCandidate(ed1All); result != "" {
		return result
	}
	return word
}

// bestFreqCandidate picks the highest-frequency candidate from the freq dict.
// Returns "" if no candidates are in the freq dict.
func (sc *SpellChecker) bestFreqCandidate(candidates []string) string {
	best, bestScore := "", int64(0)
	for _, c := range candidates {
		if s, ok := sc.freq[c]; ok && s > bestScore {
			bestScore, best = s, c
		}
	}
	return best
}

// bestCandidate tries freq dict first (highest frequency wins), then falls back
// to full dict requiring exactly one unambiguous match.
func (sc *SpellChecker) bestCandidate(candidates []string) string {
	if result := sc.bestFreqCandidate(candidates); result != "" {
		return result
	}
	// Full-dict fallback: unambiguous only.
	var fullCands []string
	for _, c := range candidates {
		if sc.fullDict[c] {
			fullCands = append(fullCands, c)
		}
	}
	if len(fullCands) == 1 {
		return fullCands[0]
	}
	return ""
}

// dedupe removes duplicate strings from a slice.
func (sc *SpellChecker) dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// transposes returns all single-adjacent-swap variants of word that are valid.
func (sc *SpellChecker) transposes(word string) []string {
	b := []byte(word)
	seen := make(map[string]bool)
	var out []string
	for i := 0; i < len(b)-1; i++ {
		b[i], b[i+1] = b[i+1], b[i]
		c := string(b)
		if !seen[c] && sc.isValid(c) {
			seen[c] = true
			out = append(out, c)
		}
		b[i], b[i+1] = b[i+1], b[i]
	}
	return out
}

// deletions returns all single-character-deletion variants of word that are valid.
func (sc *SpellChecker) deletions(word string) []string {
	seen := make(map[string]bool)
	var out []string
	for i := 0; i < len(word); i++ {
		c := word[:i] + word[i+1:]
		if !seen[c] && sc.isValid(c) {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// insertions returns all single-character-insertion variants of word that are valid.
func (sc *SpellChecker) insertions(word string) []string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	seen := make(map[string]bool)
	var out []string
	for i := 0; i <= len(word); i++ {
		for _, ch := range alphabet {
			c := word[:i] + string(ch) + word[i:]
			if !seen[c] && sc.isValid(c) {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// ed2candidates generates all valid freq-dict words reachable in exactly 2
// edit operations from original. Intermediate words need not be valid —
// only the final result is checked against the freq dict.
func (sc *SpellChecker) ed2candidates(original string, ed1Valid []string) []string {
	// Exclude the original and any ED1 hits already considered.
	seen := make(map[string]bool)
	seen[original] = true
	for _, w := range ed1Valid {
		seen[w] = true
	}

	var out []string
	add := func(c string) {
		if !seen[c] {
			seen[c] = true
			if _, ok := sc.freq[c]; ok {
				out = append(out, c)
			}
		}
	}

	// Iterate over ALL ED1 variants as intermediates (not just valid ones),
	// then apply ED1 again and check freq dict on the result.
	for _, mid := range sc.allED1(original) {
		b := []byte(mid)
		for i := 0; i < len(b)-1; i++ {
			b[i], b[i+1] = b[i+1], b[i]
			add(string(b))
			b[i], b[i+1] = b[i+1], b[i]
		}
		for i := 0; i < len(mid); i++ {
			add(mid[:i] + mid[i+1:])
		}
		const alphabet = "abcdefghijklmnopqrstuvwxyz"
		for i := 0; i <= len(mid); i++ {
			for _, ch := range alphabet {
				add(mid[:i] + string(ch) + mid[i:])
			}
		}
	}
	return out
}

// allED1 returns every string reachable from word in one edit (transpose,
// delete, or insert), regardless of whether it is a valid word.
func (sc *SpellChecker) allED1(word string) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(c string) {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	b := []byte(word)
	for i := 0; i < len(b)-1; i++ {
		b[i], b[i+1] = b[i+1], b[i]
		add(string(b))
		b[i], b[i+1] = b[i+1], b[i]
	}
	for i := 0; i < len(word); i++ {
		add(word[:i] + word[i+1:])
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i <= len(word); i++ {
		for _, ch := range alphabet {
			add(word[:i] + string(ch) + word[i:])
		}
	}
	return out
}

