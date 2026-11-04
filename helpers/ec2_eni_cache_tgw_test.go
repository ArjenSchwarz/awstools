package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Regression tests for T-1571: the ENI cache stored a single TGW attachment ID
// per VPC, so with several TGW attachments in one VPC the last paginated
// attachment overwrote the others and every TGW ENI in the VPC reported it.

func tgwENI(id, subnetID, description string) types.NetworkInterface {
	return types.NetworkInterface{
		NetworkInterfaceId: aws.String(id),
		InterfaceType:      types.NetworkInterfaceTypeTransitGateway,
		VpcId:              aws.String("vpc-shared"),
		SubnetId:           aws.String(subnetID),
		Description:        aws.String(description),
	}
}

func tgwAttachment(id string, subnetIDs ...string) types.TransitGatewayVpcAttachment {
	return types.TransitGatewayVpcAttachment{
		TransitGatewayAttachmentId: aws.String(id),
		VpcId:                      aws.String("vpc-shared"),
		SubnetIds:                  subnetIDs,
	}
}

func TestENILookupCache_TransitGatewayAttachmentPerENI_T1571(t *testing.T) {
	tests := map[string]struct {
		attachments []types.TransitGatewayVpcAttachment
		eni         types.NetworkInterface
		want        string
	}{
		"different subnets, first attachment": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-a"), tgwAttachment("tgw-attach-bbb", "subnet-b")},
			eni:         tgwENI("eni-a", "subnet-a", ""),
			want:        "tgw-attach-aaa",
		},
		"different subnets, second attachment": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-a"), tgwAttachment("tgw-attach-bbb", "subnet-b")},
			eni:         tgwENI("eni-b", "subnet-b", ""),
			want:        "tgw-attach-bbb",
		},
		"shared subnet resolved from description": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-shared"), tgwAttachment("tgw-attach-bbb", "subnet-shared")},
			eni:         tgwENI("eni-a", "subnet-shared", "Network Interface for Transit Gateway Attachment tgw-attach-aaa"),
			want:        "tgw-attach-aaa",
		},
		"shared subnet without description is not guessed": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-shared"), tgwAttachment("tgw-attach-bbb", "subnet-shared")},
			eni:         tgwENI("eni-a", "subnet-shared", ""),
			want:        "",
		},
		"description naming uncached attachment is not replaced by subnet match": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-a"), tgwAttachment("tgw-attach-bbb", "subnet-b")},
			eni:         tgwENI("eni-b", "subnet-b", "Network Interface for Transit Gateway Attachment tgw-attach-zzz"),
			want:        "",
		},
		"description without attachment ID falls back to subnet": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-a"), tgwAttachment("tgw-attach-bbb", "subnet-b")},
			eni:         tgwENI("eni-b", "subnet-b", "custom description"),
			want:        "tgw-attach-bbb",
		},
		"no attachment for subnet": {
			attachments: []types.TransitGatewayVpcAttachment{tgwAttachment("tgw-attach-aaa", "subnet-a")},
			eni:         tgwENI("eni-c", "subnet-c", ""),
			want:        "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// One attachment per page reproduces the overwrite across pages.
			mock := &mockDescribeTGWAttachmentsClient{attachments: tc.attachments, pageSize: 1}
			cache := &ENILookupCache{TransitGatewayAttachments: make(map[string][]types.TransitGatewayVpcAttachment)}
			cache.batchFetchTransitGateways(mock, map[string]bool{"vpc-shared": true})

			if got := GetAttachmentFromCache(tc.eni, cache); got != tc.want {
				t.Errorf("GetAttachmentFromCache() = %q, want %q", got, tc.want)
			}

			wantDetails := tc.want
			if wantDetails == "" {
				wantDetails = "Unknown Transit Gateway"
			}
			if got := getENIAttachmentDetailsOptimized(tc.eni, cache); got != wantDetails {
				t.Errorf("getENIAttachmentDetailsOptimized() = %q, want %q", got, wantDetails)
			}

			if _, gotID := getResourceNameAndID(tc.eni, cache); gotID != tc.want {
				t.Errorf("getResourceNameAndID() resource ID = %q, want %q", gotID, tc.want)
			}
		})
	}
}

// failingPageTGWClient serves pages from the wrapped mock but fails on page
// failOnCall (1-based).
type failingPageTGWClient struct {
	mockDescribeTGWAttachmentsClient
	failOnCall int
}

func (m *failingPageTGWClient) DescribeTransitGatewayVpcAttachments(ctx context.Context, input *ec2.DescribeTransitGatewayVpcAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	if m.callCount+1 == m.failOnCall {
		m.callCount++
		return nil, errors.New("throttled")
	}
	return m.mockDescribeTGWAttachmentsClient.DescribeTransitGatewayVpcAttachments(ctx, input, optFns...)
}

// TestENILookupCache_TransitGatewayLaterPageError_T1571 verifies that a failed
// later page does not leave a partial attachment list in the cache. Otherwise
// the subnet fallback would see one attachment on the shared subnet and report
// it as the owner, although the competing attachment was on the failed page.
func TestENILookupCache_TransitGatewayLaterPageError_T1571(t *testing.T) {
	mock := &failingPageTGWClient{
		mockDescribeTGWAttachmentsClient: mockDescribeTGWAttachmentsClient{
			attachments: []types.TransitGatewayVpcAttachment{
				tgwAttachment("tgw-attach-aaa", "subnet-shared"),
				tgwAttachment("tgw-attach-bbb", "subnet-shared"),
			},
			pageSize: 1,
		},
		failOnCall: 2,
	}
	cache := &ENILookupCache{TransitGatewayAttachments: make(map[string][]types.TransitGatewayVpcAttachment)}
	cache.batchFetchTransitGateways(mock, map[string]bool{"vpc-shared": true})

	if mock.callCount != 2 {
		t.Fatalf("DescribeTransitGatewayVpcAttachments called %d times, want 2", mock.callCount)
	}

	eni := tgwENI("eni-b", "subnet-shared", "")
	if got := GetAttachmentFromCache(eni, cache); got != "" {
		t.Errorf("GetAttachmentFromCache() = %q, want \"\" (owner inferred from incomplete cache)", got)
	}
	if got := getENIAttachmentDetailsOptimized(eni, cache); got != "Unknown Transit Gateway" {
		t.Errorf("getENIAttachmentDetailsOptimized() = %q, want %q", got, "Unknown Transit Gateway")
	}
	if _, gotID := getResourceNameAndID(eni, cache); gotID != "" {
		t.Errorf("getResourceNameAndID() resource ID = %q, want \"\"", gotID)
	}
}
