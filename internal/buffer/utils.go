package buffer

import "strings"

// caps input strings to prevent excessive memory usage
const MaxLevenshteinLength = 10000

// calculates the edit distance between two strings.
// uses space-optimized O(min(m,n)) memory instead of O(m*n).
// for very long strings, samples to estimate distance.
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// for very long strings, sample to estimate
	if len(s1) > MaxLevenshteinLength || len(s2) > MaxLevenshteinLength {
		return levenshteinSampled(s1, s2)
	}

	// ensure s1 is the shorter string for space optimization
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}

	// space-optimized: only keep two rows
	prev := make([]int, len(s1)+1)
	curr := make([]int, len(s1)+1)

	// initialize first row
	for i := range prev {
		prev[i] = i
	}

	// fill matrix row by row
	for j := 1; j <= len(s2); j++ {
		curr[0] = j

		for i := 1; i <= len(s1); i++ {
			cost := 0

			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			curr[i] = min(
				prev[i]+1,      // deletion
				curr[i-1]+1,    // insertion
				prev[i-1]+cost, // substitution
			)
		}

		prev, curr = curr, prev
	}

	return prev[len(s1)]
}

// levenshteinSampled estimates edit distance for very long strings
// by sampling chunks and extrapolating
func levenshteinSampled(s1, s2 string) int {
	// sample size for estimation
	const sampleSize = 5000

	// if lengths differ significantly, that's already a big edit
	lenDiff := len(s1) - len(s2)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}

	// sample from start, middle, and end
	sample1Start := s1[:min(sampleSize/3, len(s1))]
	sample2Start := s2[:min(sampleSize/3, len(s2))]

	midStart1 := max(0, len(s1)/2-sampleSize/6)
	midStart2 := max(0, len(s2)/2-sampleSize/6)
	sample1Mid := s1[midStart1:min(midStart1+sampleSize/3, len(s1))]
	sample2Mid := s2[midStart2:min(midStart2+sampleSize/3, len(s2))]

	endStart1 := max(0, len(s1)-sampleSize/3)
	endStart2 := max(0, len(s2)-sampleSize/3)
	sample1End := s1[endStart1:]
	sample2End := s2[endStart2:]

	// compute distance on samples
	distStart := levenshteinSmall(sample1Start, sample2Start)
	distMid := levenshteinSmall(sample1Mid, sample2Mid)
	distEnd := levenshteinSmall(sample1End, sample2End)

	// extrapolate: average sample distance * scale factor + length difference
	avgSampleLen := float64(len(sample1Start)+len(sample1Mid)+len(sample1End)) / 3
	avgDist := float64(distStart+distMid+distEnd) / 3
	maxLen := max(len(s1), len(s2))

	// scale the sample distance to full length estimate
	scaleFactor := float64(maxLen) / avgSampleLen / 3
	estimated := int(avgDist*scaleFactor) + lenDiff

	return estimated
}

// levenshteinSmall computes exact distance for small strings (used by sampling)
func levenshteinSmall(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}

	prev := make([]int, len(s1)+1)
	curr := make([]int, len(s1)+1)

	for i := range prev {
		prev[i] = i
	}

	for j := 1; j <= len(s2); j++ {
		curr[0] = j

		for i := 1; i <= len(s1); i++ {
			cost := 0

			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			curr[i] = min(
				prev[i]+1,
				curr[i-1]+1,
				prev[i-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(s1)]
}

// determines if a code update has a large delta (behavioral paste detection)
// this is server-side detection independent of frontend source field
func IsLargeDelta(previousCode, newCode string) bool {
	// check character delta
	deltaLen := len(newCode) - len(previousCode)
	if deltaLen >= PasteDeltaThreshold {
		return true
	}

	// check line delta
	newLines := strings.Count(newCode, "\n") - strings.Count(previousCode, "\n")
	return newLines >= PasteLineThreshold
}

// determines if edits are significant enough to unlock
func IsSignificantEdit(baseline, current string) bool {
	if baseline == "" {
		return true // no baseline means no lock
	}

	distance := LevenshteinDistance(baseline, current)
	baselineLen := len(baseline)

	if baselineLen == 0 {
		return true
	}

	normalized := float64(distance) / float64(baselineLen)
	return normalized >= UnlockThreshold
}
