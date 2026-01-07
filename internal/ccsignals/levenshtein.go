package ccsignals

// LevenshteinDistance calculates the edit distance between two strings
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)

	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}

	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0

			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// NormalizedEditDistance returns the edit distance as a ratio (0.0 to 1.0+)
func NormalizedEditDistance(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 0.0
	}

	distance := LevenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	return float64(distance) / float64(maxLen)
}
