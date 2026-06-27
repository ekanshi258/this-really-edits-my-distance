package main

import "unicode"

// isAlpha returns true if every rune in s is a Unicode letter.
// Rejects words with numbers, hyphens, apostrophes, etc.
func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// There just isn't an abs() function in golang stdlib for ints i think :)
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
