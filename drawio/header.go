package drawio

import (
	"encoding/json"
	"fmt"
	"strings"
)

const defaultConnectionStyle = "curved=1;endArrow=blockThin;endFill=1;fontSize=11;"
const defaultLabel = "%Name%"
const defaultStyle = "%Image%"
const defaultIgnore = "id"

const basicHeader = `# label: %s
# style: %s
# parentstyle: swimlane;whiteSpace=wrap;html=1;childLayout=stackLayout;horizontal=1;horizontalStack=0;resizeParent=1;resizeLast=0;collapsible=1;
# identity: -
# parent: -
# namespace: csvimport-
%s
# left:
# top:
# width: 78
# height: 78
# padding: 0
# ignore: %s
# link: url
# nodespacing: 40
# levelspacing: 100
# edgespacing: 40
# layout: auto
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
	label       string
	style       string
	ignore      string
	connections []Connection
}

// NewHeader returns a header with the provided label, style, and ignore
func NewHeader(label string, style string, ignore string) Header {
	return Header{
		label:  label,
		style:  style,
		ignore: ignore,
	}
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
		Style:  defaultConnectionStyle,
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

// String returns a formatted string for the header
func (header *Header) String() string {
	return fmt.Sprintf(basicHeader, header.label, header.style, header.connectionlist(), header.ignore)
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
