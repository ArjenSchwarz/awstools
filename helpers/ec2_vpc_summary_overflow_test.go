package helpers

import (
	"math"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestSaturatingAdd_DoesNotOverflow verifies that summing two "effectively
// unlimited" sentinel values (math.MaxInt, used for IPv6-only subnet capacity
// per T-774) does not wrap to a negative number. Naive integer addition of two
// math.MaxInt values overflows and produces a negative result (T-1234).
func TestSaturatingAdd_DoesNotOverflow(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{"two sentinels saturate", math.MaxInt, math.MaxInt, math.MaxInt},
		{"sentinel plus normal saturates", math.MaxInt, 256, math.MaxInt},
		{"normal plus sentinel saturates", 256, math.MaxInt, math.MaxInt},
		{"normal addition unchanged", 100, 200, 300},
		{"zero values", 0, 0, 0},
		{"just below overflow", math.MaxInt - 5, 5, math.MaxInt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := saturatingAdd(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("saturatingAdd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
			if got < 0 {
				t.Errorf("saturatingAdd(%d, %d) = %d is negative; sentinel summation overflowed", tt.a, tt.b, got)
			}
		})
	}
}

// TestSummarizeVPCUsage_TwoIPv6OnlySubnetsNoOverflow reproduces the T-1234 bug:
// a VPC with two IPv6-only subnets, each reporting math.MaxInt total/available
// IPs, must not produce a negative aggregate summary. Before the fix the inline
// `summary.TotalIPs += totalIPs` arithmetic wrapped to a negative value.
func TestSummarizeVPCUsage_TwoIPv6OnlySubnetsNoOverflow(t *testing.T) {
	vpcs := []VPCUsageInfo{
		{
			ID:   "vpc-ipv6",
			Name: "ipv6-vpc",
			CIDR: "2001:db8::/56",
			Subnets: []SubnetUsageInfo{
				{
					ID:           "subnet-ipv6-a",
					CIDR:         "2001:db8:0:1::/64",
					TotalIPs:     math.MaxInt,
					AvailableIPs: math.MaxInt - 5,
					UsedIPs:      5,
				},
				{
					ID:           "subnet-ipv6-b",
					CIDR:         "2001:db8:0:2::/64",
					TotalIPs:     math.MaxInt,
					AvailableIPs: math.MaxInt - 5,
					UsedIPs:      5,
				},
			},
		},
	}

	summary := SummarizeVPCUsage(vpcs)

	if summary.TotalIPs < 0 {
		t.Errorf("TotalIPs = %d is negative; two IPv6-only subnets overflowed the summary", summary.TotalIPs)
	}
	if summary.AvailableIPs < 0 {
		t.Errorf("AvailableIPs = %d is negative; two IPv6-only subnets overflowed the summary", summary.AvailableIPs)
	}
	if summary.TotalIPs != math.MaxInt {
		t.Errorf("TotalIPs = %d, want math.MaxInt (effectively unlimited)", summary.TotalIPs)
	}
	if summary.AvailableIPs != math.MaxInt {
		t.Errorf("AvailableIPs = %d, want math.MaxInt (effectively unlimited)", summary.AvailableIPs)
	}
	if summary.TotalVPCs != 1 {
		t.Errorf("TotalVPCs = %d, want 1", summary.TotalVPCs)
	}
	if summary.TotalSubnets != 2 {
		t.Errorf("TotalSubnets = %d, want 2", summary.TotalSubnets)
	}
	if summary.UsedIPs != 10 {
		t.Errorf("UsedIPs = %d, want 10", summary.UsedIPs)
	}
}

// TestSummarizeVPCUsage_NormalIPv4Unchanged ensures the saturating summary
// computes ordinary IPv4 totals exactly as before for non-sentinel values.
func TestSummarizeVPCUsage_NormalIPv4Unchanged(t *testing.T) {
	vpcs := []VPCUsageInfo{
		{
			ID: "vpc-ipv4",
			Subnets: []SubnetUsageInfo{
				{
					ID:           "subnet-a",
					CIDR:         "10.0.0.0/28", // 16 IPs
					TotalIPs:     16,
					AvailableIPs: 11,
					UsedIPs:      5,
					IPDetails: []IPAddressInfo{
						{IPAddress: "10.0.0.0", UsageType: "RESERVED BY AWS"},
						{IPAddress: "10.0.0.1", UsageType: "RESERVED BY AWS"},
						{IPAddress: "10.0.0.5", UsageType: "EC2 Instance"},
					},
				},
				{
					ID:           "subnet-b",
					CIDR:         "10.0.1.0/28", // 16 IPs
					TotalIPs:     16,
					AvailableIPs: 11,
					UsedIPs:      5,
				},
			},
		},
	}

	summary := SummarizeVPCUsage(vpcs)

	if summary.TotalIPs != 32 {
		t.Errorf("TotalIPs = %d, want 32", summary.TotalIPs)
	}
	if summary.AvailableIPs != 22 {
		t.Errorf("AvailableIPs = %d, want 22", summary.AvailableIPs)
	}
	if summary.UsedIPs != 10 {
		t.Errorf("UsedIPs = %d, want 10", summary.UsedIPs)
	}
	if summary.AWSReservedIPs != 2 {
		t.Errorf("AWSReservedIPs = %d, want 2", summary.AWSReservedIPs)
	}
	if summary.ServiceIPs != 1 {
		t.Errorf("ServiceIPs = %d, want 1", summary.ServiceIPs)
	}
}

// compile-time guard that the types referenced above are the real ones.
var _ = types.Tag{}
var _ = aws.String
