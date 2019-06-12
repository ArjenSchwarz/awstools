package drawio

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DefaultConnectionStyle is the default style for connecting nodes
const DefaultConnectionStyle = "curved=1;endArrow=blockThin;endFill=1;fontSize=11;"

// DefaultParentStyle is the default style for parent-child relationships
const DefaultParentStyle = "swimlane;whiteSpace=wrap;html=1;childLayout=stackLayout;horizontal=1;horizontalStack=0;resizeParent=1;resizeLast=0;collapsible=1;"

const defaultLabel = "%Name%"
const defaultStyle = "%Image%"
const defaultIgnore = "Id"
const headerComment = "##"
const defaultEdgeSpacing = 40
const defaultLevelSpacing = 100
const defaultNodeSpacing = 40
const defaultWidth = "78"
const defaultHeight = "78"
const defaultNamespace = "csvimport-"

// Standard layout styles for draw.io
const (
	LayoutAuto           = "auto"
	LayoutNone           = "none"
	LayoutHorizontalFlow = "horizontalflow"
	LayoutVerticalFlow   = "verticalflow"
	LayoutHorizontalTree = "horizontaltree"
	LayoutVerticalTree   = "verticaltree"
	LayoutOrganic        = "organic"
	LayoutCircle         = "circle"
)

const basicHeader = `# label: %s
# style: %s
# identity: %s
## Parent
%s
# namespace: %s
%s
# height: %s
# width: %s
# ignore: %s
## Connectionlist
%s
## Spacing
%s
## Padding
%s
## Left and Top (only if layout is none)
%s
# layout: %s
## ---- CSV below this line. First line are column names. ----
`

// Connection is a representation Draw.IO CSV import connection value
type Connection struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Invert bool   `json:"invert"`
	Label  string `json:"label"`
	Style  string `json:"style"`
}

// Header is a representation of the Draw.IO CSV import header
type Header struct {
	label        string
	style        string
	ignore       string
	connections  []Connection
	link         string
	layout       string
	nodespacing  int
	levelspacing int
	edgespacing  int
	parent       string
	parentStyle  string
	height       string
	width        string
	padding      int
	left         string
	top          string
	identity     string
	namespace    string
}

// NewHeader returns a header with the provided label, style, and ignore
func NewHeader(label string, style string, ignore string) Header {
	header := Header{
		label:  label,
		style:  style,
		ignore: ignore,
		layout: LayoutAuto,
		link:   headerComment,
	}
	header.SetSpacing(defaultNodeSpacing, defaultLevelSpacing, defaultEdgeSpacing)
	header.SetHeightAndWidth(defaultHeight, defaultWidth)
	return header
}

// DefaultHeader returns a header with: label: %Name%, style: %Image%, ignore: id
func DefaultHeader() Header {
	return NewHeader(defaultLabel, defaultStyle, defaultIgnore)
}

// NewConnection creates a new connection object using default values
func NewConnection() Connection {
	return Connection{
		From:   "Parent",
		To:     "Name",
		Invert: true,
		Style:  DefaultConnectionStyle,
	}
}

// AddConnection adds a connection object to the header
func (header *Header) AddConnection(connection Connection) {
	var connections []Connection
	if header.connections != nil {
		connections = header.connections
	}
	header.connections = append(connections, connection)
}

// SetLayout sets the layout
func (header *Header) SetLayout(layout string) {
	header.layout = layout
}

// SetLink sets the column to be renamed to link attribute (used as link)
func (header *Header) SetLink(columnname string) {
	header.link = "# link: " + columnname
}

// SetIdentity uses the given column name as the identity for cells (updates existing cells).
func (header *Header) SetIdentity(columnname string) {
	header.identity = columnname
}

// SetNamespace adds a prefix to the identity of cells to make sure they do not collide with existing cells (whose
// IDs are numbers from 0..n, sometimes with a GUID prefix in the context of realtime collaboration).
// Default is csvimport-
func (header *Header) SetNamespace(namespace string) {
	header.namespace = namespace
}

// SetSpacing sets the spacing between the nodes at different levels
// nodespacing: Spacing between nodes. Default is 40.
// levelspacing: Spacing between levels of hierarchical layouts. Default is 100.
// edgespacing: Spacing between parallel edges. Default is 40. Use 0 to disable.
func (header *Header) SetSpacing(nodespacing, levelspacing, edgespacing int) {
	header.nodespacing = nodespacing
	header.levelspacing = levelspacing
	header.edgespacing = edgespacing
}

// GetSpacing retrieves the current spacing values
// nodespacing: Spacing between nodes
// levelspacing: Spacing between levels of hierarchical layouts
// edgespacing: Spacing between parallel edges
func (header *Header) GetSpacing() (nodespacing, levelspacing, edgespacing int) {
	nodespacing = header.nodespacing
	levelspacing = header.levelspacing
	edgespacing = header.edgespacing
	return
}

// getSpacingString retrieves the string version of the spacing block for the header output
func (header *Header) getSpacingString() string {
	return fmt.Sprintf(
		`# nodespacing: %v
# levelspacing: %v
# edgespacing: %v`,
		header.nodespacing,
		header.levelspacing,
		header.edgespacing,
	)
}

// SetParent sets the parent information. Requires identity to be set as well
// parent: Uses the given column name as the parent reference for cells (refers to the identity column).
// Set to - to remove. Default is unused
// parentStyle: Parent style for nodes with child nodes (placeholders defined with %columnname% are replaced once).
func (header *Header) SetParent(parent, parentStyle string) {
	header.parent = parent
	header.parentStyle = parentStyle
}

// getParentBlock retrieves a string for the header containing both the parent and parentstyle values
func (header *Header) getParentBlock() string {
	if header.parent != "" {
		return fmt.Sprintf(`# parent: %s
# parentStyle: %s`,
			header.parent,
			header.parentStyle,
		)
	}
	return headerComment
}

// SetHeightAndWidth sets the height and width values of the nodes
// Possible values are a number (in px), auto or an @ sign followed by a column
// name that contains the value for the width. Default for both is 78.
func (header *Header) SetHeightAndWidth(height, width string) {
	header.height = height
	header.width = width
}

// SetPadding is for setting the padding when width and/or height are set to auto
func (header *Header) SetPadding(padding int) {
	header.padding = padding
}

// getPaddingString returns a string for the header for the padding
func (header *Header) getPaddingString() string {
	if header.padding != 0 {
		return fmt.Sprintf("# padding: %v", header.padding)
	}
	return headerComment
}

// SetLeftAndTopColumns lets you set the column names storing x (left) and y (top) coordinates
// When using anything other than none for layout, this will be ignored
func (header *Header) SetLeftAndTopColumns(left, top string) {
	header.left = left
	header.top = top
}

// getLeftAndTopString returns the left and top values for the header
func (header *Header) getLeftAndTopString() string {
	if header.layout != LayoutNone {
		return headerComment
	}
	return fmt.Sprintf(`# left: %v
# top: %v`,
		header.left,
		header.top,
	)
}

// String returns a formatted string for the header
func (header *Header) String() string {
	return fmt.Sprintf(
		basicHeader,
		header.label,
		header.style,
		header.identity,
		header.getParentBlock(),
		header.namespace,
		header.connectionlist(),
		header.height,
		header.width,
		header.ignore,
		header.getSpacingString(),
		header.getPaddingString(),
		header.link,
		header.getLeftAndTopString(),
		header.layout,
	)
}

func (header *Header) connectionlist() string {
	var result []string
	for _, connection := range header.connections {
		jsonstring, err := json.Marshal(connection)
		if err != nil {
			fmt.Println(err)
		}
		result = append(result, "# connect: "+string(jsonstring))
	}
	return strings.Join(result, "\n")
}
