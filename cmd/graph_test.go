package cmd

import (
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/stretchr/testify/assert"
)

// TestGraphEdges pins v1 OutputArray.splitFromToValues semantics (go-output
// v1.4.0 output.go:298-311) for the v2 migration:
//
//   - the to-cell is stringified with v1's toString, which joined []string
//     cells using the dot-format separator "," (outputsettings.go GetSeparator)
//   - the stringified to-cell is then split on ",", producing one edge per
//     element — so a []string element that itself contains a comma splits
//     further (join-then-split, exactly as v1 did)
//   - empty targets produce no edge (v1 toDot/toMermaid skipped To == "")
//   - the from-value is stringified with the same rules
//
// One deliberate deviation: v1 stringified a missing/nil cell as
// "%!s(<nil>)" (fmt artifact), emitting a bogus node; nil cells here
// stringify to "" and are skipped as empty targets.
func TestGraphEdges(t *testing.T) {
	tests := map[string]struct {
		rows    []map[string]any
		fromCol string
		toCol   string
		want    []output.Edge
	}{
		"string slice to-cell produces one edge per element": {
			rows: []map[string]any{
				{"Name": "mesh-node", "Endpoints": []string{"backend-a", "backend-b"}},
			},
			fromCol: "Name",
			toCol:   "Endpoints",
			want: []output.Edge{
				{From: "mesh-node", To: "backend-a"},
				{From: "mesh-node", To: "backend-b"},
			},
		},
		"comma-joined string to-cell splits": {
			rows: []map[string]any{
				{"ID": "vpc-1", "PeeringIDs": "pcx-1,pcx-2"},
			},
			fromCol: "ID",
			toCol:   "PeeringIDs",
			want: []output.Edge{
				{From: "vpc-1", To: "pcx-1"},
				{From: "vpc-1", To: "pcx-2"},
			},
		},
		"slice element containing a comma splits again (v1 join-then-split)": {
			// v1 joined []string cells with "," and split the result on ",",
			// so {"x,y", "z"} yields three edges, not two.
			rows: []map[string]any{
				{"Name": "a", "To": []string{"x,y", "z"}},
			},
			fromCol: "Name",
			toCol:   "To",
			want: []output.Edge{
				{From: "a", To: "x"},
				{From: "a", To: "y"},
				{From: "a", To: "z"},
			},
		},
		"empty string to-cell produces no edges": {
			rows: []map[string]any{
				{"Name": "lonely", "To": ""},
			},
			fromCol: "Name",
			toCol:   "To",
			want:    []output.Edge{},
		},
		"empty slice to-cell produces no edges": {
			rows: []map[string]any{
				{"Name": "lonely", "To": []string{}},
			},
			fromCol: "Name",
			toCol:   "To",
			want:    []output.Edge{},
		},
		"missing to-cell produces no edges": {
			rows: []map[string]any{
				{"Name": "lonely"},
			},
			fromCol: "Name",
			toCol:   "To",
			want:    []output.Edge{},
		},
		"empty elements within to-cell are skipped": {
			rows: []map[string]any{
				{"Name": "a", "To": "x,,y"},
			},
			fromCol: "Name",
			toCol:   "To",
			want: []output.Edge{
				{From: "a", To: "x"},
				{From: "a", To: "y"},
			},
		},
		"non-string from-value is stringified": {
			rows: []map[string]any{
				{"ID": 42, "To": "x"},
			},
			fromCol: "ID",
			toCol:   "To",
			want: []output.Edge{
				{From: "42", To: "x"},
			},
		},
		"string slice from-value joins like v1 (no splitting of from)": {
			rows: []map[string]any{
				{"ID": []string{"a", "b"}, "To": "x"},
			},
			fromCol: "ID",
			toCol:   "To",
			want: []output.Edge{
				{From: "a,b", To: "x"},
			},
		},
		"multiple rows preserve row then element order": {
			rows: []map[string]any{
				{"Name": "first", "To": []string{"t1", "t2"}},
				{"Name": "second", "To": "t3"},
			},
			fromCol: "Name",
			toCol:   "To",
			want: []output.Edge{
				{From: "first", To: "t1"},
				{From: "first", To: "t2"},
				{From: "second", To: "t3"},
			},
		},
		"no rows produce no edges": {
			rows:    []map[string]any{},
			fromCol: "Name",
			toCol:   "To",
			want:    []output.Edge{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := graphEdges(tc.rows, tc.fromCol, tc.toCol)
			if len(tc.want) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
