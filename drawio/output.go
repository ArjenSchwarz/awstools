package drawio

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strings"
)

// GetHeaderAndContentsFromFile returns the headers of a CSV in a reverse map (name:column-id) and the remaining rows
func GetHeaderAndContentsFromFile(filename string) (map[string]int, [][]string) {
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
	headers := make(map[string]int)
	for index, name := range records[0] {
		headers[name] = index
	}
	// Return headers separate from records
	return headers, records[1:]
}
