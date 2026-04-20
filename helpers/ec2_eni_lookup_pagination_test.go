package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Regression tests for T-657: the per-ENI endpoint / NAT / TGW lookup helpers
// previously called their Describe* API once and trusted the first page of
// results. In accounts where the matching resource is on page 2 or later the
// ENI appeared unattached. These tests simulate multi-page responses to prove
// the helpers now walk every page.

// mockDescribeVpcEndpointsClient implements ec2.DescribeVpcEndpointsAPIClient
// and paginates a pre-seeded slice based on NextToken so tests can force
// multi-page responses.
type mockDescribeVpcEndpointsClient struct {
	endpoints []types.VpcEndpoint
	pageSize  int
	callCount int
}

func (m *mockDescribeVpcEndpointsClient) DescribeVpcEndpoints(_ context.Context, input *ec2.DescribeVpcEndpointsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	m.callCount++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.endpoints)
	}
	end := start + pageSize
	if end > len(m.endpoints) {
		end = len(m.endpoints)
	}
	out := &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: m.endpoints[start:end],
	}
	if end < len(m.endpoints) {
		token := fmt.Sprintf("%d", end)
		out.NextToken = &token
	}
	return out, nil
}

// mockDescribeNatGatewaysClient implements ec2.DescribeNatGatewaysAPIClient
// with the same paginated-slice behaviour.
type mockDescribeNatGatewaysClient struct {
	gateways  []types.NatGateway
	pageSize  int
	callCount int
}

func (m *mockDescribeNatGatewaysClient) DescribeNatGateways(_ context.Context, input *ec2.DescribeNatGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	m.callCount++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.gateways)
	}
	end := start + pageSize
	if end > len(m.gateways) {
		end = len(m.gateways)
	}
	out := &ec2.DescribeNatGatewaysOutput{
		NatGateways: m.gateways[start:end],
	}
	if end < len(m.gateways) {
		token := fmt.Sprintf("%d", end)
		out.NextToken = &token
	}
	return out, nil
}

// mockDescribeTGWAttachmentsClient implements
// ec2.DescribeTransitGatewayVpcAttachmentsAPIClient with paginated slice
// behaviour.
type mockDescribeTGWAttachmentsClient struct {
	attachments []types.TransitGatewayVpcAttachment
	pageSize    int
	callCount   int
}

func (m *mockDescribeTGWAttachmentsClient) DescribeTransitGatewayVpcAttachments(_ context.Context, input *ec2.DescribeTransitGatewayVpcAttachmentsInput, _ ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	m.callCount++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.attachments)
	}
	end := start + pageSize
	if end > len(m.attachments) {
		end = len(m.attachments)
	}
	out := &ec2.DescribeTransitGatewayVpcAttachmentsOutput{
		TransitGatewayVpcAttachments: m.attachments[start:end],
	}
	if end < len(m.attachments) {
		token := fmt.Sprintf("%d", end)
		out.NextToken = &token
	}
	return out, nil
}

// TestGetVPCEndpointFromNetworkInterface_FindsMatchOnLaterPage verifies that
// a VPC endpoint on a second page of DescribeVpcEndpoints results is still
// found. Before the fix only the first page was inspected so the ENI looked
// unattached.
func TestGetVPCEndpointFromNetworkInterface_FindsMatchOnLaterPage(t *testing.T) {
	// Seed two pages: the matching endpoint is the last item so it is
	// forced onto page two by pageSize: 1.
	endpoints := []types.VpcEndpoint{
		{
			VpcEndpointId:       aws.String("vpce-page1-other"),
			VpcId:               aws.String("vpc-aaa"),
			NetworkInterfaceIds: []string{"eni-unrelated"},
		},
		{
			VpcEndpointId:       aws.String("vpce-page2-match"),
			VpcId:               aws.String("vpc-aaa"),
			NetworkInterfaceIds: []string{"eni-target"},
		},
	}
	mock := &mockDescribeVpcEndpointsClient{endpoints: endpoints, pageSize: 1}

	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
	}

	result := getVPCEndpointFromNetworkInterface(eni, mock)

	if result == nil {
		t.Fatal("expected matching VPC endpoint on page 2, got nil (pagination bug)")
	}
	if aws.ToString(result.VpcEndpointId) != "vpce-page2-match" {
		t.Errorf("got %s, want vpce-page2-match", aws.ToString(result.VpcEndpointId))
	}
	if mock.callCount != 2 {
		t.Errorf("DescribeVpcEndpoints called %d times, want 2 (one per page)", mock.callCount)
	}
}

// TestGetVPCEndpointFromNetworkInterface_NoMatch confirms nil is returned when
// the ENI doesn't match any endpoint across all pages.
func TestGetVPCEndpointFromNetworkInterface_NoMatch(t *testing.T) {
	endpoints := []types.VpcEndpoint{
		{VpcEndpointId: aws.String("vpce-a"), NetworkInterfaceIds: []string{"eni-x"}},
		{VpcEndpointId: aws.String("vpce-b"), NetworkInterfaceIds: []string{"eni-y"}},
	}
	mock := &mockDescribeVpcEndpointsClient{endpoints: endpoints, pageSize: 1}

	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
	}

	if result := getVPCEndpointFromNetworkInterface(eni, mock); result != nil {
		t.Errorf("expected nil when ENI has no matching endpoint, got %s", aws.ToString(result.VpcEndpointId))
	}
}

// TestGetNatGatewayFromNetworkInterface_FindsMatchOnLaterPage verifies that a
// NAT gateway on a second page is still discovered.
func TestGetNatGatewayFromNetworkInterface_FindsMatchOnLaterPage(t *testing.T) {
	gateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-page1-other"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-unrelated")},
			},
		},
		{
			NatGatewayId: aws.String("nat-page2-match"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-target")},
			},
		},
	}
	mock := &mockDescribeNatGatewaysClient{gateways: gateways, pageSize: 1}

	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
	}

	result := getNatGatewayFromNetworkInterface(eni, mock)

	if result == nil {
		t.Fatal("expected matching NAT gateway on page 2, got nil (pagination bug)")
	}
	if aws.ToString(result.NatGatewayId) != "nat-page2-match" {
		t.Errorf("got %s, want nat-page2-match", aws.ToString(result.NatGatewayId))
	}
	if mock.callCount != 2 {
		t.Errorf("DescribeNatGateways called %d times, want 2 (one per page)", mock.callCount)
	}
}

// TestGetNatGatewayFromNetworkInterface_NoMatch confirms nil is returned when
// the ENI doesn't match any NAT gateway address across pages.
func TestGetNatGatewayFromNetworkInterface_NoMatch(t *testing.T) {
	gateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-a"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-x")},
			},
		},
		{
			NatGatewayId: aws.String("nat-b"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-y")},
			},
		},
	}
	mock := &mockDescribeNatGatewaysClient{gateways: gateways, pageSize: 1}

	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
	}

	if result := getNatGatewayFromNetworkInterface(eni, mock); result != nil {
		t.Errorf("expected nil when ENI has no matching NAT gateway, got %s", aws.ToString(result.NatGatewayId))
	}
}

// TestGetTransitGatewayFromNetworkInterface_FindsMatchOnLaterPage verifies the
// sibling TGW attachment lookup also paginates.
func TestGetTransitGatewayFromNetworkInterface_FindsMatchOnLaterPage(t *testing.T) {
	attachments := []types.TransitGatewayVpcAttachment{
		{
			TransitGatewayAttachmentId: aws.String("tgw-attach-page1-other"),
			SubnetIds:                  []string{"subnet-unrelated"},
		},
		{
			TransitGatewayAttachmentId: aws.String("tgw-attach-page2-match"),
			SubnetIds:                  []string{"subnet-target"},
		},
	}
	mock := &mockDescribeTGWAttachmentsClient{attachments: attachments, pageSize: 1}

	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		SubnetId:           aws.String("subnet-target"),
	}

	result := getTransitGatewayFromNetworkInterface(eni, mock)

	if result != "tgw-attach-page2-match" {
		t.Fatalf("expected tgw-attach-page2-match, got %q (pagination bug)", result)
	}
	if mock.callCount != 2 {
		t.Errorf("DescribeTransitGatewayVpcAttachments called %d times, want 2", mock.callCount)
	}
}

// TestGetTransitGatewayFromNetworkInterface_NoMatch confirms an empty string
// is returned when no attachment references the ENI's subnet.
func TestGetTransitGatewayFromNetworkInterface_NoMatch(t *testing.T) {
	attachments := []types.TransitGatewayVpcAttachment{
		{TransitGatewayAttachmentId: aws.String("tgw-a"), SubnetIds: []string{"subnet-x"}},
		{TransitGatewayAttachmentId: aws.String("tgw-b"), SubnetIds: []string{"subnet-y"}},
	}
	mock := &mockDescribeTGWAttachmentsClient{attachments: attachments, pageSize: 1}

	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-target"),
		VpcId:              aws.String("vpc-aaa"),
		SubnetId:           aws.String("subnet-target"),
	}

	if result := getTransitGatewayFromNetworkInterface(eni, mock); result != "" {
		t.Errorf("expected empty string for unmatched subnet, got %q", result)
	}
}

// TestGetVPCEndpointFromNetworkInterface_EmptyVPC ensures the helper
// short-circuits (no API call) when VpcId is unset. Preserves the existing
// contract of the public helper.
func TestGetVPCEndpointFromNetworkInterface_EmptyVPC(t *testing.T) {
	mock := &mockDescribeVpcEndpointsClient{}
	eni := types.NetworkInterface{NetworkInterfaceId: aws.String("eni-x")}
	if result := getVPCEndpointFromNetworkInterface(eni, mock); result != nil {
		t.Errorf("expected nil result, got %s", aws.ToString(result.VpcEndpointId))
	}
	if mock.callCount != 0 {
		t.Errorf("expected no API calls when VpcId is empty, got %d", mock.callCount)
	}
}

// TestGetNatGatewayFromNetworkInterface_EmptyVPC mirrors the above for NAT.
func TestGetNatGatewayFromNetworkInterface_EmptyVPC(t *testing.T) {
	mock := &mockDescribeNatGatewaysClient{}
	eni := types.NetworkInterface{NetworkInterfaceId: aws.String("eni-x")}
	if result := getNatGatewayFromNetworkInterface(eni, mock); result != nil {
		t.Errorf("expected nil result, got %s", aws.ToString(result.NatGatewayId))
	}
	if mock.callCount != 0 {
		t.Errorf("expected no API calls when VpcId is empty, got %d", mock.callCount)
	}
}

// TestGetTransitGatewayFromNetworkInterface_EmptyVPC mirrors the above for TGW.
func TestGetTransitGatewayFromNetworkInterface_EmptyVPC(t *testing.T) {
	mock := &mockDescribeTGWAttachmentsClient{}
	eni := types.NetworkInterface{NetworkInterfaceId: aws.String("eni-x")}
	if result := getTransitGatewayFromNetworkInterface(eni, mock); result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
	if mock.callCount != 0 {
		t.Errorf("expected no API calls when VpcId is empty, got %d", mock.callCount)
	}
}
