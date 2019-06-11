package drawio

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// CreateOutputFromCSV creates the CSV complete with the header
func CreateOutputFromCSV(header Header, keys []string, contents []map[string]string, filename string) {
	total := [][]string{}
	total = append(total, keys)
	for _, holder := range contents {
		values := make([]string, len(keys))
		for counter, key := range keys {
			if val, ok := holder[key]; ok {
				values[counter] = val
			}
		}
		total = append(total, values)
	}
	var target io.Writer
	if filename == "" {
		target = os.Stdout
	} else {
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		target = bufio.NewWriter(file)
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s", header.String())
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

// GetHeaderAndContentsFromFile returns the headers of a CSV in a reverse map (name:column-id) and the remaining rows
// It filters away all of the comments
func GetHeaderAndContentsFromFile(filename string) (map[string]int, [][]string) {
	headerrow, contents := getContentsFromFile(filename)
	headers := make(map[string]int)
	for index, name := range headerrow {
		headers[name] = index
	}
	return headers, contents
}

// GetContentsFromFileAsStringMaps returns the CSV contents as a slice of string maps
func GetContentsFromFileAsStringMaps(filename string) []map[string]string {
	header, contents := getContentsFromFile(filename)
	result := make([]map[string]string, len(contents))
	for _, row := range contents {
		resultrow := make(map[string]string)
		for index, value := range row {
			resultrow[header[index]] = value
		}
		result = append(result, resultrow)
	}
	return result
}

// getContentsFromFile returns the headers of a CSV in a string slice and the remaining rows separately
// It filters away all of the comments
func getContentsFromFile(filename string) ([]string, [][]string) {
	originalfile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	originalString := string(originalfile)
	r := csv.NewReader(strings.NewReader(originalString))
	r.Comment = '#'
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	// Return headers separate from records
	return records[0], records[1:]
}
