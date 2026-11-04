package cmd

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	format "github.com/ArjenSchwarz/go-output"
)

// renderIamrolelist renders the fixture in the given format via the
// production writer and returns stdout.
func renderIamrolelist(t *testing.T, outputFormat string) string {
	t.Helper()
	return renderIamrolelistTo(t, outputFormat, "", "")
}

// renderIamrolelistTo renders the fixture with the given stdout format and,
// when file is set, a file destination in fileFormat. It returns stdout.
func renderIamrolelistTo(t *testing.T, outputFormat, file, fileFormat string) string {
	t.Helper()
	s := format.NewOutputSettings()
	s.SetOutputFormat(outputFormat)
	s.OutputFile = file
	s.OutputFileFormat = fileFormat
	s.Title = "IAM Role overview"
	roles, policies := sameNamedRolelistFixture()
	return captureIAMRoleListStdout(t, func() { writeIamrolelist(roles, policies, s) })
}

// graphEdges parses DOT or Mermaid output into sorted "from -> to" label pairs.
func graphEdges(t *testing.T, out string, node, edge *regexp.Regexp) []string {
	t.Helper()
	labels := map[string]string{}
	for _, m := range node.FindAllStringSubmatch(out, -1) {
		if _, dup := labels[m[1]]; dup {
			t.Fatalf("node ID %s is used for more than one node:\n%s", m[1], out)
		}
		labels[m[1]] = m[2]
	}
	edges := []string{}
	for _, m := range edge.FindAllStringSubmatch(out, -1) {
		from, ok1 := labels[m[1]]
		to, ok2 := labels[m[2]]
		if !ok1 || !ok2 {
			t.Fatalf("edge %s -> %s references an unknown node:\n%s", m[1], m[2], out)
		}
		edges = append(edges, from+" -> "+to)
	}
	slices.Sort(edges)
	return edges
}

// TestIamrolelist_SameNamedPolicies_JSON_T1656 verifies table-style rows keep
// friendly names, expose the ARN, and list only the roles of that ARN.
func TestIamrolelist_SameNamedPolicies_JSON_T1656(t *testing.T) {
	assertRolelistJSON(t, renderIamrolelist(t, "json"))
}

// assertRolelistJSON checks the JSON rows for the fixture: each ARN row keeps
// the friendly name and lists only its own roles, and inline rows are unchanged.
func assertRolelistJSON(t *testing.T, out string) {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	byKey := map[string]map[string]any{}
	for _, row := range rows {
		key, _ := row["ARN"].(string)
		if key == "" {
			key, _ = row["Name"].(string)
		}
		byKey[key] = row
	}

	tests := map[string]struct {
		key, name, typ string
		roles          []any
	}{
		"AWS-managed":      {awsReadOnlyArn, "ReadOnlyAccess", "Attached Policy", []any{"role-a"}},
		"customer-managed": {customerReadOnlyArn, "ReadOnlyAccess", "Attached Policy", []any{"role-b"}},
		"shared policy":    {securityAuditArn, "SecurityAudit", "Attached Policy", []any{"role-a", "role-b"}},
		"inline unchanged": {roleAInlineID, roleAInlineID, "Inline Policy", []any{"role-a"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			row, ok := byKey[tc.key]
			if !ok {
				t.Fatalf("no policy row for %s in %s", tc.key, out)
			}
			if row["Name"] != tc.name || row["Type"] != tc.typ {
				t.Errorf("Name/Type = %v/%v, want %s/%s", row["Name"], row["Type"], tc.name, tc.typ)
			}
			if got, _ := row["Roles"].([]any); !slices.Equal(got, tc.roles) {
				t.Errorf("Roles = %v, want %v", row["Roles"], tc.roles)
			}
		})
	}
	if _, ok := byKey[roleAInlineID]["ARN"]; ok {
		t.Errorf("inline policy row should have no ARN: %v", byKey[roleAInlineID])
	}
	got, _ := byKey["role-b"]["Policies"].([]any)
	slices.SortFunc(got, func(a, b any) int { return strings.Compare(a.(string), b.(string)) })
	if want := []any{"ReadOnlyAccess", "SecurityAudit"}; !slices.Equal(got, want) {
		t.Errorf("role-b Policies = %v, want %v", got, want)
	}
}

// TestIamrolelist_SameNamedPolicies_DOT_T1656 verifies each ARN is its own
// node, linked only from its role.
func TestIamrolelist_SameNamedPolicies_DOT_T1656(t *testing.T) {
	out := renderIamrolelist(t, "dot")
	got := graphEdges(t, out,
		regexp.MustCompile(`(n\d+)\[label="([^"]*)"\]`),
		regexp.MustCompile(`(n\d+)->(n\d+)`))
	want := wantRolelistEdges
	if !slices.Equal(got, want) {
		t.Errorf("edges = %q, want %q\n%s", got, want, out)
	}
}

// TestIamrolelist_SameNamedPolicies_Mermaid_T1656 verifies each role and each
// ARN is its own node and every role keeps its own edges, including roles
// with several policies (go-output's own renderer reuses node IDs there).
func TestIamrolelist_SameNamedPolicies_Mermaid_T1656(t *testing.T) {
	out := renderIamrolelist(t, "mermaid")
	got := mermaidEdges(t, out)
	want := wantRolelistEdges
	if !slices.Equal(got, want) {
		t.Errorf("edges = %q, want %q\n%s", got, want, out)
	}
}

var wantRolelistEdges = []string{
	"role-a -> " + awsReadOnlyArn,
	"role-a -> " + securityAuditArn,
	"role-a -> " + roleAInlineID,
	"role-b -> " + customerReadOnlyArn,
	"role-b -> " + securityAuditArn,
}

func mermaidEdges(t *testing.T, out string) []string {
	t.Helper()
	return graphEdges(t, out,
		regexp.MustCompile(`(n\d+)\("([^"]*)"\)`),
		regexp.MustCompile(`(n\d+) --> (n\d+)`))
}

// TestIamrolelist_MermaidMixedDestinations_T1656 verifies that whichever
// destination asks for Mermaid gets the per-role rendering, while the other
// destination keeps its own format.
func TestIamrolelist_MermaidMixedDestinations_T1656(t *testing.T) {
	tests := map[string]struct {
		stdoutFormat, fileFormat string
	}{
		"mermaid stdout, json file":    {"mermaid", "json"},
		"json stdout, mermaid file":    {"json", "mermaid"},
		"mermaid stdout, mermaid file": {"mermaid", ""},
		"mermaid stdout, html file":    {"mermaid", "html"},
		"mermaid stdout, csv file":     {"mermaid", "csv"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "roles.out")
			stdout := renderIamrolelistTo(t, tc.stdoutFormat, file, tc.fileFormat)
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("reading output file: %v", err)
			}
			fileFormat := tc.fileFormat
			if fileFormat == "" {
				fileFormat = tc.stdoutFormat
			}
			for dest, got := range map[string]struct{ format, out string }{
				"stdout": {tc.stdoutFormat, stdout},
				"file":   {fileFormat, string(data)},
			} {
				switch got.format {
				case "mermaid":
					if edges := mermaidEdges(t, got.out); !slices.Equal(edges, wantRolelistEdges) {
						t.Errorf("%s edges = %q, want %q\n%s", dest, edges, wantRolelistEdges, got.out)
					}
				case "json":
					assertRolelistJSON(t, got.out)
				case "html", "csv":
					if got.format == "html" && !strings.Contains(got.out, "<html") {
						t.Errorf("%s is not HTML: %q", dest, got.out)
					}
					if got.format == "csv" && !strings.Contains(got.out, "\nName,Type,ARN,") {
						t.Errorf("%s is not CSV: %q", dest, got.out)
					}
					for _, arn := range []string{awsReadOnlyArn, customerReadOnlyArn, securityAuditArn} {
						if n := strings.Count(got.out, arn); n != 1 {
							t.Errorf("%s contains %s %d times, want 1", dest, arn, n)
						}
					}
				}
			}
		})
	}
}

// TestIamrolelist_SameNamedPolicies_DrawIO_T1656 verifies Draw.io connects
// roles to policies by DrawioID (the ARN) while labels stay friendly.
func TestIamrolelist_SameNamedPolicies_DrawIO_T1656(t *testing.T) {
	out := renderIamrolelist(t, "drawio")
	if !strings.Contains(out, `# connect: {"from":"Policies","to":"DrawioID"`) {
		t.Fatalf("connection does not target DrawioID:\n%s", out)
	}
	var body []string
	for line := range strings.SplitSeq(out, "\n") {
		if !strings.HasPrefix(line, "#") {
			body = append(body, line)
		}
	}
	records, err := csv.NewReader(strings.NewReader(strings.Join(body, "\n"))).ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v\n%s", err, out)
	}
	col := map[string]int{}
	for i, h := range records[0] {
		col[h] = i
	}
	ids := map[string]string{} // DrawioID -> Name
	edges := []string{}
	for _, rec := range records[1:] {
		ids[rec[col["DrawioID"]]] = rec[col["Name"]]
		for ref := range strings.SplitSeq(rec[col["Policies"]], ",") {
			if ref = strings.TrimSpace(ref); ref != "" {
				edges = append(edges, rec[col["Name"]]+" -> "+ref)
			}
		}
	}
	for _, arn := range []string{awsReadOnlyArn, customerReadOnlyArn} {
		if ids[arn] != "ReadOnlyAccess" {
			t.Errorf("node %s label = %q, want ReadOnlyAccess", arn, ids[arn])
		}
	}
	slices.Sort(edges)
	want := wantRolelistEdges
	if !slices.Equal(edges, want) {
		t.Errorf("edges = %q, want %q", edges, want)
	}
	for _, e := range edges {
		if target := strings.SplitN(e, " -> ", 2)[1]; ids[target] == "" {
			t.Errorf("edge %q targets no DrawioID", e)
		}
	}
}

// TestIamrolelist_MermaidStdoutHTMLAppend_T1656 verifies an appended HTML
// file keeps its existing content and gains the table exactly once when
// stdout is Mermaid.
func TestIamrolelist_MermaidStdoutHTMLAppend_T1656(t *testing.T) {
	file := filepath.Join(t.TempDir(), "roles.html")
	renderIamrolelistTo(t, "html", file, "html")

	s := format.NewOutputSettings()
	s.SetOutputFormat("mermaid")
	s.OutputFile = file
	s.OutputFileFormat = "html"
	s.ShouldAppend = true
	roles, policies := sameNamedRolelistFixture()
	stdout := captureIAMRoleListStdout(t, func() { writeIamrolelist(roles, policies, s) })

	if edges := mermaidEdges(t, stdout); !slices.Equal(edges, wantRolelistEdges) {
		t.Errorf("stdout edges = %q, want %q", edges, wantRolelistEdges)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if n := strings.Count(string(data), "<html"); n != 1 {
		t.Errorf("file has %d <html> elements, want 1", n)
	}
	for _, arn := range []string{awsReadOnlyArn, customerReadOnlyArn} {
		if n := strings.Count(string(data), arn); n != 2 {
			t.Errorf("file contains %s %d times, want 2 (original + appended)", arn, n)
		}
	}
}

// TestIamrolelist_HTMLStdoutMermaidFileAppend_T1656 verifies HTML stdout
// with an appended Mermaid file renders both without reading a "" file.
func TestIamrolelist_HTMLStdoutMermaidFileAppend_T1656(t *testing.T) {
	s := format.NewOutputSettings()
	s.SetOutputFormat("html")
	s.OutputFile = filepath.Join(t.TempDir(), "roles.mmd")
	s.OutputFileFormat = "mermaid"
	s.ShouldAppend = true
	roles, policies := sameNamedRolelistFixture()
	stdout := captureIAMRoleListStdout(t, func() { writeIamrolelist(roles, policies, s) })

	if !strings.Contains(stdout, "<html") || !strings.Contains(stdout, customerReadOnlyArn) {
		t.Errorf("stdout is not the HTML report: %q", stdout)
	}
	data, err := os.ReadFile(s.OutputFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if edges := mermaidEdges(t, string(data)); !slices.Equal(edges, wantRolelistEdges) {
		t.Errorf("file edges = %q, want %q", edges, wantRolelistEdges)
	}
}
