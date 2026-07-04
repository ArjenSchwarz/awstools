package cmd

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/config"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// renderDrawIOToFile renders a drawio document through the production
// pipeline (settings.RenderDocuments) with the given records. output.format
// is set to drawio for the test, so the DrawIO flavor serves both the stdout
// and file destinations and the DocumentSet needs no Table flavor.
func renderDrawIOToFile(t *testing.T, records []output.Record, opts ...config.RenderOption) {
	t.Helper()
	docs := config.DocumentSet{
		DrawIO: output.New().
			DrawIO("combine test", records, drawIOBaseHeader("%Name%", "%Image%", imageColumn)).
			Build(),
	}
	if err := settings.RenderDocuments(t.Context(), docs, opts...); err != nil {
		t.Fatalf("RenderDocuments returned error: %v", err)
	}
}

// recordIDs collects the ID cell of every record, in file order.
func recordIDs(records []output.Record) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, drawIORecordString(record, "ID"))
	}
	return ids
}

// recordByID returns the first record whose ID cell matches id.
func recordByID(t *testing.T, records []output.Record, id string) output.Record {
	t.Helper()
	for _, record := range records {
		if drawIORecordString(record, "ID") == id {
			return record
		}
	}
	t.Fatalf("no record found for ID %q in %#v", id, records)
	return nil
}

// TestDrawIOCombineAndAppend is the drawio combine-and-append test (R5.4,
// R8.4): a second run with output.append set must merge the prior file's
// records with the new ones instead of blindly appending. It writes a drawio
// CSV via the v2 pipeline, runs the ported merge logic the way tgw overview
// and vpc peerings do (ParseDrawIOFile + drawIORecordString keyed by ID, with
// unique() deduplicating destinations), renders again with
// config.WithFileOverwrite(), and asserts the combined ID set, dedup, and
// column order on the final file.
func TestDrawIOCombineAndAppend(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "diagram.csv")
	viper.Set("output.format", "drawio")
	viper.Set("output.file", outFile)
	viper.Set("output.append", false)
	t.Cleanup(func() {
		viper.Set("output.format", "")
		viper.Set("output.file", "")
		viper.Set("output.append", false)
	})

	// First run: three records with distinct IDs.
	renderDrawIOToFile(t, []output.Record{
		{"ID": "tgw-1", nameColumn: "gateway one", destinationsColumn: "", imageColumn: "shape-tgw"},
		{"ID": "vpc-a", nameColumn: "vpc a", destinationsColumn: "tgw-1", imageColumn: "shape-vpc"},
		{"ID": "vpc-b", nameColumn: "vpc b", destinationsColumn: "tgw-1", imageColumn: "shape-vpc"},
	})

	firstParsed, err := output.ParseDrawIOFile(outFile)
	if err != nil {
		t.Fatalf("parsing first drawio file: %v", err)
	}
	// Without an explicit column order the renderer alphabetizes the record
	// keys, so the header row of the written CSV is deterministic.
	wantColumns := []string{destinationsColumn, "ID", imageColumn, nameColumn}
	if !slices.Equal(firstParsed.Columns, wantColumns) {
		t.Fatalf("first file columns = %v, want %v", firstParsed.Columns, wantColumns)
	}
	if got := recordIDs(firstParsed.Records); !slices.Equal(got, []string{"tgw-1", "vpc-a", "vpc-b"}) {
		t.Fatalf("first file IDs = %v, want [tgw-1 vpc-a vpc-b]", got)
	}

	// Second run with output.append set: the combine-and-append gate the
	// commands check must be active for the drawio format.
	viper.Set("output.append", true)
	if !settings.ShouldCombineAndAppend() {
		t.Fatal("ShouldCombineAndAppend() = false with output.append=true and drawio format, want true")
	}

	// Ported merge logic, as tgw overview and vpc peerings run it: read the
	// prior file's records back keyed by ID, then overlay the new run's
	// records so overlapping IDs are replaced (dedup) and new IDs are added.
	parsed, err := output.ParseDrawIOFile(settings.GetString("output.file"))
	if err != nil {
		t.Fatalf("parsing prior drawio file for merge: %v", err)
	}
	mapping := make(map[string]output.Record, len(parsed.Records))
	for _, record := range parsed.Records {
		id := drawIORecordString(record, "ID")
		mapping[id] = output.Record{
			"ID":               id,
			nameColumn:         drawIORecordString(record, nameColumn),
			destinationsColumn: drawIORecordString(record, destinationsColumn),
			imageColumn:        drawIORecordString(record, imageColumn),
		}
	}
	// New run: vpc-b overlaps with the prior file (its destinations gain
	// tgw-2 and must merge without duplicating tgw-1), vpn-1 is new.
	newRun := []output.Record{
		{"ID": "vpc-b", nameColumn: "vpc b", destinationsColumn: "tgw-2", imageColumn: "shape-vpc"},
		{"ID": "vpn-1", nameColumn: "vpn one", destinationsColumn: "tgw-1", imageColumn: "shape-vpn"},
	}
	for _, record := range newRun {
		id := drawIORecordString(record, "ID")
		destinations := strings.Split(drawIORecordString(record, destinationsColumn), ",")
		if prior, ok := mapping[id]; ok {
			destinations = unique(append(destinations, strings.Split(drawIORecordString(prior, destinationsColumn), ",")...))
		}
		record[destinationsColumn] = strings.Join(destinations, ",")
		mapping[id] = record
	}
	combined := make([]output.Record, 0, len(mapping))
	for _, id := range slices.Sorted(maps.Keys(mapping)) {
		combined = append(combined, mapping[id])
	}
	// The prior file contents are already merged into the document, so the
	// file must be written fresh instead of appended to.
	renderDrawIOToFile(t, combined, config.WithFileOverwrite())

	finalParsed, err := output.ParseDrawIOFile(outFile)
	if err != nil {
		t.Fatalf("parsing final drawio file: %v", err)
	}

	// Column order is preserved across the combine run.
	if !slices.Equal(finalParsed.Columns, firstParsed.Columns) {
		t.Errorf("final file columns = %v, want %v (same as first file)", finalParsed.Columns, firstParsed.Columns)
	}

	// Combined ID set: old + new, with the overlapping vpc-b present exactly
	// once. Blind appending would have duplicated the first run's records.
	wantIDs := []string{"tgw-1", "vpc-a", "vpc-b", "vpn-1"}
	if got := recordIDs(finalParsed.Records); !slices.Equal(got, wantIDs) {
		t.Errorf("final file IDs = %v, want %v", got, wantIDs)
	}

	// The overlapping record carries the merged, deduplicated destinations,
	// and an untouched prior record survives the merge unchanged.
	vpcB := recordByID(t, finalParsed.Records, "vpc-b")
	if got := drawIORecordString(vpcB, destinationsColumn); got != "tgw-2,tgw-1" {
		t.Errorf("vpc-b destinations = %q, want %q", got, "tgw-2,tgw-1")
	}
	vpcA := recordByID(t, finalParsed.Records, "vpc-a")
	if got := drawIORecordString(vpcA, nameColumn); got != "vpc a" {
		t.Errorf("vpc-a name = %q, want %q", got, "vpc a")
	}
	if got := drawIORecordString(vpcA, destinationsColumn); got != "tgw-1" {
		t.Errorf("vpc-a destinations = %q, want %q", got, "tgw-1")
	}
}
