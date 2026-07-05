package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/ArjenSchwarz/go-output/v2/icons"
)

// shapeWarningWriter receives awsShape's unknown-shape warnings. It defaults
// to stderr and is overridden in tests.
var shapeWarningWriter io.Writer = os.Stderr

// awsShape wraps icons.GetAWSShape with v1 drawio.AWSShape's lenient
// behavior: an unknown group/title yields an empty style instead of an error
// (R5.3). The swallowed error is reported as a stderr warning so shape-name
// typos stay visible.
func awsShape(group, title string) string {
	shape, err := icons.GetAWSShape(group, title)
	if err != nil {
		// Best-effort warning; nothing to do if stderr itself fails.
		_, _ = fmt.Fprintf(shapeWarningWriter, "Warning: unknown AWS shape %s/%s: %v\n", group, title, err)
		return ""
	}
	return shape
}

// drawIOConnection returns a connection with v1 drawio.NewConnection's
// defaults: from Parent to Name, inverted, default connection style.
func drawIOConnection() output.DrawIOConnection {
	return output.DrawIOConnection{
		From:   "Parent",
		To:     "Name",
		Invert: true,
		Style:  output.DrawIODefaultConnectionStyle,
	}
}

// drawIOBaseHeader returns the header shape shared by every command's
// create*DrawIOHeader: v1 drawio.NewHeader(label, style, ignore) followed by
// SetHeightAndWidth("78", "78"). v2's DefaultDrawIOHeader carries the same
// layout, spacing, and namespace defaults as v1's NewHeader.
func drawIOBaseHeader(label, style, ignore string) output.DrawIOHeader {
	header := output.DefaultDrawIOHeader()
	header.Label = label
	header.Style = style
	header.Ignore = ignore
	header.Height = "78"
	header.Width = "78"
	return header
}

// drawIORecords converts table rows into draw.io records. []string cells are
// joined with "," so the emitted CSV cells match v1's drawio writer output —
// the comma-separated multi-value refs that header connections and the
// combine read-back (tgw overview, vpc peerings) rely on. v2's drawio
// renderer stringifies cells with fmt.Sprint, which would emit "[a b c]" for
// a []string and break edge resolution. Generic over the map type so both
// []map[string]any and []output.Record rows can be passed directly.
func drawIORecords[M ~map[string]any](rows []M) []output.Record {
	records := make([]output.Record, 0, len(rows))
	for _, row := range rows {
		record := make(output.Record, len(row))
		for key, value := range row {
			if list, ok := value.([]string); ok {
				record[key] = strings.Join(list, ",")
				continue
			}
			record[key] = value
		}
		records = append(records, record)
	}
	return records
}

// drawIORecordString reads a cell from a parsed drawio record as a string.
// output.ParseDrawIOFile records carry every column with "" for empty cells,
// so cells are plain strings; missing keys also yield "".
func drawIORecordString(record output.Record, key string) string {
	value, _ := record[key].(string)
	return value
}
