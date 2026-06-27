package main

// GraphOutput is the JSON structure consumed by the frontend.
//
// Nodes: ordered word list — edges reference words by index, keeping the
//
//	JSON small (indices instead of repeated strings).
//
// Edges: each entry is [i, j] where i < j, both are indices into Nodes.
// Threshold: Max edit distance to create an edge (1 or 2)Max edit distance to create an edge (1 or 2)
type GraphOutput struct {
	Nodes     []string `json:"nodes"`
	Edges     [][2]int `json:"edges"`
	Threshold int      `json:"threshold"`
	Generated string   `json:"generated"`
}
