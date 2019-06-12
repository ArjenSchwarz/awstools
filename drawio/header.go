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
const defaultIgnore = "Image"
const headerComment = "##"
const defaultEdgeSpacing = 40
const defaultLevelSpacing = 100
const defaultNodeSpacing = 40
const defaultWidth = "auto"
const defaultHeight = "auto"
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
// label: Node label with placeholders and HTML.
// style: Node style (placeholders are replaced once).
// ignore: Comma-separated list of ignored columns for metadata. (These can be
// used for connections and styles but will not be added as metadata.)
func NewHeader(label string, style string, ignore string) Header {
	header := Header{
		label:  label,
		style:  style,
		ignore: ignore,
		layout: LayoutAuto,
	}
	header.SetSpacing(defaultNodeSpacing, defaultLevelSpacing, defaultEdgeSpacing)
	header.SetHeightAndWidth(defaultHeight, defaultWidth)
	return header
}

// DefaultHeader returns a header with: label: %Name%, style: %Image%, ignore: Image
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

// SetIdentity uses the given column name as the identity for cells (updates existing cells).
func (header *Header) SetIdentity(columnname string) {
	header.identity = columnname
}

// SetNamespace adds a prefix to the identity of cells to make sure they do not collide with existing cells (whose
// IDs are numbers from 0..n, sometimes with a GUID prefix in the context of realtime collaboration).
// This should be ignored by draw.io when identity is used, but regardless it is always ignored.
// Only left in here in case it is fixed.
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

// SetParent sets the parent information. Requires identity to be set as well
// parent: Uses the given column name as the parent reference for cells (refers to the identity column).
// Set to - to remove. Default is unused
// parentStyle: Parent style for nodes with child nodes (placeholders defined with %columnname% are replaced once).
// Only used when parent is defined. Default is value of DefaultParentStyle
func (header *Header) SetParent(parent, parentStyle string) {
	header.parent = parent
	header.parentStyle = parentStyle
}

// SetHeightAndWidth sets the height and width values of the nodes
// Possible values are a number (in px), auto or an @ sign followed by a column
// name that contains the value for the width. Default for both is auto.
func (header *Header) SetHeightAndWidth(height, width string) {
	header.height = height
	header.width = width
}

// SetPadding is for setting the padding when width and/or height are set to auto
func (header *Header) SetPadding(padding int) {
	header.padding = padding
}

// SetLeftAndTopColumns lets you set the column names storing x (left) and y (top) coordinates
// When using anything other than none for layout, this will be ignored
func (header *Header) SetLeftAndTopColumns(left, top string) {
	header.left = left
	header.top = top
}

// String returns a formatted string for the header
func (header *Header) String() string {
	headerArray := append(header.getLabel(), header.getStyle()...)
	headerArray = append(headerArray, header.getIdentity()...)
	headerArray = append(headerArray, header.getParent()...)
	headerArray = append(headerArray, header.getNamespace()...)
	headerArray = append(headerArray, header.connectionlist()...)
	headerArray = append(headerArray, header.getHeightAndWidth()...)
	headerArray = append(headerArray, header.getIgnore()...)
	headerArray = append(headerArray, header.getSpacing()...)
	headerArray = append(headerArray, header.getPadding()...)
	headerArray = append(headerArray, header.getLink()...)
	headerArray = append(headerArray, header.getLeftAndTop()...)
	headerArray = append(headerArray, header.getLayout()...)
	return strings.Join(headerArray, "\n") + "\n"
}

func (header *Header) connectionlist() []string {
	var result []string
	for _, connection := range header.connections {
		jsonstring, err := json.Marshal(connection)
		if err != nil {
			fmt.Println(err)
		}
		result = append(result, "# connect: "+string(jsonstring))
	}
	return result
}

func (header *Header) getLabel() []string {
	label := fmt.Sprintf("# label: %s", header.label)
	return []string{label}
}

func (header *Header) getStyle() []string {
	style := fmt.Sprintf("# style: %s", header.style)
	return []string{style}
}

func (header *Header) getIgnore() []string {
	ignore := fmt.Sprintf("# ignore: %s", header.ignore)
	return []string{ignore}
}

func (header *Header) getLayout() []string {
	layout := fmt.Sprintf("# layout: %s", header.layout)
	return []string{layout}
}

// SetLink sets the column to be renamed to link attribute (used as link)
func (header *Header) SetLink(columnname string) {
	header.link = columnname
}

func (header *Header) getLink() []string {
	if header.link != "" {
		link := fmt.Sprintf("# link: %s", header.link)
		return []string{link}
	}
	return []string{}
}

func (header *Header) getIdentity() []string {
	if header.identity != "" {
		identity := fmt.Sprintf("# identity: %s", header.identity)
		return []string{identity}
	}
	return []string{}
}

func (header *Header) getLeftAndTop() []string {
	if header.layout != LayoutNone {
		return []string{}
	}
	left := fmt.Sprintf("# left: %v", header.left)
	top := fmt.Sprintf("# top: %v", header.top)
	return []string{left, top}
}

func (header *Header) getNamespace() []string {
	if header.namespace != "" {
		namespace := fmt.Sprintf("# namespace: %s", header.namespace)
		return []string{namespace}
	}
	return []string{}
}

func (header *Header) getPadding() []string {
	if header.padding != 0 {
		padding := fmt.Sprintf("# padding: %v", header.padding)
		return []string{padding}
	}
	return []string{}
}

func (header *Header) getHeightAndWidth() []string {
	height := fmt.Sprintf("# height: %v", header.height)
	width := fmt.Sprintf("# width: %v", header.width)
	return []string{height, width}
}

func (header *Header) getParent() []string {
	if header.parent == "" || header.parent == "-" {
		return []string{}
	}
	parent := fmt.Sprintf("# parent: %s", header.parent)
	if header.parentStyle == "" {
		header.parentStyle = DefaultParentStyle
	}
	parentStyle := fmt.Sprintf("# parentstyle: %s", header.parentStyle)
	return []string{parent, parentStyle}
}

func (header *Header) getSpacing() []string {
	nodespacing := fmt.Sprintf("# nodespacing: %v", header.nodespacing)
	levelspacing := fmt.Sprintf("# levelspacing: %v", header.levelspacing)
	edgespacing := fmt.Sprintf("# edgespacing: %v", header.edgespacing)
	return []string{nodespacing, levelspacing, edgespacing}
}
