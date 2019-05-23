package helpers

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/emicklei/dot"

	"github.com/ArjenSchwarz/awstools/config"
)

// OutputHolder holds key-value pairs that belong together in the output
type OutputHolder struct {
	Contents map[string]string
}

// OutputArray holds all the different OutputHolders that will be provided as
// output, as well as the keys (headers) that will actually need to be printed
type OutputArray struct {
	Contents []OutputHolder
	Keys     []string
}

// Write will provide the output as configured in the configuration
func (output OutputArray) Write(settings config.Config) {
	switch strings.ToLower(*settings.OutputFormat) {
	case "csv":
		output.toCSV(*settings.OutputFile, "")
	case "dot":
		output.toDot(*settings.OutputFile)
	case "drawio":
		output.toDrawIO(*settings.OutputFile)
	default:
		output.toJSON(*settings.OutputFile)
	}
}

func (output OutputArray) toCSV(outputFile string, metadata string) {
	total := [][]string{}
	total = append(total, output.Keys)
	for _, holder := range output.Contents {
		values := make([]string, len(output.Keys))
		for counter, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = val
			}
		}
		total = append(total, values)
	}
	var target io.Writer
	if outputFile == "" {
		target = os.Stdout
	} else {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		target = bufio.NewWriter(file)
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s", metadata)
	buf.WriteTo(target)
	w := csv.NewWriter(target)

	for _, record := range total {
		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func (output OutputArray) toJSON(outputFile string) {
	total := make([]map[string]string, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]string)
		for _, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = val
			}
		}
		total = append(total, values)
	}
	buf := new(bytes.Buffer)
	responseString, _ := json.Marshal(total)
	fmt.Fprintf(buf, "%s", responseString)
	var target io.Writer
	if outputFile == "" {
		target = os.Stdout
	} else {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		target = bufio.NewWriter(file)
	}
	buf.WriteTo(target)
}

func (output OutputArray) toDot(outputFile string) {
	if len(output.Keys) != 2 {
		log.Fatal("You can only use DOT format when you only have To and From keys")
	}
	if !stringInSlice("To", output.Keys) {
		log.Fatal("You need a To key to use DOT format")
	}
	if !stringInSlice("From", output.Keys) {
		log.Fatal("You need a From key to use DOT format")
	}
	g := dot.NewGraph(dot.Directed)

	nodelist := make(map[string]dot.Node)

	// Step 1: Put all nodes in the list
	for _, holder := range output.Contents {
		if _, ok := nodelist[holder.Contents["From"]]; !ok {
			node := g.Node(holder.Contents["From"])
			nodelist[holder.Contents["From"]] = node
		}
	}

	// Step 2: Add all the edges/connections
	for _, holder := range output.Contents {
		if holder.Contents["To"] != "" {
			g.Edge(nodelist[holder.Contents["From"]], nodelist[holder.Contents["To"]])
		}
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s", g.String())
	var target io.Writer
	if outputFile == "" {
		target = os.Stdout
	} else {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		target = bufio.NewWriter(file)
	}
	buf.WriteTo(target)
}

func (output OutputArray) toDrawIO(outputFile string) {
	metadata := `# label: %Name%
# style: %Image%
# parentstyle: swimlane;whiteSpace=wrap;html=1;childLayout=stackLayout;horizontal=1;horizontalStack=0;resizeParent=1;resizeLast=0;collapsible=1;
# identity: -
# parent: -
# namespace: csvimport-
# connect: {"from": "Parent", "to": "Name", "invert": true, "label": "", \
#          "style": "curved=1;endArrow=blockThin;endFill=1;fontSize=11;"}
# left:
# top:
# width: 78
# height: 78
# padding: 0
# ignore: id,Image,fill,stroke
# link: url
# nodespacing: 40
# levelspacing: 100
# edgespacing: 40
# layout: auto
## ---- CSV below this line. First line are column names. ----
`
	output.toCSV(outputFile, metadata)
}

// AddHolder adds the provided OutputHolder to the OutputArray
func (output *OutputArray) AddHolder(holder OutputHolder) {
	var contents []OutputHolder
	if output.Contents != nil {
		contents = output.Contents
	}
	output.Contents = append(contents, holder)
}
