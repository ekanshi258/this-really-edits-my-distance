package main

// levenshtein returns the standard edit distance between two strings
// (insertions, deletions, substitutions each cost 1).
//
// Uses two-row DP so memory is O(min(len(a), len(b))) instead of O(m*n).
//
// Note: swap to damerauLevenshtein below if you want transpositions
// ("teh" → "the" = 1 instead of 2). The rest of the code is identical.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Keep `a` as the shorter string to minimise the row allocation.
	if len(a) > len(b) {
		a, b = b, a
	}

	la, lb := len(a), len(b)
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(
					prev[j-1], // substitute
					curr[j-1], // insert
					prev[j],   // delete
				)
			}
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// damerauLevenshtein adds transpositions (swap adjacent chars) to the
// standard Levenshtein operations. Uncomment its call in main() to use it.
func damerauLevenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Full m×n matrix needed for Damerau (previous-previous row required).
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min3(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)
			// Transposition
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				if d[i-2][j-2]+cost < d[i][j] {
					d[i][j] = d[i-2][j-2] + cost
				}
			}
		}
	}

	return d[la][lb]
}
