package cmd

import (
	"fmt"
	"io"
	"os"

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
		fmt.Fprintf(shapeWarningWriter, "Warning: unknown AWS shape %s/%s: %v\n", group, title, err)
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
