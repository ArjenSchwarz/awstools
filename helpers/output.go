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

	"github.com/ArjenSchwarz/awstools/config"
)

type OutputHolder struct {
	Contents map[string]string
}

type OutputArray struct {
	Contents []OutputHolder
	Keys     []string
}

func (output OutputArray) Write(settings config.Config) {
	switch strings.ToLower(*settings.OutputFormat) {
	case "json":
		output.toJSON(*settings.OutputFile)
	default:
		output.toCSV(*settings.OutputFile)
	}
}

func (output OutputArray) toCSV(outputFile string) {
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

func (output *OutputArray) AddHolder(holder OutputHolder) {
	var contents []OutputHolder
	if output.Contents != nil {
		contents = output.Contents
	}
	output.Contents = append(contents, holder)
}
