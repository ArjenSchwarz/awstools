package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
)

// reservedIPs returns the five AWS-reserved address details of a subnet.
func reservedIPs(prefix string, last string) []helpers.IPAddressInfo {
	ips := make([]helpers.IPAddressInfo, 0, 5)
	for _, suffix := range []string{"0", "1", "2", "3", last} {
		ips = append(ips, helpers.IPAddressInfo{IPAddress: prefix + suffix, UsageType: "RESERVED BY AWS"})
	}
	return ips
}

// allocationVPCs returns two VPCs with several subnets and service types,
// including an EKS node ENI with secondary IPs and an IPv6-only subnet.
func allocationVPCs() []helpers.VPCUsageInfo {
	subnetA := append(reservedIPs("10.0.1.", "255"),
		helpers.IPAddressInfo{IPAddress: "10.0.1.10", UsageType: "EC2 Instance", AttachmentInfo: "i-eks", EKSCluster: "prod"},
		helpers.IPAddressInfo{IPAddress: "10.0.1.11", UsageType: "EC2 Instance", AttachmentInfo: "i-eks", EKSCluster: "prod"},
		helpers.IPAddressInfo{IPAddress: "10.0.1.12", UsageType: "EC2 Instance", AttachmentInfo: "i-eks", EKSCluster: "prod"},
		helpers.IPAddressInfo{IPAddress: "10.0.1.20", UsageType: "EC2 Instance", AttachmentInfo: "i-web"},
		helpers.IPAddressInfo{IPAddress: "10.0.1.30", UsageType: "Lambda Function", AttachmentInfo: "AWS Lambda VPC ENI"},
	)
	subnetC := append(reservedIPs("10.0.3.", "255"),
		helpers.IPAddressInfo{IPAddress: "10.0.3.7", UsageType: "RDS Database", AttachmentInfo: "RDSNetworkInterface"},
	)
	subnetB := append(reservedIPs("10.1.1.", "255"),
		helpers.IPAddressInfo{IPAddress: "10.1.1.20", UsageType: "NAT Gateway", AttachmentInfo: "nat-0456"},
	)
	return []helpers.VPCUsageInfo{
		{
			ID: "vpc-111",
			Subnets: []helpers.SubnetUsageInfo{
				{ID: "subnet-aaa", CIDR: "10.0.1.0/24", VPCId: "vpc-111", TotalIPs: 256, UsedIPs: 10, AvailableIPs: 246, IPDetails: subnetA},
				{ID: "subnet-ccc", CIDR: "10.0.3.0/24", VPCId: "vpc-111", TotalIPs: 256, UsedIPs: 6, AvailableIPs: 250, IPDetails: subnetC},
				{ID: "subnet-v6", CIDR: "2600:1f18::/64", VPCId: "vpc-111", TotalIPs: math.MaxInt, AvailableIPs: math.MaxInt},
			},
		},
		{
			ID: "vpc-222",
			Subnets: []helpers.SubnetUsageInfo{
				{ID: "subnet-bbb", CIDR: "10.1.1.0/24", VPCId: "vpc-222", TotalIPs: 256, UsedIPs: 6, AvailableIPs: 250, IPDetails: subnetB},
			},
		},
	}
}

// allocationRow formats one IP allocation summary row for comparison.
func allocationRow(row map[string]any) string {
	return fmt.Sprintf("%v|%v|%v|%v", row[subnetColumn], row[allocationColumn], row[categoryColumn], row[ipAddressesColumn])
}

var (
	vpc111AllocationRows = []string{
		"subnet-aaa|Service|EC2 Instance|1",
		"subnet-aaa|Service|EKS|3",
		"subnet-aaa|Service|Lambda Function|1",
		"subnet-aaa|AWS Reserved||5",
		"subnet-aaa|Available||246",
		"subnet-ccc|Service|RDS Database|1",
		"subnet-ccc|AWS Reserved||5",
		"subnet-ccc|Available||250",
	}
	vpc222AllocationRows = []string{
		"subnet-bbb|Service|NAT Gateway|1",
		"subnet-bbb|AWS Reserved||5",
		"subnet-bbb|Available||250",
	}
)

// TestBuildVPCOverviewTables_IPAllocationSummary_T1538 checks the per-VPC IP
// allocation summary: IPs (not ENIs) per service category in a fixed order,
// AWS-reserved and available addresses on their own rows, IPv6-only subnets
// left out, --vpc filtering, and the existing tables and totals unchanged.
func TestBuildVPCOverviewTables_IPAllocationSummary_T1538(t *testing.T) {
	tests := map[string]struct {
		vpcFilter  string
		wantTitles []string
		wantRows   map[string][]string
		wantTotals map[string]any
	}{
		"all vpcs": {
			wantTitles: []string{
				"Subnet Overview for vpc-111 in account " + testAccount,
				"Subnet Overview for vpc-222 in account " + testAccount,
				"IP Allocation Summary for vpc-111 in account " + testAccount,
				"IP Allocation Summary for vpc-222 in account " + testAccount,
				"IP Details for subnet subnet-aaa in VPC vpc-111",
				"IP Details for subnet subnet-ccc in VPC vpc-111",
				"IP Details for subnet subnet-bbb in VPC vpc-222",
				"VPC Usage Summary for account " + testAccount,
			},
			wantRows: map[string][]string{
				"IP Allocation Summary for vpc-111 in account " + testAccount: vpc111AllocationRows,
				"IP Allocation Summary for vpc-222 in account " + testAccount: vpc222AllocationRows,
			},
			wantTotals: map[string]any{"  - AWS Reserved IPs": 15, "  - Service IPs": 7, "Total Subnets": 4},
		},
		"vpc filter": {
			vpcFilter: "vpc-222",
			wantTitles: []string{
				"Subnet Overview for vpc-222 in account " + testAccount,
				"IP Allocation Summary for vpc-222 in account " + testAccount,
				"IP Details for subnet subnet-bbb in VPC vpc-222",
				"VPC Usage Summary for vpc-222 in account " + testAccount,
			},
			wantRows: map[string][]string{
				"IP Allocation Summary for vpc-222 in account " + testAccount: vpc222AllocationRows,
			},
			wantTotals: map[string]any{"  - AWS Reserved IPs": 5, "  - Service IPs": 1, "Total Subnets": 1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			configureOutput(t, "table", "")
			tables := buildVPCOverviewTables(allocationVPCs(), nil, testAccount, tc.vpcFilter)

			var gotTitles []string
			for _, table := range tables {
				title := table.Settings.Title
				gotTitles = append(gotTitles, title)
				if want, ok := tc.wantRows[title]; ok {
					var got []string
					for _, row := range table.GetContentsMapRaw() {
						got = append(got, allocationRow(row))
					}
					if !slices.Equal(got, want) {
						t.Errorf("%s rows:\ngot  %q\nwant %q", title, got, want)
					}
				}
			}
			if !slices.Equal(gotTitles, tc.wantTitles) {
				t.Errorf("titles:\ngot  %q\nwant %q", gotTitles, tc.wantTitles)
			}

			totals := map[string]any{}
			for _, row := range tables[len(tables)-1].GetContentsMapRaw() {
				totals[row["Metric"].(string)] = row["Count"]
			}
			for metric, want := range tc.wantTotals {
				if totals[metric] != want {
					t.Errorf("summary %q = %v, want %v", metric, totals[metric], want)
				}
			}
		})
	}
}

// TestVPCOverviewIPAllocationSummaryOutput_T1538 checks that the summary
// reaches the terminal and the parsed JSON file through the report writer.
func TestVPCOverviewIPAllocationSummaryOutput_T1538(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "overview.json")
	stdout, err := writeVPCOverview(t, outFile, overviewRun{outputFormat: "table", fileFormat: "json", vpcs: allocationVPCs()})
	if err != nil {
		t.Fatalf("writeVPCOverviewTables returned error: %v", err)
	}

	for _, want := range []string{"IP Allocation Summary for vpc-111 in account " + testAccount, "EKS", "Lambda Function", "AWS Reserved"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}

	var sections []vpcOverviewSection
	if err := json.Unmarshal([]byte(readFile(t, outFile)), &sections); err != nil {
		t.Fatalf("file is not a JSON list of sections: %v", err)
	}
	for _, section := range sections {
		if section.Title != "IP Allocation Summary for vpc-111 in account "+testAccount {
			continue
		}
		var got []string
		for _, row := range section.Rows {
			// JSON numbers decode as float64; normalise for comparison.
			row[ipAddressesColumn] = int(row[ipAddressesColumn].(float64))
			got = append(got, allocationRow(row))
		}
		if !slices.Equal(got, vpc111AllocationRows) {
			t.Errorf("file rows:\ngot  %q\nwant %q", got, vpc111AllocationRows)
		}
		return
	}
	t.Errorf("file has no IP allocation summary for vpc-111; titles %q", sectionTitles(sections))
}
