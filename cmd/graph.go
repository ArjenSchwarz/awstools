package cmd

import (
	"fmt"
	"strconv"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// graphEdges builds graph (dot/mermaid) edges from table rows, replicating v1
// OutputArray.splitFromToValues: the to-cell is stringified ([]string cells
// are joined with ",", v1's dot-format separator), split on ",", and each
// non-empty element becomes one edge; the from-value is stringified the same
// way. Empty targets are skipped, matching v1's toDot/toMermaid edge loops.
func graphEdges(rows []map[string]any, fromCol, toCol string) []output.Edge {
	edges := make([]output.Edge, 0, len(rows))
	for _, row := range rows {
		from := graphCellString(row[fromCol])
		for target := range strings.SplitSeq(graphCellString(row[toCol]), ",") {
			if target == "" {
				continue
			}
			edges = append(edges, output.Edge{From: from, To: target})
		}
	}
	return edges
}

// graphCellString stringifies a cell value like v1's OutputArray.toString did
// under the dot format: []string joined with ",", ints via Itoa. One
// deliberate deviation: nil (missing cell) becomes "" instead of v1's
// "%!s(<nil>)" fmt artifact, so rows without a to-cell simply produce no
// edges. Bool cells never appear in from/to columns, so v1's Yes/No
// conversion is not replicated.
func graphCellString(value any) string {
	switch converted := value.(type) {
	case nil:
		return ""
	case string:
		return converted
	case []string:
		return strings.Join(converted, ",")
	case int:
		return strconv.Itoa(converted)
	default:
		return fmt.Sprint(converted)
	}
}
