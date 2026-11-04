package helpers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Regression tests for T-1706: `vpc ip-finder` accepted IPv6 addresses but
// always searched with the IPv4-only addresses.private-ip-address filter and
// only compared IPv4 fields, so IPv6 addresses attached to ENIs were reported
// as not found.

// filteringENIClient answers DescribeNetworkInterfaces like EC2 does for the
// two IP finder filters, so tests exercise the real filter selection.
type filteringENIClient struct {
	interfaces []types.NetworkInterface
	filters    []types.Filter
}

func (m *filteringENIClient) DescribeNetworkInterfaces(_ context.Context, input *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	m.filters = input.Filters
	out := &ec2.DescribeNetworkInterfacesOutput{}
	for _, eni := range m.interfaces {
		if eniMatchesFilters(eni, input.Filters) {
			out.NetworkInterfaces = append(out.NetworkInterfaces, eni)
		}
	}
	return out, nil
}

func eniMatchesFilters(eni types.NetworkInterface, filters []types.Filter) bool {
	for _, f := range filters {
		for _, v := range f.Values {
			switch aws.ToString(f.Name) {
			case "addresses.private-ip-address":
				for _, a := range eni.PrivateIpAddresses {
					if aws.ToString(a.PrivateIpAddress) == v {
						return true
					}
				}
			case "ipv6-addresses.ipv6-address":
				for _, a := range eni.Ipv6Addresses {
					if aws.ToString(a.Ipv6Address) == v {
						return true
					}
				}
			}
		}
	}
	return false
}

func dualStackENI(id, primaryIPv4, secondaryIPv4 string, ipv6 ...types.NetworkInterfaceIpv6Address) types.NetworkInterface {
	return types.NetworkInterface{
		NetworkInterfaceId: aws.String(id),
		PrivateIpAddress:   aws.String(primaryIPv4),
		PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
			{PrivateIpAddress: aws.String(primaryIPv4), Primary: aws.Bool(true)},
			{PrivateIpAddress: aws.String(secondaryIPv4), Primary: aws.Bool(false)},
		},
		Ipv6Addresses: ipv6,
	}
}

func TestFindENIsByIP_IPv4AndIPv6(t *testing.T) {
	enis := []types.NetworkInterface{
		dualStackENI("eni-primary6", "10.0.1.10", "10.0.1.11",
			types.NetworkInterfaceIpv6Address{Ipv6Address: aws.String("2001:db8::10"), IsPrimaryIpv6: aws.Bool(true)},
			types.NetworkInterfaceIpv6Address{Ipv6Address: aws.String("2001:db8::11"), IsPrimaryIpv6: aws.Bool(false)},
		),
		dualStackENI("eni-noprimary6", "10.0.2.10", "10.0.2.11",
			types.NetworkInterfaceIpv6Address{Ipv6Address: aws.String("2001:db8::20")},
		),
	}

	tests := map[string]struct {
		input         string
		wantIP        string
		wantFilter    string
		wantENI       string
		wantSecondary bool
	}{
		"IPv4 primary": {
			input: "10.0.1.10", wantIP: "10.0.1.10",
			wantFilter: "addresses.private-ip-address", wantENI: "eni-primary6", wantSecondary: false,
		},
		"IPv4 secondary": {
			input: "10.0.1.11", wantIP: "10.0.1.11",
			wantFilter: "addresses.private-ip-address", wantENI: "eni-primary6", wantSecondary: true,
		},
		"IPv6 flagged primary": {
			input: "2001:db8::10", wantIP: "2001:db8::10",
			wantFilter: "ipv6-addresses.ipv6-address", wantENI: "eni-primary6", wantSecondary: false,
		},
		"IPv6 not flagged primary": {
			input: "2001:db8::11", wantIP: "2001:db8::11",
			wantFilter: "ipv6-addresses.ipv6-address", wantENI: "eni-primary6", wantSecondary: true,
		},
		"IPv6 on ENI without primary IPv6": {
			input: "2001:db8::20", wantIP: "2001:db8::20",
			wantFilter: "ipv6-addresses.ipv6-address", wantENI: "eni-noprimary6", wantSecondary: true,
		},
		"IPv6 uppercase expanded input is canonicalised": {
			input: "2001:DB8:0:0:0:0:0:10", wantIP: "2001:db8::10",
			wantFilter: "ipv6-addresses.ipv6-address", wantENI: "eni-primary6", wantSecondary: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &filteringENIClient{interfaces: enis}

			gotIP, gotENIs := findENIsByIP(client, tc.input)

			if gotIP != tc.wantIP {
				t.Errorf("findENIsByIP() ip = %q, want %q", gotIP, tc.wantIP)
			}
			if len(client.filters) != 1 || aws.ToString(client.filters[0].Name) != tc.wantFilter {
				t.Fatalf("filters = %+v, want single %q filter", client.filters, tc.wantFilter)
			}
			if got := client.filters[0].Values; len(got) != 1 || got[0] != tc.wantIP {
				t.Errorf("filter values = %v, want [%s]", got, tc.wantIP)
			}
			if len(gotENIs) != 1 {
				t.Fatalf("findENIsByIP() returned %d ENIs, want 1", len(gotENIs))
			}

			results := buildBaseIPFinderResults(gotIP, gotENIs)
			if len(results) != 1 || !results[0].Found {
				t.Fatalf("buildBaseIPFinderResults() = %+v, want one found result", results)
			}
			if got := aws.ToString(results[0].ENI.NetworkInterfaceId); got != tc.wantENI {
				t.Errorf("ENI = %s, want %s", got, tc.wantENI)
			}
			if results[0].IsSecondaryIP != tc.wantSecondary {
				t.Errorf("IsSecondaryIP = %v, want %v", results[0].IsSecondaryIP, tc.wantSecondary)
			}
		})
	}
}

func TestFindENIsByIP_IPv6NotFound(t *testing.T) {
	client := &filteringENIClient{interfaces: []types.NetworkInterface{dualStackENI("eni-v4only", "10.0.3.10", "10.0.3.11")}}

	gotIP, gotENIs := findENIsByIP(client, "2001:db8::99")

	if gotIP != "2001:db8::99" {
		t.Errorf("findENIsByIP() ip = %q, want 2001:db8::99", gotIP)
	}
	if len(gotENIs) != 0 {
		t.Errorf("findENIsByIP() returned %d ENIs, want 0", len(gotENIs))
	}
}
