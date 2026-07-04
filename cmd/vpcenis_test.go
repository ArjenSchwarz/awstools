package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/viper"
)

// TestEnisGraphFormatError_T1145 verifies that the vpc enis command rejects the
// graph output formats it cannot produce. ENIs are leaf resources with no
// from/to relationship, so dot/drawio cannot generate a meaningful diagram.
//
// Before the T-1145 fix the command advertised dot/drawio in its help but never
// configured DrawIOHeader or FromToColumns. go-output v1.4.0 then either called
// log.Fatal (non-split path) or silently produced nothing (split path). This
// test pins the expectation that those formats now yield a clear, returnable
// error rather than a fatal exit or empty output.
func TestEnisGraphFormatError_T1145(t *testing.T) {
	rejected := []string{"dot", "drawio", "DOT", "DrawIO"}
	for _, format := range rejected {
		if err := enisGraphFormatError(format); err == nil {
			t.Errorf("format %q: expected an error for unsupported graph format, got nil", format)
		}
	}

	allowed := []string{"json", "csv", "table", "html", "mermaid", ""}
	for _, format := range allowed {
		if err := enisGraphFormatError(format); err != nil {
			t.Errorf("format %q: expected no error for supported format, got %v", format, err)
		}
	}
}

// eniAttachmentLookupClient is the minimum EC2 API surface used by
// getAttachment (below) to resolve ENI attachment metadata via the paginated
// helpers. Kept in the test file after T-727 moved production code to use
// ENILookupCache; these tests still validate the per-ENI paginated helpers.
type eniAttachmentLookupClient interface {
	ec2.DescribeVpcEndpointsAPIClient
	ec2.DescribeNatGatewaysAPIClient
	ec2.DescribeTransitGatewayVpcAttachmentsAPIClient
}

// getAttachment resolves the attachment label for a given ENI by dispatching
// to the paginated helpers. Retained in the test file to validate those helpers
// still paginate correctly (T-657/T-705 regression coverage).
func getAttachment(netinterface types.NetworkInterface, svc eniAttachmentLookupClient) string {
	if netinterface.Attachment != nil && netinterface.Attachment.InstanceId != nil {
		return *netinterface.Attachment.InstanceId
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeTransitGateway {
		return helpers.GetTransitGatewayFromNetworkInterface(netinterface, svc)
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeNatGateway || netinterface.InterfaceType == "nat_gateway" {
		natgw := helpers.GetNatGatewayFromNetworkInterface(netinterface, svc)
		if natgw != nil {
			return aws.ToString(natgw.NatGatewayId)
		}
		return ""
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeVpcEndpoint {
		endpoint := helpers.GetVPCEndpointFromNetworkInterface(netinterface, svc)
		if endpoint != nil {
			return fmt.Sprintf("%s (%s)", aws.ToString(endpoint.ServiceName), aws.ToString(endpoint.VpcEndpointId))
		}
		return ""
	}
	return ""
}

// Regression tests for T-705: the vpc enis command resolves ENI attachment
// labels via getAttachment, which dispatches to the paginated helpers fixed
// in T-657. These tests pin the command-side path to the paginated helpers
// so that a future refactor cannot silently reintroduce a single-page lookup.
//
// Each test seeds the mock client with multiple pages of results and puts the
// matching resource on page 2. If pagination is dropped at the command side,
// getAttachment will return an empty attachment label.

// mockENILookupClient paginates endpoints, NAT gateways, and TGW attachments
// from pre-seeded slices so a single mock can satisfy all three branches of
// getAttachment. Each NextToken is the integer offset into the relevant slice.
type mockENILookupClient struct {
	endpoints      []types.VpcEndpoint
	natGateways    []types.NatGateway
	tgwAttachments []types.TransitGatewayVpcAttachment
	pageSize       int

	describeVpcEndpointsCalls int
	describeNatGatewaysCalls  int
	describeTGWCalls          int
}

func paginateToken(next int, total int) *string {
	if next >= total {
		return nil
	}
	token := fmt.Sprintf("%d", next)
	return &token
}

func parseToken(token *string) int {
	if token == nil {
		return 0
	}
	var start int
	if _, err := fmt.Sscanf(*token, "%d", &start); err != nil {
		return 0
	}
	return start
}

func (m *mockENILookupClient) DescribeVpcEndpoints(_ context.Context, input *ec2.DescribeVpcEndpointsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	m.describeVpcEndpointsCalls++
	start := parseToken(input.NextToken)
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.endpoints)
	}
	end := start + pageSize
	if end > len(m.endpoints) {
		end = len(m.endpoints)
	}
	return &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: m.endpoints[start:end],
		NextToken:    paginateToken(end, len(m.endpoints)),
	}, nil
}

func (m *mockENILookupClient) DescribeNatGateways(_ context.Context, input *ec2.DescribeNatGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	m.describeNatGatewaysCalls++
	start := parseToken(input.NextToken)
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.natGateways)
	}
	end := start + pageSize
	if end > len(m.natGateways) {
		end = len(m.natGateways)
	}
	return &ec2.DescribeNatGatewaysOutput{
		NatGateways: m.natGateways[start:end],
		NextToken:   paginateToken(end, len(m.natGateways)),
	}, nil
}

func (m *mockENILookupClient) DescribeTransitGatewayVpcAttachments(_ context.Context, input *ec2.DescribeTransitGatewayVpcAttachmentsInput, _ ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	m.describeTGWCalls++
	start := parseToken(input.NextToken)
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.tgwAttachments)
	}
	end := start + pageSize
	if end > len(m.tgwAttachments) {
		end = len(m.tgwAttachments)
	}
	return &ec2.DescribeTransitGatewayVpcAttachmentsOutput{
		TransitGatewayVpcAttachments: m.tgwAttachments[start:end],
		NextToken:                    paginateToken(end, len(m.tgwAttachments)),
	}, nil
}

// TestGetAttachment_InstanceShortCircuits_T705 ensures that when an ENI is
// directly attached to an EC2 instance, no AWS API call is issued. This keeps
// the fast path fast — the Attachment.InstanceId field is authoritative.
func TestGetAttachment_InstanceShortCircuits_T705(t *testing.T) {
	mock := &mockENILookupClient{}
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-instance"),
		InterfaceType:      types.NetworkInterfaceTypeInterface,
		Attachment: &types.NetworkInterfaceAttachment{
			InstanceId: aws.String("i-1234567890abcdef0"),
		},
	}

	got := getAttachment(eni, mock)

	if got != "i-1234567890abcdef0" {
		t.Errorf("expected instance id, got %q", got)
	}
	if mock.describeVpcEndpointsCalls+mock.describeNatGatewaysCalls+mock.describeTGWCalls != 0 {
		t.Errorf("expected zero API calls for instance-attached ENI, got endpoint=%d nat=%d tgw=%d",
			mock.describeVpcEndpointsCalls, mock.describeNatGatewaysCalls, mock.describeTGWCalls)
	}
}

// TestGetAttachment_VpcEndpoint_FindsMatchOnLaterPage_T705 confirms the
// vpc-endpoint branch walks every page. Before T-657 the matching endpoint
// on page 2 was missed and the attachment field rendered blank.
func TestGetAttachment_VpcEndpoint_FindsMatchOnLaterPage_T705(t *testing.T) {
	mock := &mockENILookupClient{
		endpoints: []types.VpcEndpoint{
			{
				VpcEndpointId:       aws.String("vpce-page1"),
				VpcId:               aws.String("vpc-aaa"),
				ServiceName:         aws.String("com.amazonaws.region.s3"),
				NetworkInterfaceIds: []string{"eni-other"},
			},
			{
				VpcEndpointId:       aws.String("vpce-page2-match"),
				VpcId:               aws.String("vpc-aaa"),
				ServiceName:         aws.String("com.amazonaws.region.dynamodb"),
				NetworkInterfaceIds: []string{"eni-target"},
			},
		},
		pageSize: 1,
	}
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		InterfaceType:      types.NetworkInterfaceTypeVpcEndpoint,
	}

	got := getAttachment(eni, mock)

	want := "com.amazonaws.region.dynamodb (vpce-page2-match)"
	if got != want {
		t.Errorf("attachment label: got %q, want %q — command-side pagination regression", got, want)
	}
	if mock.describeVpcEndpointsCalls != 2 {
		t.Errorf("expected 2 DescribeVpcEndpoints calls (one per page), got %d", mock.describeVpcEndpointsCalls)
	}
}

// TestGetAttachment_NatGateway_FindsMatchOnLaterPage_T705 confirms the NAT
// gateway branch walks every page. Before T-657 the matching gateway on
// page 2 was missed and the attachment column rendered blank.
func TestGetAttachment_NatGateway_FindsMatchOnLaterPage_T705(t *testing.T) {
	mock := &mockENILookupClient{
		natGateways: []types.NatGateway{
			{
				NatGatewayId: aws.String("nat-page1"),
				NatGatewayAddresses: []types.NatGatewayAddress{
					{NetworkInterfaceId: aws.String("eni-other")},
				},
			},
			{
				NatGatewayId: aws.String("nat-page2-match"),
				NatGatewayAddresses: []types.NatGatewayAddress{
					{NetworkInterfaceId: aws.String("eni-target")},
				},
			},
		},
		pageSize: 1,
	}
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		InterfaceType:      types.NetworkInterfaceTypeNatGateway,
	}

	got := getAttachment(eni, mock)

	if got != "nat-page2-match" {
		t.Errorf("attachment label: got %q, want %q — command-side pagination regression", got, "nat-page2-match")
	}
	if mock.describeNatGatewaysCalls != 2 {
		t.Errorf("expected 2 DescribeNatGateways calls, got %d", mock.describeNatGatewaysCalls)
	}
}

// TestGetAttachment_NatGatewayLegacyType_FindsMatchOnLaterPage_T705 covers
// the legacy "nat_gateway" string literal branch — the EC2 API returned that
// form historically and the command accepts both. The pagination behaviour
// must be identical.
func TestGetAttachment_NatGatewayLegacyType_FindsMatchOnLaterPage_T705(t *testing.T) {
	mock := &mockENILookupClient{
		natGateways: []types.NatGateway{
			{
				NatGatewayId: aws.String("nat-page1"),
				NatGatewayAddresses: []types.NatGatewayAddress{
					{NetworkInterfaceId: aws.String("eni-other")},
				},
			},
			{
				NatGatewayId: aws.String("nat-legacy-match"),
				NatGatewayAddresses: []types.NatGatewayAddress{
					{NetworkInterfaceId: aws.String("eni-target")},
				},
			},
		},
		pageSize: 1,
	}
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		InterfaceType:      types.NetworkInterfaceType("nat_gateway"),
	}

	got := getAttachment(eni, mock)

	if got != "nat-legacy-match" {
		t.Errorf("attachment label: got %q, want %q", got, "nat-legacy-match")
	}
	if mock.describeNatGatewaysCalls != 2 {
		t.Errorf("expected 2 DescribeNatGateways calls, got %d", mock.describeNatGatewaysCalls)
	}
}

// TestGetAttachment_TransitGateway_FindsMatchOnLaterPage_T705 confirms the
// TGW branch walks every page. Before T-657 the matching attachment on
// page 2 was missed and the attachment column rendered blank.
func TestGetAttachment_TransitGateway_FindsMatchOnLaterPage_T705(t *testing.T) {
	mock := &mockENILookupClient{
		tgwAttachments: []types.TransitGatewayVpcAttachment{
			{
				TransitGatewayAttachmentId: aws.String("tgw-attach-page1"),
				SubnetIds:                  []string{"subnet-other"},
			},
			{
				TransitGatewayAttachmentId: aws.String("tgw-attach-page2-match"),
				SubnetIds:                  []string{"subnet-target"},
			},
		},
		pageSize: 1,
	}
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		SubnetId:           aws.String("subnet-target"),
		InterfaceType:      types.NetworkInterfaceTypeTransitGateway,
	}

	got := getAttachment(eni, mock)

	if got != "tgw-attach-page2-match" {
		t.Errorf("attachment label: got %q, want %q — command-side pagination regression", got, "tgw-attach-page2-match")
	}
	if mock.describeTGWCalls != 2 {
		t.Errorf("expected 2 DescribeTransitGatewayVpcAttachments calls, got %d", mock.describeTGWCalls)
	}
}

// TestGetAttachment_VpcEndpoint_NoMatch_T705 confirms that a VPC-endpoint
// ENI with no matching endpoint across all pages returns an empty string
// (not a panic, not a partial label).
func TestGetAttachment_VpcEndpoint_NoMatch_T705(t *testing.T) {
	mock := &mockENILookupClient{
		endpoints: []types.VpcEndpoint{
			{VpcEndpointId: aws.String("vpce-a"), NetworkInterfaceIds: []string{"eni-x"}},
			{VpcEndpointId: aws.String("vpce-b"), NetworkInterfaceIds: []string{"eni-y"}},
		},
		pageSize: 1,
	}
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		InterfaceType:      types.NetworkInterfaceTypeVpcEndpoint,
	}

	if got := getAttachment(eni, mock); got != "" {
		t.Errorf("expected empty string for unmatched endpoint ENI, got %q", got)
	}
	if mock.describeVpcEndpointsCalls != 2 {
		t.Errorf("expected exhaustive pagination (2 calls), got %d", mock.describeVpcEndpointsCalls)
	}
}

// sampleENIs returns a small fixed set of ENIs for output-path tests. Two
// subnets are represented so the --split path produces more than one group.
func sampleENIs() []types.NetworkInterface {
	return []types.NetworkInterface{
		{
			NetworkInterfaceId: aws.String("eni-aaa111"),
			InterfaceType:      types.NetworkInterfaceTypeInterface,
			VpcId:              aws.String("vpc-111"),
			SubnetId:           aws.String("subnet-a"),
			PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
				{PrivateIpAddress: aws.String("10.0.0.10")},
			},
		},
		{
			NetworkInterfaceId: aws.String("eni-bbb222"),
			InterfaceType:      types.NetworkInterfaceTypeInterface,
			VpcId:              aws.String("vpc-111"),
			SubnetId:           aws.String("subnet-b"),
			PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
				{PrivateIpAddress: aws.String("10.0.1.20")},
			},
		},
	}
}

// configureOutput sets the viper-backed output settings consumed by
// settings.RenderDocuments() and resets them when the test finishes.
func configureOutput(t *testing.T, format, file string) {
	t.Helper()
	viper.Set("output.format", format)
	viper.Set("output.file", file)
	t.Cleanup(func() {
		viper.Set("output.format", "")
		viper.Set("output.file", "")
	})
}

// noAttachment is a stub resolver so output-path tests do not need AWS clients.
func noAttachment(types.NetworkInterface) string { return "" }

// captureRenderedStdout redirects os.Stdout to a pipe while fn runs and
// returns the bytes written. RenderDocument constructs its stdout writer
// during the call, so redirecting beforehand captures the stdout rendering.
func captureRenderedStdout(t *testing.T, fn func() error) []byte {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fnErr := fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	os.Stdout = old
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	if fnErr != nil {
		t.Fatalf("render failed: %v", fnErr)
	}
	return data
}

// TestEnisRender_NonSplit_FileMatchesStdout_T1294 preserves the T-1294 intent
// for the single-Document flow: `awstools vpc enis --file <path>` must write
// the ENI rows to the file (v1 routed rows through a shared package buffer
// that Write() reset between the stdout and file passes, leaving the file
// empty). In v2 one Document is rendered to both destinations, so the file
// must contain every ENI row and — with matching formats — carry the same
// content as stdout (R4.6).
func TestEnisRender_NonSplit_FileMatchesStdout_T1294(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "enis.json")
	configureOutput(t, "json", outFile)

	doc := buildENIsDocument(sampleENIs(), map[string]string{}, "VPC ENIs", noAttachment)
	stdout := captureRenderedStdout(t, func() error {
		return settings.RenderDocument(t.Context(), doc)
	})

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	contents := strings.TrimSpace(string(data))
	if contents == "" || contents == "[]" || contents == "null" {
		t.Fatalf("output file is empty/has no ENI rows: %q", contents)
	}
	if !json.Valid(data) {
		t.Fatalf("output file is not a single valid JSON document: %q", contents)
	}
	if !strings.Contains(contents, "eni-aaa111") || !strings.Contains(contents, "eni-bbb222") {
		t.Fatalf("output file missing expected ENI rows, got: %q", contents)
	}
	// The stdout writer appends a trailing newline for terminal friendliness;
	// the file writer writes the rendered bytes verbatim. R4.6 is about content
	// equality, so compare with the trailing newline normalized.
	if !bytes.Equal(bytes.TrimRight(stdout, "\n"), bytes.TrimRight(data, "\n")) {
		t.Errorf("file content differs from stdout content for matching formats (R4.6)\nstdout: %q\nfile:   %q", stdout, data)
	}
}

// TestEnisRender_Split_FileContainsAllSubnetTables_T1294 covers the --split
// path against the single-Document flow: one Document holds one table per
// subnet group, and a single render serves both destinations (R7.2). The file
// must contain the rows of ALL subnet groups as one valid JSON document (v1
// wrote a flattened combined array to the file while stdout got separate
// per-group tables, so stdout and file disagreed), and with matching formats
// the file must carry the same content as stdout (R4.6).
func TestEnisRender_Split_FileContainsAllSubnetTables_T1294(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "enis-split.json")
	configureOutput(t, "json", outFile)

	doc := buildENIsBySubnetDocument(sampleENIs(), map[string]string{}, "VPC ENIs", noAttachment)
	stdout := captureRenderedStdout(t, func() error {
		return settings.RenderDocument(t.Context(), doc)
	})

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	contents := strings.TrimSpace(string(data))
	if contents == "" || contents == "[]" || contents == "null" {
		t.Fatalf("split output file is empty/has no ENI rows: %q", contents)
	}
	if !json.Valid(data) {
		t.Fatalf("split output file is not a single valid JSON document: %q", contents)
	}
	// Both subnet groups' ENIs must be present in the file output.
	if !strings.Contains(contents, "eni-aaa111") || !strings.Contains(contents, "eni-bbb222") {
		t.Fatalf("split output file missing ENI rows from one or more subnet groups, got: %q", contents)
	}
	// The stdout writer appends a trailing newline for terminal friendliness;
	// the file writer writes the rendered bytes verbatim. R4.6 is about content
	// equality, so compare with the trailing newline normalized.
	if !bytes.Equal(bytes.TrimRight(stdout, "\n"), bytes.TrimRight(data, "\n")) {
		t.Errorf("split file content differs from stdout content for matching formats (R4.6)\nstdout: %q\nfile:   %q", stdout, data)
	}
}

// TestEnisSplitDocument_PerSubnetTables pins the v1 table shape of the split
// document (R7.2): one table per subnet group, each carrying the v1 group
// title ("<title> - <vpc>: <subnet>") with rows from that subnet only. Table
// format renders titles, so both group titles and both ENIs must appear.
func TestEnisSplitDocument_PerSubnetTables(t *testing.T) {
	configureOutput(t, "table", "")

	doc := buildENIsBySubnetDocument(sampleENIs(), map[string]string{}, "VPC ENIs", noAttachment)
	stdout := string(captureRenderedStdout(t, func() error {
		return settings.RenderDocument(t.Context(), doc)
	}))

	for _, groupTitle := range []string{
		"VPC ENIs - vpc-111: subnet-a",
		"VPC ENIs - vpc-111: subnet-b",
	} {
		if !strings.Contains(stdout, groupTitle) {
			t.Errorf("split output missing per-subnet table title %q, got:\n%s", groupTitle, stdout)
		}
	}
	for _, eni := range []string{"eni-aaa111", "eni-bbb222"} {
		if !strings.Contains(stdout, eni) {
			t.Errorf("split output missing ENI row %q, got:\n%s", eni, stdout)
		}
	}
}
