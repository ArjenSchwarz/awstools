package cmd

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// sampleVPCOverview returns two VPCs with subnets and IP details so every
// table type of the overview report is produced.
func sampleVPCOverview() []helpers.VPCUsageInfo {
	return []helpers.VPCUsageInfo{
		{
			ID:   "vpc-111",
			CIDR: "10.0.0.0/16",
			Subnets: []helpers.SubnetUsageInfo{
				{
					ID: "subnet-aaa", CIDR: "10.0.1.0/24", VPCId: "vpc-111",
					TotalIPs: 256, AvailableIPs: 250, UsedIPs: 6,
					IPDetails: []helpers.IPAddressInfo{
						{IPAddress: "10.0.1.10", UsageType: "EC2 Instance", AttachmentInfo: "i-0123"},
					},
				},
			},
		},
		{
			ID:   "vpc-222",
			CIDR: "10.1.0.0/16",
			Subnets: []helpers.SubnetUsageInfo{
				{
					ID: "subnet-bbb", CIDR: "10.1.1.0/24", VPCId: "vpc-222", IsPublic: true,
					TotalIPs: 256, AvailableIPs: 249, UsedIPs: 7,
					IPDetails: []helpers.IPAddressInfo{
						{IPAddress: "10.1.1.20", UsageType: "NAT Gateway", AttachmentInfo: "nat-0456", PublicIP: "203.0.113.5"},
					},
				},
			},
		},
	}
}

const testAccount = "123456789012"

// Table titles produced for sampleVPCOverview, in report order.
var (
	allVPCTitles = []string{
		"Subnet Overview for vpc-111 in account " + testAccount,
		"Subnet Overview for vpc-222 in account " + testAccount,
		"IP Allocation Summary for vpc-111 in account " + testAccount,
		"IP Allocation Summary for vpc-222 in account " + testAccount,
		"IP Details for subnet subnet-aaa in VPC vpc-111",
		"IP Details for subnet subnet-bbb in VPC vpc-222",
		"VPC Usage Summary for account " + testAccount,
	}
	vpc222Titles = []string{
		"Subnet Overview for vpc-222 in account " + testAccount,
		"IP Allocation Summary for vpc-222 in account " + testAccount,
		"IP Details for subnet subnet-bbb in VPC vpc-222",
		"VPC Usage Summary for vpc-222 in account " + testAccount,
	}
)

// overviewRun describes one vpc overview invocation for writeVPCOverview.
type overviewRun struct {
	outputFormat string
	fileFormat   string
	vpcFilter    string
	appendFile   bool
	vpcs         []helpers.VPCUsageInfo // defaults to sampleVPCOverview()
}

// writeVPCOverview renders the VPC overview for run through the production
// builder and writer into outFile and returns what went to stdout.
func writeVPCOverview(t *testing.T, outFile string, run overviewRun) (string, error) {
	t.Helper()
	configureOutput(t, run.outputFormat, outFile)
	viper.Set("output.file-format", run.fileFormat)
	viper.Set("output.append", run.appendFile)
	t.Cleanup(func() {
		viper.Set("output.file-format", "")
		viper.Set("output.append", false)
	})

	vpcs := run.vpcs
	if vpcs == nil {
		vpcs = sampleVPCOverview()
	}
	tables := buildVPCOverviewTables(vpcs, nil, testAccount, run.vpcFilter)
	var writeErr error
	stdout := captureVPCOverviewStdout(t, func() {
		writeErr = writeVPCOverviewTables(tables, "VPC Overview for account "+testAccount)
	})
	return stdout, writeErr
}

// captureVPCOverviewStdout returns everything f writes to os.Stdout.
func captureVPCOverviewStdout(t *testing.T, f func()) string {
	t.Helper()
	tmp, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("creating stdout capture file: %v", err)
	}
	t.Cleanup(func() { _ = tmp.Close() })
	original := os.Stdout
	os.Stdout = tmp
	t.Cleanup(func() { os.Stdout = original })
	f()
	os.Stdout = original
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return string(data)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	return string(data)
}

// sectionTitles returns the title of each section.
func sectionTitles(sections []vpcOverviewSection) []string {
	titles := make([]string, 0, len(sections))
	for _, section := range sections {
		titles = append(titles, section.Title)
	}
	return titles
}

// TestVPCOverviewStructuredFile_T1536 is the regression test for T-1536:
// with --file only the summary table was saved, because go-output v1.4.0
// truncates the file on every Write() call. JSON and YAML files must be one
// valid document that holds every table and its rows, independent of the
// stdout format.
func TestVPCOverviewStructuredFile_T1536(t *testing.T) {
	tests := map[string]struct {
		run        overviewRun
		wantTitles []string
		wantIPs    []string
	}{
		"table stdout, json file": {
			run:        overviewRun{outputFormat: "table", fileFormat: "json"},
			wantTitles: allVPCTitles,
			wantIPs:    []string{"10.0.1.10", "10.1.1.20"},
		},
		"json stdout and file": {
			run:        overviewRun{outputFormat: "json"},
			wantTitles: allVPCTitles,
			wantIPs:    []string{"10.0.1.10", "10.1.1.20"},
		},
		"json file with vpc filter": {
			run:        overviewRun{outputFormat: "table", fileFormat: "json", vpcFilter: "vpc-222"},
			wantTitles: vpc222Titles,
			wantIPs:    []string{"10.1.1.20"},
		},
		"markdown stdout, yaml file": {
			run:        overviewRun{outputFormat: "markdown", fileFormat: "yaml"},
			wantTitles: allVPCTitles,
			wantIPs:    []string{"10.0.1.10", "10.1.1.20"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "overview")
			if _, err := writeVPCOverview(t, outFile, tc.run); err != nil {
				t.Fatalf("writeVPCOverviewTables returned error: %v", err)
			}
			data := readFile(t, outFile)

			var sections []vpcOverviewSection
			if tc.run.fileFormat == "yaml" {
				if err := yaml.Unmarshal([]byte(data), &sections); err != nil {
					t.Fatalf("file is not valid YAML: %v\n%s", err, data)
				}
			} else {
				if !json.Valid([]byte(data)) {
					t.Fatalf("file is not a single valid JSON document:\n%s", data)
				}
				if err := json.Unmarshal([]byte(data), &sections); err != nil {
					t.Fatalf("file is not a JSON list of sections: %v", err)
				}
			}

			if got, want := sectionTitles(sections), tc.wantTitles; !slices.Equal(got, want) {
				t.Errorf("section titles mismatch:\ngot  %q\nwant %q", got, want)
			}
			var gotIPs []string
			for _, section := range sections {
				for _, row := range section.Rows {
					if ip, ok := row["IP Address"].(string); ok {
						gotIPs = append(gotIPs, ip)
					}
				}
			}
			if got, want := gotIPs, tc.wantIPs; !slices.Equal(got, want) {
				t.Errorf("IP rows mismatch:\ngot  %q\nwant %q", got, want)
			}
			summary := sections[len(sections)-1]
			if len(summary.Rows) != 7 {
				t.Errorf("summary has %d rows, want 7", len(summary.Rows))
			}
		})
	}
}

// htmlSummary is what parseHTMLReport finds in an HTML file.
type htmlSummary struct {
	htmlElements int
	tables       int
	captions     []string
}

// parseHTMLReport tokenizes data with the lenient HTML mode of encoding/xml
// and returns the document structure, failing on malformed markup.
func parseHTMLReport(t *testing.T, data string) htmlSummary {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(data))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var summary htmlSummary
	inCaption := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("HTML file does not parse: %v\n%s", err, data)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "html":
				summary.htmlElements++
			case "table":
				summary.tables++
			case "caption":
				inCaption = true
			}
		case xml.EndElement:
			if tok.Name.Local == "caption" {
				inCaption = false
			}
		case xml.CharData:
			if inCaption {
				summary.captions = append(summary.captions, strings.TrimSpace(string(tok)))
			}
		}
	}
	return summary
}

// TestVPCOverviewHTMLFile_T1536 checks that an HTML file is one page holding
// every table, rather than one full page per table.
func TestVPCOverviewHTMLFile_T1536(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "overview.html")
	if _, err := writeVPCOverview(t, outFile, overviewRun{outputFormat: "table", fileFormat: "html"}); err != nil {
		t.Fatalf("writeVPCOverviewTables returned error: %v", err)
	}
	data := readFile(t, outFile)

	got := parseHTMLReport(t, data)
	if got.htmlElements != 1 {
		t.Errorf("file has %d <html> elements, want 1", got.htmlElements)
	}
	if got.tables != len(allVPCTitles) {
		t.Errorf("file has %d tables, want %d", got.tables, len(allVPCTitles))
	}
	if !slices.Equal(got.captions, allVPCTitles) {
		t.Errorf("table captions mismatch:\ngot  %q\nwant %q", got.captions, allVPCTitles)
	}
	for _, ip := range []string{"10.0.1.10", "10.1.1.20", "203.0.113.5"} {
		if !strings.Contains(data, ip) {
			t.Errorf("file missing IP %q", ip)
		}
	}
}

// TestVPCOverviewHTMLAppend_T1536 checks that --append keeps an existing HTML
// report and adds the new tables to the same page.
func TestVPCOverviewHTMLAppend_T1536(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "overview.html")
	if _, err := writeVPCOverview(t, outFile, overviewRun{outputFormat: "html", vpcFilter: "vpc-111"}); err != nil {
		t.Fatalf("first run returned error: %v", err)
	}
	first := readFile(t, outFile)
	if _, err := writeVPCOverview(t, outFile, overviewRun{outputFormat: "html", vpcFilter: "vpc-222", appendFile: true}); err != nil {
		t.Fatalf("append run returned error: %v", err)
	}
	data := readFile(t, outFile)

	firstBody, _, _ := strings.Cut(first, htmlEndPlaceholder)
	if !strings.HasPrefix(data, firstBody) {
		t.Errorf("appended file does not keep the existing report\nfirst:\n%s\nappended:\n%s", first, data)
	}
	got := parseHTMLReport(t, data)
	if got.htmlElements != 1 {
		t.Errorf("file has %d <html> elements, want 1", got.htmlElements)
	}
	want := []string{
		"Subnet Overview for vpc-111 in account " + testAccount,
		"IP Allocation Summary for vpc-111 in account " + testAccount,
		"IP Details for subnet subnet-aaa in VPC vpc-111",
		"VPC Usage Summary for vpc-111 in account " + testAccount,
	}
	want = append(want, vpc222Titles...)
	if !slices.Equal(got.captions, want) {
		t.Errorf("table captions mismatch:\ngot  %q\nwant %q", got.captions, want)
	}
	if strings.Count(data, htmlEndPlaceholder) != 1 {
		t.Errorf("file has %d end markers, want 1", strings.Count(data, htmlEndPlaceholder))
	}
}

// TestVPCOverviewHTMLAppendUnknownFile_T1536 checks that appending to an HTML
// file without the go-output marker fails instead of dropping the tables or
// the existing content.
func TestVPCOverviewHTMLAppendUnknownFile_T1536(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "other.html")
	existing := "<html><body><p>keep me</p></body></html>"
	if err := os.WriteFile(outFile, []byte(existing), 0o600); err != nil {
		t.Fatalf("writing existing file: %v", err)
	}
	if _, err := writeVPCOverview(t, outFile, overviewRun{outputFormat: "html", appendFile: true}); err == nil {
		t.Fatal("expected an error when appending to an HTML file without the end marker")
	}
	if got := readFile(t, outFile); got != existing {
		t.Errorf("existing file changed to:\n%s", got)
	}
}

// TestVPCOverviewTextFile_T1536 checks text formats: the file holds every
// table, matches stdout when the formats are the same, and respects --vpc.
func TestVPCOverviewTextFile_T1536(t *testing.T) {
	tests := map[string]struct {
		run          overviewRun
		want         []string
		notWant      []string
		equalsStdout bool
	}{
		"markdown stdout and file": {
			run:          overviewRun{outputFormat: "markdown"},
			want:         append([]string{"10.0.1.10", "203.0.113.5"}, allVPCTitles...),
			equalsStdout: true,
		},
		"json stdout, csv file": {
			run:  overviewRun{outputFormat: "json", fileFormat: "csv"},
			want: []string{"10.0.1.10", "10.1.1.20", "10.1.1.0/24", "Total VPCs"},
		},
		"markdown file with vpc filter": {
			run:     overviewRun{outputFormat: "markdown", vpcFilter: "vpc-222"},
			want:    vpc222Titles,
			notWant: []string{"vpc-111", "10.0.1.10"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "overview")
			stdout, err := writeVPCOverview(t, outFile, tc.run)
			if err != nil {
				t.Fatalf("writeVPCOverviewTables returned error: %v", err)
			}
			got := readFile(t, outFile)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("file missing %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("file unexpectedly contains %q, got:\n%s", notWant, got)
				}
			}
			if tc.equalsStdout && got != stdout {
				t.Errorf("file differs from stdout\nstdout:\n%s\nfile:\n%s", stdout, got)
			}
		})
	}
}
