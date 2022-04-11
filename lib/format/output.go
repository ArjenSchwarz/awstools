package format

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/emicklei/dot"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/templates"
)

// OutputHolder holds key-value pairs that belong together in the output
type OutputHolder struct {
	Contents map[string]interface{}
}

// OutputArray holds all the different OutputHolders that will be provided as
// output, as well as the keys (headers) that will actually need to be printed
type OutputArray struct {
	Settings *OutputSettings
	Contents []OutputHolder
	Keys     []string
}

// GetContentsMap returns a stringmap of the output contents
func (output OutputArray) GetContentsMap() []map[string]string {
	total := make([]map[string]string, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]string)
		for _, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	return total
}

// Write will provide the output as configured in the configuration
func (output OutputArray) Write(settings config.Config) {
	switch output.Settings.OutputFormat {
	case "csv":
		output.toCSV()
	case "html":
		output.toHTML()
	case "drawio":
		if !output.Settings.DrawIOHeader.IsSet() {
			log.Fatal("This command doesn't currently support the drawio output format")
		}
		drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), *settings.OutputFile)
	case "dot":
		if output.Settings.DotColumns == nil {
			log.Fatal("This command doesn't currently support the dot output format")
		}
		output.toDot()
	default:
		output.toJSON()
	}
}

func (output OutputArray) toCSV() {
	total := [][]string{}
	if !output.Settings.ShouldAppend {
		total = append(total, output.Keys)
	}
	for _, holder := range output.Contents {
		values := make([]string, len(output.Keys))
		for counter, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	var target io.Writer
	if output.Settings.OutputFile == "" {
		target = os.Stdout
	} else {
		if output.Settings.ShouldAppend {
			file, err := os.OpenFile(output.Settings.OutputFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			target = bufio.NewWriter(file)
		} else {
			file, err := os.Create(output.Settings.OutputFile)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			target = bufio.NewWriter(file)
		}

	}
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

func (output OutputArray) toJSON() {
	jsonString, _ := json.Marshal(output.GetContentsMap())

	err := PrintByteSlice(jsonString, output.Settings.OutputFile)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (output OutputArray) toDot() {
	type dotholder struct {
		To   string
		From string
	}
	// Create new lines using the dotcolumns, splitting up multi values
	cleanedlist := []dotholder{}
	for _, holder := range output.Contents {
		for _, tovalue := range strings.Split(output.toString(holder.Contents[output.Settings.DotColumns.To]), ",") {
			dothold := dotholder{
				From: output.toString(holder.Contents[output.Settings.DotColumns.From]),
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
	err := PrintByteSlice([]byte(g.String()), output.Settings.OutputFile)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (output OutputArray) toHTML() {
	t := template.New("table")
	t, _ = t.Parse(templates.HTMLTableTemplate)
	var baseTemplate string
	if output.Settings.ShouldAppend {
		originalfile, err := ioutil.ReadFile(output.Settings.OutputFile)
		if err != nil {
			panic(err)
		}
		baseTemplate = string(originalfile)
	} else {
		b := template.New("base")
		b, _ = b.Parse(templates.BaseHTMLTemplate)
		baseBuf := new(bytes.Buffer)
		b.Execute(baseBuf, output)
		baseTemplate = baseBuf.String()
	}
	tableBuf := new(bytes.Buffer)
	t.Execute(tableBuf, output)
	resultString := strings.Replace(baseTemplate, "<div id='end'></div>", tableBuf.String(), 1)
	err := PrintByteSlice([]byte(resultString), output.Settings.OutputFile)
	if err != nil {
		log.Fatal(err.Error())
	}
}

// PrintByteSlice prints the provided contents to stdout or the provided filepath
func PrintByteSlice(contents []byte, outputFile string) error {
	var target io.Writer
	var err error
	if outputFile == "" {
		target = os.Stdout
	} else {
		target, err = os.Create(outputFile)
		if err != nil {
			return err
		}
	}
	w := bufio.NewWriter(target)
	w.Write(contents)
	err = w.Flush()
	return err
}

// AddHolder adds the provided OutputHolder to the OutputArray
func (output *OutputArray) AddHolder(holder OutputHolder) {
	var contents []OutputHolder
	if output.Contents != nil {
		contents = output.Contents
	}
	contents = append(contents, holder)
	if output.Settings.SortKey != "" {
		sort.Slice(contents,
			func(i, j int) bool {
				return output.toString(contents[i].Contents[output.Settings.SortKey]) < output.toString(contents[j].Contents[output.Settings.SortKey])
			})
	}
	output.Contents = contents
}

func (output *OutputArray) toString(val interface{}) string {
	if tmp, ok := val.(bool); ok {
		if tmp {
			if output.Settings.UseEmoji {
				return "✅"
			} else {
				return "Yes"
			}
		}
		if output.Settings.UseEmoji {
			return "❌"
		} else {
			return "No"
		}
	}
	return fmt.Sprintf("%s", val)
}
