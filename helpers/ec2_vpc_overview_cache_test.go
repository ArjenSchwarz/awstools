package helpers

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// countingEC2Transport answers EC2 query-protocol requests with canned XML
// and records how often each Action was called. No real AWS endpoint or
// local AWS configuration is involved.
type countingEC2Transport struct {
	mu        sync.Mutex
	calls     map[string]int
	responses map[string]string
}

func (c *countingEC2Transport) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	action := values.Get("Action")

	c.mu.Lock()
	c.calls[action]++
	c.mu.Unlock()

	inner, ok := c.responses[action]
	if !ok {
		inner = ""
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?><` + action + `Response xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">` +
		inner + `</` + action + `Response>`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/xml"}},
		Body:       io.NopCloser(strings.NewReader(xml)),
		Request:    req,
	}, nil
}

func newCountingEC2Client(transport *countingEC2Transport) *ec2.Client {
	return ec2.New(ec2.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://ec2.test.invalid"),
		Credentials:  aws.AnonymousCredentials{},
		HTTPClient:   transport,
		Retryer:      aws.NopRetryer{},
	})
}

// TestGetVPCUsageOverview_BuildsENICacheOnce_T1597 verifies that the ENI
// lookup cache (VPC endpoints, instances, NAT gateways, TGW attachments) is
// fetched once per overview rather than once per IPv4 subnet.
func TestGetVPCUsageOverview_BuildsENICacheOnce_T1597(t *testing.T) {
	transport := &countingEC2Transport{
		calls: map[string]int{},
		responses: map[string]string{
			"DescribeVpcs": `<vpcSet><item><vpcId>vpc-1</vpcId><cidrBlock>10.0.0.0/16</cidrBlock></item></vpcSet>`,
			"DescribeSubnets": `<subnetSet>` +
				`<item><subnetId>subnet-a</subnetId><vpcId>vpc-1</vpcId><cidrBlock>10.0.1.0/28</cidrBlock></item>` +
				`<item><subnetId>subnet-b</subnetId><vpcId>vpc-1</vpcId><cidrBlock>10.0.2.0/28</cidrBlock></item>` +
				`<item><subnetId>subnet-c</subnetId><vpcId>vpc-1</vpcId><cidrBlock>10.0.3.0/28</cidrBlock></item>` +
				`<item><subnetId>subnet-v6</subnetId><vpcId>vpc-1</vpcId><ipv6Native>true</ipv6Native>` +
				`<ipv6CidrBlockAssociationSet><item><ipv6CidrBlock>2600:1f18::/64</ipv6CidrBlock></item></ipv6CidrBlockAssociationSet></item>` +
				`</subnetSet>`,
			"DescribeNetworkInterfaces": `<networkInterfaceSet>` +
				`<item><networkInterfaceId>eni-a</networkInterfaceId><subnetId>subnet-a</subnetId><vpcId>vpc-1</vpcId>` +
				`<attachment><instanceId>i-0123456789abcdef0</instanceId></attachment>` +
				`<privateIpAddressesSet><item><privateIpAddress>10.0.1.10</privateIpAddress></item></privateIpAddressesSet></item>` +
				`<item><networkInterfaceId>eni-b</networkInterfaceId><subnetId>subnet-b</subnetId><vpcId>vpc-1</vpcId>` +
				`<privateIpAddressesSet><item><privateIpAddress>10.0.2.10</privateIpAddress></item></privateIpAddressesSet></item>` +
				`</networkInterfaceSet>`,
			"DescribeInstances": `<reservationSet><item><instancesSet><item><instanceId>i-0123456789abcdef0</instanceId>` +
				`<tagSet><item><key>Name</key><value>web-1</value></item></tagSet></item></instancesSet></item></reservationSet>`,
		},
	}

	overview := GetVPCUsageOverview(newCountingEC2Client(transport))

	for _, action := range []string{
		"DescribeVpcEndpoints",
		"DescribeInstances",
		"DescribeNatGateways",
		"DescribeTransitGatewayVpcAttachments",
	} {
		if got := transport.calls[action]; got != 1 {
			t.Errorf("%s called %d times for 3 IPv4 subnets, want 1", action, got)
		}
	}

	if overview.Summary.TotalSubnets != 4 {
		t.Fatalf("TotalSubnets = %d, want 4", overview.Summary.TotalSubnets)
	}

	// The shared cache must still resolve ENI details in every subnet.
	found := map[string]string{}
	for _, subnet := range overview.VPCs[0].Subnets {
		for _, ip := range subnet.IPDetails {
			if ip.UsageType != "RESERVED BY AWS" {
				found[ip.IPAddress] = ip.AttachmentInfo
			}
		}
	}
	if len(found) != 2 {
		t.Fatalf("expected ENI IPs from two subnets, got %v", found)
	}
	if !strings.Contains(found["10.0.1.10"], "web-1") {
		t.Errorf("10.0.1.10 attachment = %q, want instance name web-1 from cache", found["10.0.1.10"])
	}
}

// TestGetVPCUsageOverview_IPv6OnlySkipsENICache_T1597 verifies that an
// overview with only IPv6-only subnets, which skip per-IP analysis, does
// not build the ENI lookup cache at all.
func TestGetVPCUsageOverview_IPv6OnlySkipsENICache_T1597(t *testing.T) {
	transport := &countingEC2Transport{
		calls: map[string]int{},
		responses: map[string]string{
			"DescribeVpcs": `<vpcSet><item><vpcId>vpc-1</vpcId></item></vpcSet>`,
			"DescribeSubnets": `<subnetSet><item><subnetId>subnet-v6</subnetId><vpcId>vpc-1</vpcId><ipv6Native>true</ipv6Native>` +
				`<ipv6CidrBlockAssociationSet><item><ipv6CidrBlock>2600:1f18::/64</ipv6CidrBlock></item></ipv6CidrBlockAssociationSet></item></subnetSet>`,
			"DescribeNetworkInterfaces": `<networkInterfaceSet><item><networkInterfaceId>eni-a</networkInterfaceId>` +
				`<subnetId>subnet-v6</subnetId><vpcId>vpc-1</vpcId></item></networkInterfaceSet>`,
		},
	}

	GetVPCUsageOverview(newCountingEC2Client(transport))

	for _, action := range []string{"DescribeVpcEndpoints", "DescribeNatGateways", "DescribeTransitGatewayVpcAttachments"} {
		if got := transport.calls[action]; got != 0 {
			t.Errorf("%s called %d times for IPv6-only overview, want 0", action, got)
		}
	}
}
