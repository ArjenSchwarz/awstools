package cmd

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/config"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// The per-format equivalence oracle (R2.x, R8.1, R9.3): a fixed in-memory
// fixture rendered through the production pipeline (config.DocumentSet +
// settings.RenderDocuments), with the resulting bytes pinned per format.
// Multi-value ([]string) and bool cells are asserted against go-output v2's
// default formatting, which Decision 7 accepts as the new baseline.

// oracleRows returns the fixture rows: string, []string, bool, and int cells.
func oracleRows() []map[string]any {
	return []map[string]any{
		{"Name": "alpha", "Targets": []string{"beta", "gamma"}, "Public": true, "Count": 3},
		{"Name": "beta", "Targets": []string{"alpha"}, "Public": false, "Count": 1},
	}
}

var oracleKeys = []string{"Name", "Targets", "Public", "Count"}

// oracleRecords returns the drawio flavor's records for the same entities.
// The []string Parent cell pins drawIORecords' comma-join: passed raw, v2's
// renderer would emit fmt.Sprint's "[alpha beta]" and break edge resolution.
func oracleRecords() []output.Record {
	return []output.Record{
		{"Name": "alpha", "Image": "img-a", "Parent": ""},
		{"Name": "beta", "Image": "img-b", "Parent": "alpha"},
		{"Name": "gamma", "Image": "img-c", "Parent": []string{"alpha", "beta"}},
	}
}

func oracleDrawIOHeader() output.DrawIOHeader {
	header := drawIOBaseHeader("%Name%", "%Image%", "Image")
	header.Connections = append(header.Connections, drawIOConnection())
	return header
}

// renderOracleToFile renders the given fixture through the production
// pipeline into a file in the requested format and returns the file bytes.
func renderOracleToFile(t *testing.T, format string, rows []map[string]any, records []output.Record) []byte {
	t.Helper()
	outFile := filepath.Join(t.TempDir(), "oracle-out")
	viper.Set("output.format", format)
	viper.Set("output.file", outFile)
	t.Cleanup(func() {
		viper.Set("output.format", "")
		viper.Set("output.file", "")
	})
	docs := config.DocumentSet{
		Table:  output.New().Table("Oracle Fixture", rows, output.WithKeys(oracleKeys...)).Build(),
		Graph:  output.New().Graph("Oracle Fixture", graphEdges(rows, "Name", "Targets")).Build(),
		DrawIO: output.New().DrawIO("Oracle Fixture", records, oracleDrawIOHeader()).Build(),
	}
	if err := settings.RenderDocuments(t.Context(), docs); err != nil {
		t.Fatalf("RenderDocuments(%s) returned error: %v", format, err)
	}
	contents, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading %s output: %v", format, err)
	}
	return contents
}

// renderedDocument is the shape v2's json and yaml renderers emit.
type renderedDocument struct {
	Title  string           `json:"title" yaml:"title"`
	Data   []map[string]any `json:"data" yaml:"data"`
	Schema struct {
		Keys []string `json:"keys" yaml:"keys"`
	} `json:"schema" yaml:"schema"`
}

// firstDataObjectKeys walks the JSON token stream and returns the keys of the
// first object inside "data", in emission order.
func firstDataObjectKeys(t *testing.T, contents []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(contents))
	for {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("scanning for data key: %v", err)
		}
		if s, ok := tok.(string); ok && s == "data" {
			break
		}
	}
	for _, want := range []json.Delim{'[', '{'} {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("entering data object: %v", err)
		}
		if d, ok := tok.(json.Delim); !ok || d != want {
			t.Fatalf("entering data object: got token %v, want %v", tok, want)
		}
	}
	var keys []string
	for {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("reading data object keys: %v", err)
		}
		if d, ok := tok.(json.Delim); ok && d == '}' {
			return keys
		}
		keys = append(keys, tok.(string))
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			t.Fatalf("skipping value of %q: %v", keys[len(keys)-1], err)
		}
	}
}

// assertOrderedSubstrings fails unless every wanted substring occurs and each
// occurs after the previous one — order-sensitive but styling-agnostic.
func assertOrderedSubstrings(t *testing.T, contents string, wants []string) {
	t.Helper()
	offset := 0
	for _, want := range wants {
		idx := strings.Index(contents[offset:], want)
		if idx < 0 {
			t.Fatalf("substring %q missing (or out of order) in output:\n%s", want, contents)
		}
		offset += idx + len(want)
	}
}

// edgeSet extracts "from->to" pairs matched by the given pattern (two capture
// groups) and returns the sorted edge list plus the sorted node set.
func edgeSet(contents string, pattern *regexp.Regexp) (edges, nodes []string) {
	nodeSet := make(map[string]bool)
	for _, match := range pattern.FindAllStringSubmatch(contents, -1) {
		edges = append(edges, match[1]+"->"+match[2])
		nodeSet[match[1]] = true
		nodeSet[match[2]] = true
	}
	slices.Sort(edges)
	return edges, slices.Sorted(maps.Keys(nodeSet))
}

func TestFormatOracleJSON(t *testing.T) {
	contents := renderOracleToFile(t, "json", oracleRows(), oracleRecords())

	var doc renderedDocument
	if err := json.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("json output does not parse: %v", err)
	}
	wantData := []map[string]any{
		{"Name": "alpha", "Targets": []any{"beta", "gamma"}, "Public": true, "Count": float64(3)},
		{"Name": "beta", "Targets": []any{"alpha"}, "Public": false, "Count": float64(1)},
	}
	if doc.Title != "Oracle Fixture" {
		t.Errorf("json title = %q, want %q", doc.Title, "Oracle Fixture")
	}
	if !reflect.DeepEqual(doc.Data, wantData) {
		t.Errorf("json data = %#v, want %#v", doc.Data, wantData)
	}
	if !slices.Equal(doc.Schema.Keys, oracleKeys) {
		t.Errorf("json schema keys = %v, want %v", doc.Schema.Keys, oracleKeys)
	}
	// Key order inside data objects is deterministic (alphabetical), pinned
	// via the token stream rather than the order-losing unmarshal.
	wantOrder := []string{"Count", "Name", "Public", "Targets"}
	if got := firstDataObjectKeys(t, contents); !slices.Equal(got, wantOrder) {
		t.Errorf("json data object key order = %v, want %v", got, wantOrder)
	}
}

func TestFormatOracleYAML(t *testing.T) {
	contents := renderOracleToFile(t, "yaml", oracleRows(), oracleRecords())

	var doc renderedDocument
	if err := yaml.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("yaml output does not parse: %v", err)
	}
	wantData := []map[string]any{
		{"Name": "alpha", "Targets": []any{"beta", "gamma"}, "Public": true, "Count": 3},
		{"Name": "beta", "Targets": []any{"alpha"}, "Public": false, "Count": 1},
	}
	if doc.Title != "Oracle Fixture" {
		t.Errorf("yaml title = %q, want %q", doc.Title, "Oracle Fixture")
	}
	if !reflect.DeepEqual(doc.Data, wantData) {
		t.Errorf("yaml data = %#v, want %#v", doc.Data, wantData)
	}
	if !slices.Equal(doc.Schema.Keys, oracleKeys) {
		t.Errorf("yaml schema keys = %v, want %v", doc.Schema.Keys, oracleKeys)
	}
}

func TestFormatOracleCSV(t *testing.T) {
	contents := renderOracleToFile(t, "csv", oracleRows(), oracleRecords())

	// Header order follows WithKeys; []string and bool cells take v2's
	// default formatting (D7): Go slice syntax and true/false.
	want := "Name,Targets,Public,Count\n" +
		"alpha,[beta gamma],true,3\n" +
		"beta,[alpha],false,1\n"
	if got := string(contents); got != want {
		t.Errorf("csv output = %q, want %q", got, want)
	}
}

func TestFormatOracleTable(t *testing.T) {
	contents := string(renderOracleToFile(t, "table", oracleRows(), oracleRecords()))

	// Styling-agnostic: headers and row values in order, no box-drawing pins.
	assertOrderedSubstrings(t, contents, []string{
		"Oracle Fixture",
		"NAME", "TARGETS", "PUBLIC", "COUNT",
		"alpha", "beta", "true", "3",
		"gamma",
		"beta", "alpha", "false", "1",
	})
}

func TestFormatOracleMarkdown(t *testing.T) {
	contents := string(renderOracleToFile(t, "markdown", oracleRows(), oracleRecords()))

	for _, want := range []string{
		"### Oracle Fixture",
		"| Name | Targets | Public | Count |",
		"| alpha | beta<br/>gamma | true | 3 |",
		"| beta | alpha | false | 1 |",
	} {
		if !strings.Contains(contents, want) {
			t.Errorf("markdown output missing %q:\n%s", want, contents)
		}
	}
}

func TestFormatOracleHTML(t *testing.T) {
	contents := string(renderOracleToFile(t, "html", oracleRows(), oracleRecords()))

	if !strings.HasPrefix(contents, "<!DOCTYPE html>") || !strings.Contains(contents, "</html>") {
		t.Fatalf("html output is not a full DefaultHTMLTemplate document:\n%.200s", contents)
	}
	for _, want := range []string{"Oracle Fixture", "alpha", "gamma"} {
		if !strings.Contains(contents, want) {
			t.Errorf("html output missing data value %q", want)
		}
	}
}

var (
	dotEdgePattern     = regexp.MustCompile(`(\w+) -> (\w+);`)
	mermaidEdgePattern = regexp.MustCompile(`(\w+) --> (\w+)`)
)

func TestFormatOracleDot(t *testing.T) {
	contents := string(renderOracleToFile(t, "dot", oracleRows(), oracleRecords()))

	if !strings.Contains(contents, "digraph {") {
		t.Fatalf("dot output is not a digraph:\n%s", contents)
	}
	edges, nodes := edgeSet(contents, dotEdgePattern)
	if want := []string{"alpha->beta", "alpha->gamma", "beta->alpha"}; !slices.Equal(edges, want) {
		t.Errorf("dot edge set = %v, want %v", edges, want)
	}
	if want := []string{"alpha", "beta", "gamma"}; !slices.Equal(nodes, want) {
		t.Errorf("dot node set = %v, want %v", nodes, want)
	}
}

func TestFormatOracleMermaid(t *testing.T) {
	contents := string(renderOracleToFile(t, "mermaid", oracleRows(), oracleRecords()))

	if !strings.HasPrefix(contents, "graph TD") {
		t.Fatalf("mermaid output is not a TD flowchart:\n%s", contents)
	}
	edges, nodes := edgeSet(contents, mermaidEdgePattern)
	if want := []string{"alpha->beta", "alpha->gamma", "beta->alpha"}; !slices.Equal(edges, want) {
		t.Errorf("mermaid edge set = %v, want %v", edges, want)
	}
	if want := []string{"alpha", "beta", "gamma"}; !slices.Equal(nodes, want) {
		t.Errorf("mermaid node set = %v, want %v", nodes, want)
	}
}

func TestFormatOracleDrawIO(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "oracle.csv")
	viper.Set("output.format", "drawio")
	viper.Set("output.file", outFile)
	t.Cleanup(func() {
		viper.Set("output.format", "")
		viper.Set("output.file", "")
	})
	docs := config.DocumentSet{
		DrawIO: output.New().DrawIO("Oracle Fixture", drawIORecords(oracleRecords()), oracleDrawIOHeader()).Build(),
	}
	if err := settings.RenderDocuments(t.Context(), docs); err != nil {
		t.Fatalf("RenderDocuments(drawio) returned error: %v", err)
	}

	// Round-trip: nodes, shapes, and connections all survive the CSV.
	parsed, err := output.ParseDrawIOFile(outFile)
	if err != nil {
		t.Fatalf("drawio output does not round-trip: %v", err)
	}
	if want := []string{"Image", "Name", "Parent"}; !slices.Equal(parsed.Columns, want) {
		t.Errorf("drawio columns = %v, want %v", parsed.Columns, want)
	}
	if len(parsed.Records) != 3 {
		t.Fatalf("drawio records = %d, want 3", len(parsed.Records))
	}
	// Expected cells spelled out: the []string Parent cell must round-trip as
	// v1's comma-joined multi-value ref, never fmt.Sprint's "[alpha beta]".
	wantRecords := []output.Record{
		{"Name": "alpha", "Image": "img-a", "Parent": ""},
		{"Name": "beta", "Image": "img-b", "Parent": "alpha"},
		{"Name": "gamma", "Image": "img-c", "Parent": "alpha,beta"},
	}
	for i, want := range wantRecords {
		for key, wantValue := range want {
			if got := drawIORecordString(parsed.Records[i], key); got != wantValue {
				t.Errorf("drawio record %d %s = %q, want %q", i, key, got, wantValue)
			}
		}
	}
	if parsed.Header.Label != "%Name%" || parsed.Header.Style != "%Image%" || parsed.Header.Ignore != "Image" {
		t.Errorf("drawio header shape = %q/%q/%q, want %%Name%%/%%Image%%/Image",
			parsed.Header.Label, parsed.Header.Style, parsed.Header.Ignore)
	}
	if parsed.Header.Height != "78" || parsed.Header.Width != "78" {
		t.Errorf("drawio header size = %sx%s, want 78x78", parsed.Header.Height, parsed.Header.Width)
	}
	wantConnection := drawIOConnection()
	if len(parsed.Header.Connections) != 1 || parsed.Header.Connections[0] != wantConnection {
		t.Errorf("drawio connections = %+v, want [%+v]", parsed.Header.Connections, wantConnection)
	}
}

// TestFormatOracleEmpty pins R9.3: an empty result set still renders valid
// output in every format.
func TestFormatOracleEmpty(t *testing.T) {
	wants := map[string][]string{
		"json":     {`"title": "Oracle Fixture"`, `"keys"`},
		"yaml":     {"title: Oracle Fixture", "keys:"},
		"csv":      {"Name,Targets,Public,Count"},
		"table":    {"Oracle Fixture", "NAME", "COUNT"},
		"markdown": {"### Oracle Fixture", "| Name | Targets | Public | Count |"},
		"html":     {"<!DOCTYPE html>", "</html>"},
		"dot":      {"digraph {", "}"},
		"mermaid":  {"graph TD"},
		"drawio":   {"# label: %Name%", "# height: 78"},
	}
	for format, substrings := range wants {
		t.Run(format, func(t *testing.T) {
			contents := string(renderOracleToFile(t, format, []map[string]any{}, []output.Record{}))
			for _, want := range substrings {
				if !strings.Contains(contents, want) {
					t.Errorf("empty %s output missing %q:\n%s", format, want, contents)
				}
			}
			if format == "json" {
				var doc renderedDocument
				if err := json.Unmarshal([]byte(contents), &doc); err != nil {
					t.Errorf("empty json output does not parse: %v", err)
				}
			}
		})
	}
}
