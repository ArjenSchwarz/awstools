package helpers

import (
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// fakeEC2Transport answers EC2 query-protocol requests with canned XML keyed
// by Action. No real AWS endpoint or local AWS configuration is used.
type fakeEC2Transport struct {
	responses map[string]string
}

func (f *fakeEC2Transport) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	action := values.Get("Action")
	xml := `<?xml version="1.0" encoding="UTF-8"?><` + action + `Response xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">` +
		f.responses[action] + `</` + action + `Response>`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/xml"}},
		Body:       io.NopCloser(strings.NewReader(xml)),
		Request:    req,
	}, nil
}

func newFakeEC2Client(responses map[string]string) *ec2.Client {
	return ec2.New(ec2.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://ec2.test.invalid"),
		Credentials:  aws.AnonymousCredentials{},
		HTTPClient:   &fakeEC2Transport{responses: responses},
		Retryer:      aws.NopRetryer{},
	})
}

// eniXML builds a DescribeNetworkInterfaces item. Like AWS, it reports the
// "interface" type unless extra sets another one.
func eniXML(id, subnetID, extra string, ips ...string) string {
	if !strings.Contains(extra, "<interfaceType>") {
		extra = `<interfaceType>interface</interfaceType>` + extra
	}
	var ipItems strings.Builder
	for _, ip := range ips {
		ipItems.WriteString(`<item><privateIpAddress>` + ip + `</privateIpAddress></item>`)
	}
	return `<item><networkInterfaceId>` + id + `</networkInterfaceId><subnetId>` + subnetID + `</subnetId>` +
		`<vpcId>vpc-1</vpcId>` + extra + `<privateIpAddressesSet>` + ipItems.String() + `</privateIpAddressesSet></item>`
}

func tagSetXML(kv ...string) string {
	var b strings.Builder
	b.WriteString(`<tagSet>`)
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteString(`<item><key>` + kv[i] + `</key><value>` + kv[i+1] + `</value></item>`)
	}
	b.WriteString(`</tagSet>`)
	return b.String()
}

const (
	eksInstance     = "i-0123456789abcdef0"
	plainInstance   = "i-0123456789abcdef1"
	unknownInstance = "i-0123456789abcdef2"
)

// TestGetVPCUsageOverview_IPAllocationCategories_T1538 runs the production
// overview against fake DescribeInstances/DescribeNetworkInterfaces responses
// and checks the per-subnet IP allocation summary: EKS from the
// aws:eks:cluster-name tag on the instance or ENI, EC2 Instance when there is
// no such tag (including instances whose tags could not be fetched and
// instances with other Kubernetes markers), every private IP counted, and
// AWS-reserved addresses excluded.
func TestGetVPCUsageOverview_IPAllocationCategories_T1538(t *testing.T) {
	responses := map[string]string{
		"DescribeVpcs": `<vpcSet><item><vpcId>vpc-1</vpcId><cidrBlock>10.0.0.0/16</cidrBlock></item></vpcSet>`,
		"DescribeSubnets": `<subnetSet>` +
			`<item><subnetId>subnet-a</subnetId><vpcId>vpc-1</vpcId><cidrBlock>10.0.1.0/27</cidrBlock></item>` +
			`<item><subnetId>subnet-b</subnetId><vpcId>vpc-1</vpcId><cidrBlock>10.0.2.0/28</cidrBlock></item>` +
			`</subnetSet>`,
		"DescribeNetworkInterfaces": `<networkInterfaceSet>` +
			// EKS node: primary plus two secondary IPs, EKS tag on the instance.
			eniXML("eni-eks-node", "subnet-a", `<attachment><instanceId>`+eksInstance+`</instanceId></attachment>`,
				"10.0.1.10", "10.0.1.11", "10.0.1.12") +
			// Second ENI on the same EKS node (VPC CNI secondary ENI).
			eniXML("eni-eks-node-2", "subnet-a", `<attachment><instanceId>`+eksInstance+`</instanceId></attachment>`,
				"10.0.1.13") +
			// Instance with only non-authoritative Kubernetes markers.
			eniXML("eni-plain", "subnet-a", `<attachment><instanceId>`+plainInstance+`</instanceId></attachment>`+
				tagSetXML("cluster.k8s.amazonaws.com/name", "prod"), "10.0.1.20", "10.0.1.21") +
			// Instance missing from DescribeInstances: no evidence, stays EC2.
			eniXML("eni-unknown", "subnet-a", `<attachment><instanceId>`+unknownInstance+`</instanceId></attachment>`,
				"10.0.1.22") +
			eniXML("eni-lambda", "subnet-a", `<interfaceType>lambda</interfaceType><description>AWS Lambda VPC ENI-fn</description>`,
				"10.0.1.30") +
			// ENI carrying the EKS tag itself, not attached to an instance.
			eniXML("eni-eks-tagged", "subnet-b", `<description>Amazon EKS prod</description>`+
				tagSetXML(eksClusterTagKey, "prod"), "10.0.2.6") +
			eniXML("eni-rds", "subnet-b", `<description>RDSNetworkInterface</description>`, "10.0.2.7", "10.0.2.8") +
			`</networkInterfaceSet>`,
		"DescribeInstances": `<reservationSet><item><instancesSet>` +
			`<item><instanceId>` + eksInstance + `</instanceId>` +
			tagSetXML("Name", "node-1", eksClusterTagKey, "prod") + `</item>` +
			`<item><instanceId>` + plainInstance + `</instanceId>` +
			tagSetXML("Name", "web-1", "kubernetes.io/cluster/prod", "owned", "eks:cluster-name", "prod") + `</item>` +
			`</instancesSet></item></reservationSet>`,
	}

	overview := GetVPCUsageOverview(newFakeEC2Client(responses))
	if len(overview.VPCs) != 1 || len(overview.VPCs[0].Subnets) != 2 {
		t.Fatalf("unexpected overview shape: %+v", overview.VPCs)
	}

	want := map[string][]IPAllocationCount{
		"subnet-a": {
			{Category: "EC2 Instance", IPs: 3},
			{Category: "EKS", IPs: 4},
			{Category: "Lambda Function", IPs: 1},
		},
		"subnet-b": {
			{Category: "EKS", IPs: 1},
			{Category: "RDS Database", IPs: 2},
		},
	}
	for _, subnet := range overview.VPCs[0].Subnets {
		got := SummarizeSubnetIPAllocations(subnet)
		if !slices.Equal(got, want[subnet.ID]) {
			t.Errorf("%s allocations:\ngot  %+v\nwant %+v", subnet.ID, got, want[subnet.ID])
		}
	}

	// The EKS cluster name is kept on the IP details; the detailed usage type
	// is unchanged.
	for _, ip := range overview.VPCs[0].Subnets[0].IPDetails {
		if ip.IPAddress == "10.0.1.11" {
			if ip.EKSCluster != "prod" || ip.UsageType != "EC2 Instance" {
				t.Errorf("10.0.1.11 = %+v, want EKSCluster prod and UsageType EC2 Instance", ip)
			}
		}
	}

	// Totals still split reserved from service IPs: 2 subnets x 5 reserved,
	// 8 + 3 service IPs.
	if overview.Summary.AWSReservedIPs != 10 || overview.Summary.ServiceIPs != 11 {
		t.Errorf("summary reserved/service = %d/%d, want 10/11", overview.Summary.AWSReservedIPs, overview.Summary.ServiceIPs)
	}
}

func TestIPAllocationCategory_T1538(t *testing.T) {
	tests := map[string]struct {
		ip           IPAddressInfo
		wantCategory string
		wantService  bool
	}{
		"aws reserved is not a service": {ip: IPAddressInfo{UsageType: awsReservedUsageType}, wantService: false},
		"eks cluster tag":               {ip: IPAddressInfo{UsageType: "EC2 Instance", EKSCluster: "prod"}, wantCategory: "EKS", wantService: true},
		"ec2 without tag":               {ip: IPAddressInfo{UsageType: "EC2 Instance"}, wantCategory: "EC2 Instance", wantService: true},
		"other usage types unchanged":   {ip: IPAddressInfo{UsageType: "NAT Gateway"}, wantCategory: "NAT Gateway", wantService: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotCategory, gotService := IPAllocationCategory(tc.ip)
			if gotCategory != tc.wantCategory || gotService != tc.wantService {
				t.Errorf("IPAllocationCategory() = (%q, %v), want (%q, %v)", gotCategory, gotService, tc.wantCategory, tc.wantService)
			}
		})
	}
}
