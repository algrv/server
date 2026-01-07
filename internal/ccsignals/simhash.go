package ccsignals

import (
	"hash/fnv"
	"regexp"
	"strings"
	"unicode"
)

const (
	HashBits           = 64
	DefaultShingleSize = 3
)

// represents a 64-bit SimHash fingerprint
type Fingerprint uint64

// generates SimHash fingerprints from text content
type SimHasher struct {
	shingleSize int
}

// creates a new SimHasher with the given shingle size
func NewSimHasher(shingleSize int) *SimHasher {
	if shingleSize < 1 {
		shingleSize = DefaultShingleSize
	}

	return &SimHasher{shingleSize: shingleSize}
}

// generates a SimHash fingerprint from text content
func (s *SimHasher) Hash(content string) Fingerprint {
	normalized := normalizeText(content)

	words := tokenize(normalized)
	if len(words) == 0 {
		return 0
	}

	shingles := s.generateShingles(words)
	if len(shingles) == 0 {
		return 0
	}

	return computeSimHash(shingles)
}

func normalizeText(text string) string {
	text = strings.ToLower(text)

	var builder strings.Builder
	prevSpace := true

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			builder.WriteRune(' ')
			prevSpace = true
		}
	}

	return strings.TrimSpace(builder.String())
}

var wordSplitRegex = regexp.MustCompile(`\s+`)

func tokenize(text string) []string {
	if text == "" {
		return nil
	}

	return wordSplitRegex.Split(text, -1)
}

func (s *SimHasher) generateShingles(words []string) []string {
	if len(words) < s.shingleSize {
		return []string{strings.Join(words, " ")}
	}

	shingles := make([]string, 0, len(words)-s.shingleSize+1)

	for i := 0; i <= len(words)-s.shingleSize; i++ {
		shingle := strings.Join(words[i:i+s.shingleSize], " ")
		shingles = append(shingles, shingle)
	}

	return shingles
}

func computeSimHash(shingles []string) Fingerprint {
	var v [HashBits]int

	for _, shingle := range shingles {
		hash := hashString(shingle)

		for i := range HashBits {
			bit := (hash >> i) & 1
			if bit == 1 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	var fingerprint Fingerprint
	for i := 0; i < HashBits; i++ {
		if v[i] > 0 {
			fingerprint |= (1 << i)
		}
	}

	return fingerprint
}

func hashString(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s)) // hash.Hash.Write never returns an error
	return h.Sum64()
}

// calculates the number of differing bits between two fingerprints
func HammingDistance(a, b Fingerprint) int {
	xor := a ^ b
	count := 0

	for xor != 0 {
		count++
		xor &= xor - 1
	}

	return count
}

// checks if two fingerprints are similar within the given threshold
func IsSimilar(a, b Fingerprint, threshold int) bool {
	return HammingDistance(a, b) <= threshold
}
