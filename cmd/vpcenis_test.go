package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

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
