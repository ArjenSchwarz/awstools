package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/viper"
)

// sampleVPCOverview returns two VPCs with subnets and IP details so every
// table type of the overview report is produced.
func sampleVPCOverview() []helpers.VPCUsageInfo {
	return []helpers.VPCUsageInfo{
		{
			ID:   "vpc-111",
			CIDR: "10.0.0.0/16",
			Subnets: []helpers.SubnetUsageInfo{
				{
					ID: "subnet-aaa", CIDR: "10.0.1.0/24", VPCId: "vpc-111",
					TotalIPs: 256, AvailableIPs: 250, UsedIPs: 6,
					IPDetails: []helpers.IPAddressInfo{
						{IPAddress: "10.0.1.10", UsageType: "EC2 Instance", AttachmentInfo: "i-0123"},
					},
				},
			},
		},
		{
			ID:   "vpc-222",
			CIDR: "10.1.0.0/16",
			Subnets: []helpers.SubnetUsageInfo{
				{
					ID: "subnet-bbb", CIDR: "10.1.1.0/24", VPCId: "vpc-222", IsPublic: true,
					TotalIPs: 256, AvailableIPs: 249, UsedIPs: 7,
					IPDetails: []helpers.IPAddressInfo{
						{IPAddress: "10.1.1.20", UsageType: "NAT Gateway", AttachmentInfo: "nat-0456", PublicIP: "203.0.113.5"},
					},
				},
			},
		},
	}
}

// writeVPCOverviewToFile renders the VPC overview through the production
// builder and writer and returns what went to stdout and to the file.
func writeVPCOverviewToFile(t *testing.T, outputFormat, fileFormat, vpcFilter string) (string, string) {
	t.Helper()
	outFile := filepath.Join(t.TempDir(), "overview")
	configureOutput(t, outputFormat, outFile)
	viper.Set("output.file-format", fileFormat)
	t.Cleanup(func() { viper.Set("output.file-format", "") })

	tables := buildVPCOverviewTables(sampleVPCOverview(), nil, "123456789012", vpcFilter)
	stdout, err := captureStdout(func() {
		if err := writeOutputTables(tables); err != nil {
			t.Errorf("writeOutputTables returned error: %v", err)
		}
	})
	if err != nil {
		t.Fatalf("capturing stdout: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	return string(stdout), string(data)
}

// TestVPCOverviewFileContainsAllTables_T1536 is the regression test for
// T-1536: with --file only the summary table was saved, because go-output
// v1.4.0 truncates the file on every Write() call and the overview writes one
// call per table. The file must contain every table shown on stdout.
func TestVPCOverviewFileContainsAllTables_T1536(t *testing.T) {
	tests := map[string]struct {
		outputFormat string
		fileFormat   string
		vpcFilter    string
		want         []string
		notWant      []string
	}{
		"markdown file": {
			outputFormat: "markdown",
			want: []string{
				"Subnet Overview for vpc-111", "Subnet Overview for vpc-222",
				"IP Details for subnet subnet-aaa", "IP Details for subnet subnet-bbb",
				"10.0.1.10", "203.0.113.5",
				"VPC Usage Summary for account 123456789012",
			},
		},
		"table on stdout and json file": {
			outputFormat: "table",
			fileFormat:   "json",
			want:         []string{`"IP Address":"10.0.1.10"`, `"IP Address":"10.1.1.20"`, `"CIDR":"10.1.1.0/24"`, `"Metric":"Total VPCs"`},
		},
		"vpc filter": {
			outputFormat: "markdown",
			vpcFilter:    "vpc-222",
			want:         []string{"Subnet Overview for vpc-222", "IP Details for subnet subnet-bbb", "VPC Usage Summary for vpc-222"},
			notWant:      []string{"vpc-111", "10.0.1.10"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, got := writeVPCOverviewToFile(t, tc.outputFormat, tc.fileFormat, tc.vpcFilter)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("output file missing %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("output file unexpectedly contains %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}

// TestVPCOverviewFileMatchesStdout_T1536 checks that, with the same format on
// screen and in the file, the file holds exactly the report printed to stdout.
func TestVPCOverviewFileMatchesStdout_T1536(t *testing.T) {
	stdout, file := writeVPCOverviewToFile(t, "markdown", "", "")
	if file != stdout {
		t.Fatalf("file content differs from stdout\nstdout:\n%s\nfile:\n%s", stdout, file)
	}
	if !strings.Contains(stdout, "IP Details for subnet subnet-aaa") {
		t.Fatalf("stdout missing IP details table, got:\n%s", stdout)
	}
}
