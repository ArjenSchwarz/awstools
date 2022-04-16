package format

import (
	"bufio"
	"bytes"
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
	"github.com/jedib0t/go-pretty/v6/table"

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
func (output OutputArray) Write() {
	switch output.Settings.OutputFormat {
	case "csv":
		output.toCSV()
	case "html":
		output.toHTML()
	case "table":
		output.toTable()
	case "markdown":
		output.toMarkdown()
	case "drawio":
		if !output.Settings.DrawIOHeader.IsSet() {
			log.Fatal("This command doesn't currently support the drawio output format")
		}
		drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), output.Settings.OutputFile)
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
	t := output.buildTable()
	t.RenderCSV()
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
	t := output.buildTable()
	tableBuf := new(bytes.Buffer)
	t.SetOutputMirror(tableBuf)
	t.SetHTMLCSSClass("responstable")
	t.RenderHTML()
	tableBuf.Write([]byte("<div id='end'></div>")) // Add the placeholder
	resultString := strings.Replace(baseTemplate, "<div id='end'></div>", tableBuf.String(), 1)

	err := PrintByteSlice([]byte(resultString), output.Settings.OutputFile)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (output OutputArray) toTable() {
	if output.Settings.SeparateTables {
		fmt.Println("")
	}
	t := output.buildTable()
	t.SetStyle(output.Settings.TableStyle)
	t.Render()
	if output.Settings.SeparateTables {
		fmt.Println("")
	}
}

func (output OutputArray) toMarkdown() {
	t := output.buildTable()
	t.RenderMarkdown()
}

func (output OutputArray) buildTable() table.Writer {
	t := table.NewWriter()
	if output.Settings.Title != "" {
		t.SetTitle(output.Settings.Title)
	}
	var target io.Writer
	// var err error
	if output.Settings.OutputFile == "" {
		target = os.Stdout
	} else {
		//Always create if append flag isn't provided
		if !output.Settings.ShouldAppend {
			target, _ = os.Create(output.Settings.OutputFile)
		} else {
			target, _ = os.OpenFile(output.Settings.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		}
	}
	t.SetOutputMirror(target)
	t.AppendHeader(output.KeysAsInterface())
	for _, cont := range output.ContentsAsInterfaces() {
		t.AppendRow(cont)
	}
	columnConfigs := make([]table.ColumnConfig, 0)
	for _, key := range output.Keys {
		columnConfig := table.ColumnConfig{
			Name:     key,
			WidthMin: 6,
			WidthMax: output.Settings.TableMaxColumnWidth,
		}
		columnConfigs = append(columnConfigs, columnConfig)
	}
	t.SetColumnConfigs(columnConfigs)
	return t
}

func (output *OutputArray) KeysAsInterface() []interface{} {
	b := make([]interface{}, len(output.Keys))
	for i := range output.Keys {
		b[i] = output.Keys[i]
	}

	return b
}

func (output *OutputArray) ContentsAsInterfaces() [][]interface{} {
	total := make([][]interface{}, 0)

	for _, holder := range output.Contents {
		values := make([]interface{}, len(output.Keys))
		for counter, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	return total
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
