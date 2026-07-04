package cmd

import (
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Fake AWS query-protocol responses for the three calls listResources makes:
// STS GetCallerIdentity and IAM ListAccountAliases (both via
// config.DefaultAwsConfig) and CloudFormation ListStackResources (via
// helpers.GetNestedCloudFormationResources). No NextToken is returned, so the
// ListStackResources paginator stops after one page.
const (
	fakeCallerIdentityXML = `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws:iam::123456789012:user/fake</Arn>
    <UserId>AIDAFAKE</UserId>
    <Account>123456789012</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata><RequestId>fake-request-id</RequestId></ResponseMetadata>
</GetCallerIdentityResponse>`

	fakeListAccountAliasesXML = `<ListAccountAliasesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListAccountAliasesResult>
    <AccountAliases><member>fake-alias</member></AccountAliases>
    <IsTruncated>false</IsTruncated>
  </ListAccountAliasesResult>
  <ResponseMetadata><RequestId>fake-request-id</RequestId></ResponseMetadata>
</ListAccountAliasesResponse>`

	fakeListStackResourcesXML = `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <ListStackResourcesResult>
    <StackResourceSummaries>
      <member>
        <LogicalResourceId>MyBucket</LogicalResourceId>
        <PhysicalResourceId>my-bucket-physical</PhysicalResourceId>
        <ResourceType>AWS::S3::Bucket</ResourceType>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <LastUpdatedTimestamp>2024-01-01T00:00:00.000Z</LastUpdatedTimestamp>
      </member>
      <member>
        <LogicalResourceId>MyRole</LogicalResourceId>
        <PhysicalResourceId>my-role-physical</PhysicalResourceId>
        <ResourceType>AWS::IAM::Role</ResourceType>
        <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
        <LastUpdatedTimestamp>2024-01-01T00:00:00.000Z</LastUpdatedTimestamp>
      </member>
    </StackResourceSummaries>
  </ListStackResourcesResult>
  <ResponseMetadata><RequestId>fake-request-id</RequestId></ResponseMetadata>
</ListStackResourcesResponse>`
)

// newFakeAWSEndpoint starts a local HTTP server that answers the AWS
// query-protocol actions listResources depends on, so the command's full
// production path (including its inline verbose key/row building) can be
// exercised without AWS credentials or network access.
func newFakeAWSEndpoint(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/xml")
		switch action := r.FormValue("Action"); action {
		case "GetCallerIdentity":
			io.WriteString(w, fakeCallerIdentityXML)
		case "ListAccountAliases":
			io.WriteString(w, fakeListAccountAliasesXML)
		case "ListStackResources":
			io.WriteString(w, fakeListStackResourcesXML)
		default:
			http.Error(w, "unexpected action "+action, http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// configureFakeAWSEnv points the AWS SDK at the fake endpoint with static
// credentials and neutralizes the ambient environment (profiles, shared
// config files, IMDS) so the test behaves the same on any machine.
func configureFakeAWSEnv(t *testing.T, endpoint string) {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", "fake-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "fake-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "fake-session-token")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ENDPOINT_URL", endpoint)
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "nonexistent-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "nonexistent-credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
}

// runListResourcesToCSV runs the real listResources command function with the
// given verbose setting, writing CSV to a temp file, and returns the parsed
// header and data rows. The CSV renderer emits the header in WithKeys order,
// so the header row is the command's key set verbatim.
func runListResourcesToCSV(t *testing.T, verbose bool) ([]string, [][]string) {
	t.Helper()
	outFile := filepath.Join(t.TempDir(), "resources.csv")
	viper.Set("output.format", "csv")
	viper.Set("output.file", outFile)
	viper.Set("output.verbose", verbose)
	t.Cleanup(func() {
		viper.Set("output.format", "")
		viper.Set("output.file", "")
		viper.Set("output.verbose", false)
	})

	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	if err := listResources(cmd, nil); err != nil {
		t.Fatalf("listResources(verbose=%v) returned error: %v", verbose, err)
	}

	f, err := os.Open(outFile)
	if err != nil {
		t.Fatalf("opening output file: %v", err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV output: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected a header and at least one data row, got %d records", len(records))
	}
	return records[0], records[1:]
}

// rowByResourceID indexes data rows by their first cell (ResourceID). The
// command builds rows from a channel fan-in, so row order is not guaranteed.
func rowByResourceID(t *testing.T, rows [][]string, id string) []string {
	t.Helper()
	for _, row := range rows {
		if len(row) > 0 && row[0] == id {
			return row
		}
	}
	t.Fatalf("no row found for ResourceID %q in %#v", id, rows)
	return nil
}

// TestListResources_VerboseColumnDelta is the verbose-dimension equivalence
// test (R2.6, R8.2): it exercises cfn resources' key/row building with
// --verbose off and on through the real command path and asserts the column
// and row delta. Verbose must add exactly the Status and LogicalName columns
// (in that order, appended after the base columns) and populate the matching
// cells, leaving the base columns and values untouched.
func TestListResources_VerboseColumnDelta(t *testing.T) {
	server := newFakeAWSEndpoint(t)
	configureFakeAWSEnv(t, server.URL)

	originalStackname := stackname
	stackname = aws.String("teststack")
	t.Cleanup(func() { stackname = originalStackname })

	baseHeader, baseRows := runListResourcesToCSV(t, false)
	verboseHeader, verboseRows := runListResourcesToCSV(t, true)

	wantBaseHeader := []string{"ResourceID", typeColumn, stackColumn, nameColumn}
	wantVerboseHeader := append(slices.Clone(wantBaseHeader), "Status", "LogicalName")

	if !slices.Equal(baseHeader, wantBaseHeader) {
		t.Errorf("non-verbose columns = %v, want %v", baseHeader, wantBaseHeader)
	}
	if !slices.Equal(verboseHeader, wantVerboseHeader) {
		t.Errorf("verbose columns = %v, want %v", verboseHeader, wantVerboseHeader)
	}

	if len(baseRows) != 2 || len(verboseRows) != 2 {
		t.Fatalf("expected 2 data rows in both modes, got %d (non-verbose) and %d (verbose)",
			len(baseRows), len(verboseRows))
	}

	// Base rows: no namefile is configured, so Name falls back to the
	// physical resource id.
	wantBucketBase := []string{"my-bucket-physical", "AWS::S3::Bucket", "teststack", "my-bucket-physical"}
	wantRoleBase := []string{"my-role-physical", "AWS::IAM::Role", "teststack", "my-role-physical"}

	if got := rowByResourceID(t, baseRows, "my-bucket-physical"); !slices.Equal(got, wantBucketBase) {
		t.Errorf("non-verbose bucket row = %v, want %v", got, wantBucketBase)
	}
	if got := rowByResourceID(t, baseRows, "my-role-physical"); !slices.Equal(got, wantRoleBase) {
		t.Errorf("non-verbose role row = %v, want %v", got, wantRoleBase)
	}

	// Verbose rows: identical base cells plus the Status and LogicalName
	// values from the CloudFormation response.
	wantBucketVerbose := append(slices.Clone(wantBucketBase), "CREATE_COMPLETE", "MyBucket")
	wantRoleVerbose := append(slices.Clone(wantRoleBase), "UPDATE_COMPLETE", "MyRole")

	if got := rowByResourceID(t, verboseRows, "my-bucket-physical"); !slices.Equal(got, wantBucketVerbose) {
		t.Errorf("verbose bucket row = %v, want %v", got, wantBucketVerbose)
	}
	if got := rowByResourceID(t, verboseRows, "my-role-physical"); !slices.Equal(got, wantRoleVerbose) {
		t.Errorf("verbose role row = %v, want %v", got, wantRoleVerbose)
	}
}
