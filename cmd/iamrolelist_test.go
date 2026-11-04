package cmd

import (
	"encoding/csv"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// renderIamrolelist renders the fixture in the given format via the
// production writer and returns stdout.
func renderIamrolelist(t *testing.T, format string, withInline bool) string {
	t.Helper()
	configureOutput(t, format, "")
	roles, policies := sameNamedRolelistFixture(withInline)
	return captureStdout(t, func() { writeIamrolelist(roles, policies, "IAM Role overview") })
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
	var rows []map[string]any
	out := renderIamrolelist(t, "json", true)
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
	if got := byKey["role-b"]["Policies"]; !slices.Equal(got.([]any), []any{"ReadOnlyAccess"}) {
		t.Errorf("role-b Policies = %v, want [ReadOnlyAccess]", got)
	}
}

// TestIamrolelist_SameNamedPolicies_DOT_T1656 verifies each ARN is its own
// node, linked only from its role.
func TestIamrolelist_SameNamedPolicies_DOT_T1656(t *testing.T) {
	out := renderIamrolelist(t, "dot", true)
	got := graphEdges(t, out,
		regexp.MustCompile(`(n\d+)\[label="([^"]*)"\]`),
		regexp.MustCompile(`(n\d+)->(n\d+)`))
	want := []string{
		"role-a -> " + awsReadOnlyArn,
		"role-a -> " + roleAInlineID,
		"role-b -> " + customerReadOnlyArn,
	}
	if !slices.Equal(got, want) {
		t.Errorf("edges = %q, want %q\n%s", got, want, out)
	}
}

// TestIamrolelist_SameNamedPolicies_Mermaid_T1656 verifies each ARN is its
// own node, linked only from its role. It uses one policy per role because
// go-output v1.4.0 reuses Mermaid node IDs for sources with several edges.
func TestIamrolelist_SameNamedPolicies_Mermaid_T1656(t *testing.T) {
	out := renderIamrolelist(t, "mermaid", false)
	got := graphEdges(t, out,
		regexp.MustCompile(`(n\d+)\("([^"]*)"\)`),
		regexp.MustCompile(`(n\d+) --> (n\d+)`))
	want := []string{
		"role-a -> " + awsReadOnlyArn,
		"role-b -> " + customerReadOnlyArn,
	}
	if !slices.Equal(got, want) {
		t.Errorf("edges = %q, want %q\n%s", got, want, out)
	}
}

// TestIamrolelist_SameNamedPolicies_DrawIO_T1656 verifies Draw.io connects
// roles to policies by DrawioID (the ARN) while labels stay friendly.
func TestIamrolelist_SameNamedPolicies_DrawIO_T1656(t *testing.T) {
	out := renderIamrolelist(t, "drawio", true)
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
	want := []string{
		"role-a -> " + awsReadOnlyArn,
		"role-a -> " + roleAInlineID,
		"role-b -> " + customerReadOnlyArn,
	}
	if !slices.Equal(edges, want) {
		t.Errorf("edges = %q, want %q", edges, want)
	}
	for _, e := range edges {
		if target := strings.SplitN(e, " -> ", 2)[1]; ids[target] == "" {
			t.Errorf("edge %q targets no DrawioID", e)
		}
	}
}
