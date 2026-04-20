package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// mockDescribeVpcPeeringConnectionsClient simulates DescribeVpcPeeringConnections
// pagination by splitting a pre-configured slice of peering connections across
// multiple pages based on the NextToken. It satisfies
// ec2.DescribeVpcPeeringConnectionsAPIClient.
type mockDescribeVpcPeeringConnectionsClient struct {
	peers     []types.VpcPeeringConnection
	pageSize  int
	callCount int
}

func (m *mockDescribeVpcPeeringConnectionsClient) DescribeVpcPeeringConnections(_ context.Context, input *ec2.DescribeVpcPeeringConnectionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	m.callCount++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(m.peers) {
		end = len(m.peers)
	}
	output := &ec2.DescribeVpcPeeringConnectionsOutput{
		VpcPeeringConnections: m.peers[start:end],
	}
	if end < len(m.peers) {
		next := fmt.Sprintf("%d", end)
		output.NextToken = &next
	}
	return output, nil
}

// makeVpcPeeringConnections builds n dummy peering connections with unique IDs.
func makeVpcPeeringConnections(n int) []types.VpcPeeringConnection {
	peers := make([]types.VpcPeeringConnection, n)
	for i := range n {
		id := fmt.Sprintf("pcx-%08d", i)
		requester := fmt.Sprintf("vpc-req-%08d", i)
		accepter := fmt.Sprintf("vpc-acc-%08d", i)
		peers[i] = types.VpcPeeringConnection{
			VpcPeeringConnectionId: aws.String(id),
			RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
				VpcId:   aws.String(requester),
				OwnerId: aws.String("111111111111"),
			},
			AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
				VpcId:   aws.String(accepter),
				OwnerId: aws.String("222222222222"),
			},
			Tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("peer-name-%d", i))},
			},
		}
	}
	return peers
}

// TestGetAllVpcPeers_Pagination verifies that GetAllVpcPeers retrieves every
// peering connection across multiple pages. Before the fix it only returned
// the first page of DescribeVpcPeeringConnections.
func TestGetAllVpcPeers_Pagination(t *testing.T) {
	total := 5
	mock := &mockDescribeVpcPeeringConnectionsClient{
		peers:    makeVpcPeeringConnections(total),
		pageSize: 2, // force 3 pages: [0,1], [2,3], [4]
	}

	result := getAllVpcPeers(mock)

	if len(result) != total {
		t.Fatalf("getAllVpcPeers() returned %d peerings, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.callCount < 3 {
		t.Errorf("expected at least 3 DescribeVpcPeeringConnections calls for %d peerings at page size %d, got %d", total, mock.pageSize, mock.callCount)
	}
	for i, peer := range result {
		wantID := fmt.Sprintf("pcx-%08d", i)
		if peer.PeeringID != wantID {
			t.Errorf("result[%d].PeeringID = %q, want %q", i, peer.PeeringID, wantID)
		}
		wantReq := fmt.Sprintf("vpc-req-%08d", i)
		if peer.RequesterVpc.ID != wantReq {
			t.Errorf("result[%d].RequesterVpc.ID = %q, want %q", i, peer.RequesterVpc.ID, wantReq)
		}
		wantAcc := fmt.Sprintf("vpc-acc-%08d", i)
		if peer.AccepterVpc.ID != wantAcc {
			t.Errorf("result[%d].AccepterVpc.ID = %q, want %q", i, peer.AccepterVpc.ID, wantAcc)
		}
	}
}

// mockDescribeVpnConnectionsClient satisfies ec2.DescribeVpnConnectionsAPIClient
// and returns a preset slice of VPN connections. AWS's DescribeVpnConnections
// API does not support pagination (no NextToken/MaxResults fields), so this
// mock simply returns the full slice in one call. The regression test verifies
// that addAllVpnNames aggregates every VPN name in the response and uses the
// API client interface so the helper is unit-testable without a real client.
type mockDescribeVpnConnectionsClient struct {
	vpns      []types.VpnConnection
	callCount int
}

func (m *mockDescribeVpnConnectionsClient) DescribeVpnConnections(_ context.Context, _ *ec2.DescribeVpnConnectionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpnConnectionsOutput, error) {
	m.callCount++
	return &ec2.DescribeVpnConnectionsOutput{VpnConnections: m.vpns}, nil
}

// makeVpnConnections builds n VPN connections with predictable IDs and Name
// tags. The first half have Name tags; the second half only have IDs so the
// fallback path is also exercised.
func makeVpnConnections(n int) []types.VpnConnection {
	vpns := make([]types.VpnConnection, n)
	for i := range n {
		id := fmt.Sprintf("vpn-%08d", i)
		conn := types.VpnConnection{
			VpnConnectionId: aws.String(id),
		}
		if i < n/2 {
			conn.Tags = []types.Tag{
				{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("vpn-name-%d", i))},
			}
		}
		vpns[i] = conn
	}
	return vpns
}

// TestAddAllVpnNames_AggregatesAllConnections verifies that addAllVpnNames
// aggregates every VPN connection returned by the AWS API. The AWS API is
// not paginated, but the helper must still work against the
// DescribeVpnConnectionsAPIClient interface so it can be unit tested and so
// the single-call behaviour is guaranteed.
func TestAddAllVpnNames_AggregatesAllConnections(t *testing.T) {
	total := 4
	mock := &mockDescribeVpnConnectionsClient{
		vpns: makeVpnConnections(total),
	}

	result := addAllVpnNames(mock, map[string]string{})

	if len(result) != total {
		t.Fatalf("addAllVpnNames() returned %d entries, want %d", len(result), total)
	}
	for i := range total {
		id := fmt.Sprintf("vpn-%08d", i)
		got, ok := result[id]
		if !ok {
			t.Errorf("addAllVpnNames() missing entry for %s", id)
			continue
		}
		want := id
		if i < total/2 {
			want = fmt.Sprintf("vpn-name-%d", i)
		}
		if got != want {
			t.Errorf("addAllVpnNames()[%s] = %q, want %q", id, got, want)
		}
	}
	if mock.callCount != 1 {
		t.Errorf("DescribeVpnConnections called %d times, want 1", mock.callCount)
	}
}
