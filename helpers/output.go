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
	case "drawio":
		if settings.OutputHeaders == nil {
			log.Fatal("This command doesn't currently support the drawio output format")
		}
		output.toCSV(*settings.OutputFile, *settings.OutputHeaders)
	case "dot":
		if settings.DotColumns == nil {
			log.Fatal("This command doesn't currently support the dot output format")
		}
		output.toDot(*settings.OutputFile, settings.DotColumns)
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

func (output OutputArray) toDot(outputFile string, columns *config.DotColumns) {
	type dotholder struct {
		To   string
		From string
	}
	// Create new lines using the dotcolumns, splitting up multi values
	cleanedlist := []dotholder{}
	for _, holder := range output.Contents {
		for _, tovalue := range strings.Split(holder.Contents[columns.To], ",") {
			dothold := dotholder{
				From: holder.Contents[columns.From],
				To:   tovalue,
			}
			cleanedlist = append(cleanedlist, dothold)
		}
	}

	g := dot.NewGraph(dot.Directed)

	nodelist := make(map[string]dot.Node)

	// Step 1: Put all nodes in the list
	for _, cleaned := range cleanedlist {
		if _, ok := nodelist[cleaned.From]; !ok {
			node := g.Node(cleaned.From)
			nodelist[cleaned.From] = node
		}
	}

	// Step 2: Add all the edges/connections
	for _, cleaned := range cleanedlist {
		if cleaned.To != "" {
			g.Edge(nodelist[cleaned.From], nodelist[cleaned.To])
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

// AddHolder adds the provided OutputHolder to the OutputArray
func (output *OutputArray) AddHolder(holder OutputHolder) {
	var contents []OutputHolder
	if output.Contents != nil {
		contents = output.Contents
	}
	output.Contents = append(contents, holder)
}
