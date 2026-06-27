// buildGraph: pre-computes a Levenshtein adjacency graph from a frequency-ranked
// word list and writes it as JSON ready for the frontend.
//
// Quick start (no flags needed — downloads word list automatically):
//
//	go run .
//
// Common overrides:
//
//	go run . -n 1000 -threshold 2
//	go run . -damerau -output dist/graph.json
//	go run . -word-list ./my-words.txt -pretty
//	go run . -no-cache          # force re-download
//
// All flags:
// -n 800              words from top of frequency list
// -threshold 1        edge distance (1 or 2)
// -min-len 3          min word length
// -max-len 8          max word length
// -damerau            transpositions cost 1 (default false)
// -output ../public/graph.json
// -word-list ./my.txt skip download, use local file
// -url <url>          override download source (default `google-10000-english“ on GitHub)
// -no-cache           force re-download
// -pretty             indented JSON for debugging (~3x larger, so default false)
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultWordListURL = "https://raw.githubusercontent.com/first20hours/google-10000-english/master/google-10000-english-no-swears.txt"
	defaultCachePath   = ".cache/words.txt"
)

func main() {
	// ── Flags ────────────────────────────────────────────────────────────────

	n := flag.Int("n", 800, "Words to include (taken from top of frequency list)")
	threshold := flag.Int("threshold", 1, "Max edit distance to create an edge — 1 (sparse) or 2 (dense)")
	minLen := flag.Int("min-len", 3, "Minimum word length (inclusive)")
	maxLen := flag.Int("max-len", 8, "Maximum word length (inclusive)")
	damerau := flag.Bool("damerau", false, "Use Damerau-Levenshtein: transpositions cost 1 (e.g. \"teh\"→\"the\" = 1, not 2)")
	output := flag.String("output", "../public/graph.json", "Output JSON file path (directory is created if needed)")
	wordList := flag.String("word-list", "", "Local word list file (one word per line, most-common first). Skips download when set.")
	url := flag.String("url", defaultWordListURL, "Word list URL — used when -word-list is empty")
	noCache := flag.Bool("no-cache", false, "Force re-download even if a cached word list exists")
	pretty := flag.Bool("pretty", false, "Indent JSON output (easier to debug, ~3× larger)")

	flag.Parse()

	// ── Resolve word list ─────────────────────────────────────────────────────

	listPath := *wordList
	if listPath == "" {
		listPath = ensureWordList(*url, defaultCachePath, *noCache)
	}

	// ── Load & filter ─────────────────────────────────────────────────────────

	words := loadWords(listPath, *n, *minLen, *maxLen)
	fmt.Printf("Loaded %d words  (length %d–%d)\n", len(words), *minLen, *maxLen)

	// ── Choose distance function ──────────────────────────────────────────────

	distFn := levenshtein
	algo := "Levenshtein"
	if *damerau {
		distFn = damerauLevenshtein
		algo = "Damerau-Levenshtein"
	}
	fmt.Printf("Algorithm:  %s  |  threshold: %d\n\n", algo, *threshold)

	// ── Compute edges ─────────────────────────────────────────────────────────

	edges := computeEdges(words, *threshold, distFn)

	// ── Write JSON ────────────────────────────────────────────────────────────

	writeOutput(*output, words, edges, *threshold, *pretty)

	// ── Summary ───────────────────────────────────────────────────────────────

	printStats(words, edges)
}

// ensureWordList returns a path to a local copy of the word list,
// downloading and caching it first if necessary.
func ensureWordList(rawURL, cachePath string, force bool) string {
	if !force {
		if _, err := os.Stat(cachePath); err == nil {
			fmt.Printf("Using cached word list → %s\n  (pass -no-cache to refresh)\n\n", cachePath)
			return cachePath
		}
	}

	fmt.Printf("Downloading word list...\n  %s\n", rawURL)

	if dir := filepath.Dir(cachePath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("Cannot create cache dir %s: %v", dir, err)
		}
	}

	resp, err := http.Get(rawURL) //nolint:noctx // simple CLI tool, no context needed
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Download failed: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	f, err := os.Create(cachePath)
	if err != nil {
		log.Fatalf("Cannot create %s: %v", cachePath, err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}

	fmt.Printf("Saved %.1f KB → %s\n\n", float64(written)/1024, cachePath)
	return cachePath
}

// loadWords reads up to n qualifying words from path.
// Words are taken in file order (preserving frequency rank).
func loadWords(path string, n, minLen, maxLen int) []string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Cannot open word list %s: %v", path, err)
	}
	defer f.Close()

	seen := make(map[string]bool)
	var words []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(words) < n {
		w := strings.ToLower(strings.TrimSpace(scanner.Text()))
		l := len(w)
		if l >= minLen && l <= maxLen && isAlpha(w) && !seen[w] {
			seen[w] = true
			words = append(words, w)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Read error: %v", err)
	}
	return words
}

// computeEdges finds all word pairs whose distance is within threshold.
func computeEdges(words []string, threshold int, distFn func(string, string) int) [][2]int {
	n := len(words)
	var edges [][2]int
	start := time.Now()

	for i := 0; i < n; i++ {
		if i > 0 && i%100 == 0 {
			elapsed := time.Since(start).Round(time.Millisecond)
			pct := float64(i) / float64(n) * 100
			fmt.Printf("  %d / %d  (%.0f%%)  edges so far: %d  [%s]\n",
				i, n, pct, len(edges), elapsed)
		}
		for j := i + 1; j < n; j++ {
			// Length difference is a free lower bound on edit distance.
			// If it already exceeds the threshold, skip the DP entirely.
			if abs(len(words[i])-len(words[j])) > threshold {
				continue
			}
			d := distFn(words[i], words[j])
			if d > 0 && d <= threshold {
				edges = append(edges, [2]int{i, j})
			}
		}
	}

	fmt.Printf("\nComputed %d edges in %s\n",
		len(edges), time.Since(start).Round(time.Millisecond))
	return edges
}

// writeOutput marshals the graph to JSON and writes it to path.
// The output directory is created if it does not exist.
func writeOutput(path string, words []string, edges [][2]int, threshold int, pretty bool) {
	out := GraphOutput{
		Nodes:     words,
		Edges:     edges,
		Threshold: threshold,
		Generated: time.Now().UTC().Format(time.RFC3339),
	}

	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(out, "", "  ")
	} else {
		data, err = json.Marshal(out)
	}
	if err != nil {
		log.Fatalf("JSON marshal error: %v", err)
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("Cannot create output dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Fatalf("Write error: %v", err)
	}

	fmt.Printf("Written → %s  (%.1f KB)\n", path, float64(len(data))/1024)
}

// printStats prints a brief summary of the resulting graph.
func printStats(words []string, edges [][2]int) {
	n := len(words)
	deg := make([]int, n)
	for _, e := range edges {
		deg[e[0]]++
		deg[e[1]]++
	}

	isolated, maxDeg, maxIdx := 0, 0, 0
	for i, d := range deg {
		if d == 0 {
			isolated++
		}
		if d > maxDeg {
			maxDeg = d
			maxIdx = i
		}
	}

	fmt.Printf("\n── Graph stats ──────────────────────\n")
	fmt.Printf("  Words:           %d\n", n)
	fmt.Printf("  Edges:           %d\n", len(edges))
	fmt.Printf("  Avg degree:      %.1f\n", float64(len(edges)*2)/float64(n))
	fmt.Printf("  Isolated nodes:  %d\n", isolated)
	fmt.Printf("  Most connected:  %q (%d neighbours)\n", words[maxIdx], maxDeg)
}
