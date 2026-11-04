package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	dxtypes "github.com/aws/aws-sdk-go-v2/service/directconnect/types"
)

// mockDescribeDirectConnectGatewaysAPIClient serves gateways in fixed-size
// pages, using the start index as the NextToken.
type mockDescribeDirectConnectGatewaysAPIClient struct {
	gateways []dxtypes.DirectConnectGateway
	pageSize int
	tokens   []string
}

func (m *mockDescribeDirectConnectGatewaysAPIClient) DescribeDirectConnectGateways(_ context.Context, input *directconnect.DescribeDirectConnectGatewaysInput, _ ...func(*directconnect.Options)) (*directconnect.DescribeDirectConnectGatewaysOutput, error) {
	token := aws.ToString(input.NextToken)
	m.tokens = append(m.tokens, token)
	start := 0
	if token != "" {
		if _, err := fmt.Sscanf(token, "%d", &start); err != nil {
			return nil, err
		}
	}
	end := min(start+m.pageSize, len(m.gateways))
	out := &directconnect.DescribeDirectConnectGatewaysOutput{
		DirectConnectGateways: m.gateways[start:end],
	}
	if end < len(m.gateways) {
		out.NextToken = aws.String(fmt.Sprintf("%d", end))
	}
	return out, nil
}

// TestAddAllDirectConnectGatewayNames_Pagination verifies that gateways on
// every page are resolved. Before T-1623 only the first response was read.
func TestAddAllDirectConnectGatewayNames_Pagination(t *testing.T) {
	gateways := []dxtypes.DirectConnectGateway{
		{DirectConnectGatewayId: aws.String("dxgw-0"), DirectConnectGatewayName: aws.String("first")},
		{DirectConnectGatewayId: aws.String("dxgw-1"), DirectConnectGatewayName: aws.String("second")},
		{DirectConnectGatewayId: aws.String("dxgw-2"), DirectConnectGatewayName: aws.String("third")},
		{DirectConnectGatewayId: aws.String("dxgw-3")},
		{DirectConnectGatewayId: aws.String("dxgw-4"), DirectConnectGatewayName: aws.String("")},
	}
	mock := &mockDescribeDirectConnectGatewaysAPIClient{gateways: gateways, pageSize: 2}

	got := addAllDirectConnectGatewayNames(mock, map[string]string{"vpc-1": "existing"})

	want := map[string]string{
		"vpc-1":  "existing",
		"dxgw-0": "first",
		"dxgw-1": "second",
		"dxgw-2": "third",
		"dxgw-3": "dxgw-3",
		"dxgw-4": "dxgw-4",
	}
	if len(got) != len(want) {
		t.Errorf("got %d entries, want %d: %v", len(got), len(want), got)
	}
	for id, name := range want {
		if got[id] != name {
			t.Errorf("result[%q] = %q, want %q", id, got[id], name)
		}
	}

	wantTokens := []string{"", "2", "4"}
	if fmt.Sprint(mock.tokens) != fmt.Sprint(wantTokens) {
		t.Errorf("request tokens = %v, want %v", mock.tokens, wantTokens)
	}
}
