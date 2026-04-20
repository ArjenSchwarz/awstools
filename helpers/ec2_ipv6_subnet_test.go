package helpers

import (
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestGenerateIPRange_IPv6CIDRRejected verifies that generateIPRange refuses
// to enumerate IPv6 CIDRs. Prior to T-774 it would attempt to iterate every
// address — for a /64 that is 2^64 addresses, which exhausts memory.
func TestGenerateIPRange_IPv6CIDRRejected(t *testing.T) {
	done := make(chan struct{})
	var ips []net.IP
	var err error

	go func() {
		defer close(done)
		ips, err = generateIPRange("2001:db8::/64")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("generateIPRange did not return within 2s — likely enumerating the full IPv6 range")
	}

	if err == nil {
		t.Fatalf("expected an error for IPv6 CIDR, got nil (returned %d IPs)", len(ips))
	}
}

// TestGenerateIPRange_IPv4StillWorks ensures the IPv4 code path is unchanged.
func TestGenerateIPRange_IPv4StillWorks(t *testing.T) {
	ips, err := generateIPRange("10.0.0.0/30")
	if err != nil {
		t.Fatalf("unexpected error for IPv4 /30: %v", err)
	}
	// /30 has 4 addresses
	if len(ips) != 4 {
		t.Fatalf("expected 4 IPs, got %d", len(ips))
	}
	expected := []string{"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"}
	for i, want := range expected {
		if ips[i].String() != want {
			t.Errorf("ip[%d] = %s, want %s", i, ips[i].String(), want)
		}
	}
}

// TestCalculateSubnetStats_IPv6DoesNotOverflow verifies that calculateSubnetStats
// returns a sensible (non-panicking, non-overflowing) result for IPv6 CIDRs.
// Prior to T-774 it used `1 << uint(bits-ones)` which is undefined for shifts
// >= 64 and produces 0 on 64-bit platforms.
func TestCalculateSubnetStats_IPv6DoesNotOverflow(t *testing.T) {
	total, available, err := calculateSubnetStats("2001:db8::/64")
	if err != nil {
		t.Fatalf("unexpected error for IPv6 /64: %v", err)
	}
	if total <= 0 {
		t.Errorf("expected total IPs > 0 for IPv6 /64, got %d", total)
	}
	if available < 0 {
		t.Errorf("available IPs should not be negative, got %d", available)
	}
}

// TestAnalyzeSubnetIPUsage_IPv6NativeSubnetDoesNotPanic verifies that an
// IPv6-native subnet (no IPv4 CIDR) is handled gracefully. Prior to T-774
// analyzeSubnetIPUsage returned "subnet has no CIDR block" which the caller
// turned into a panic.
func TestAnalyzeSubnetIPUsage_IPv6NativeSubnetDoesNotPanic(t *testing.T) {
	subnet := types.Subnet{
		SubnetId:   aws.String("subnet-ipv6-only"),
		VpcId:      aws.String("vpc-12345"),
		Ipv6Native: aws.Bool(true),
		Ipv6CidrBlockAssociationSet: []types.SubnetIpv6CidrBlockAssociation{
			{
				Ipv6CidrBlock: aws.String("2001:db8::/64"),
				Ipv6CidrBlockState: &types.SubnetCidrBlockState{
					State: types.SubnetCidrBlockStateCodeAssociated,
				},
			},
		},
	}

	done := make(chan struct{})
	var err error
	var panicValue any

	go func() {
		defer close(done)
		defer func() {
			panicValue = recover()
		}()
		_, _, _, _, _, err = analyzeSubnetIPUsage(subnet, nil, nil)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("analyzeSubnetIPUsage did not return within 2s — likely enumerating the full IPv6 range")
	}

	if panicValue != nil {
		t.Fatalf("analyzeSubnetIPUsage panicked on IPv6-native subnet: %v", panicValue)
	}
	if err != nil {
		t.Errorf("IPv6-native subnet should be handled without error, got: %v", err)
	}
}

// TestAnalyzeSubnetIPUsage_DualStackOnlyEnumeratesIPv4 verifies that a subnet
// with both an IPv4 CIDR and an IPv6 CIDR association enumerates only IPv4
// addresses. IPv6 is too large to enumerate and should be ignored for per-IP
// analysis.
func TestAnalyzeSubnetIPUsage_DualStackOnlyEnumeratesIPv4(t *testing.T) {
	subnet := types.Subnet{
		SubnetId:  aws.String("subnet-dual"),
		VpcId:     aws.String("vpc-12345"),
		CidrBlock: aws.String("10.0.0.0/28"), // 16 IPs
		Ipv6CidrBlockAssociationSet: []types.SubnetIpv6CidrBlockAssociation{
			{
				Ipv6CidrBlock: aws.String("2001:db8::/64"),
				Ipv6CidrBlockState: &types.SubnetCidrBlockState{
					State: types.SubnetCidrBlockStateCodeAssociated,
				},
			},
		},
	}

	done := make(chan struct{})
	var usedIPs, availableIPs, awsReserved, serviceIPs int
	var err error

	go func() {
		defer close(done)
		_, usedIPs, availableIPs, awsReserved, serviceIPs, err = analyzeSubnetIPUsage(subnet, nil, nil)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("analyzeSubnetIPUsage did not return within 2s — likely enumerating IPv6 addresses")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// /28 has 16 IPs; 5 are AWS-reserved (first 4 + broadcast), 0 are service.
	if usedIPs != 5 {
		t.Errorf("expected 5 used IPs for /28 reserved, got %d", usedIPs)
	}
	if availableIPs != 11 {
		t.Errorf("expected 11 available IPs, got %d", availableIPs)
	}
	if awsReserved != 5 {
		t.Errorf("expected 5 AWS-reserved IPs, got %d", awsReserved)
	}
	if serviceIPs != 0 {
		t.Errorf("expected 0 service IPs, got %d", serviceIPs)
	}
}
