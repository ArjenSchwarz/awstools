package cmd

import (
	"fmt"
	"strconv"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
)

// listrulesCmd represents the listrules command
var listrulesCmd = &cobra.Command{
	Use:   "listrules",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: listRules,
}

func init() {
	sgCmd.AddCommand(listrulesCmd)
}

func listRules(cmd *cobra.Command, args []string) {
	svc := helpers.Ec2Session()
	params := &ec2.DescribeSecurityGroupsInput{}
	filters := make([]*ec2.Filter, 1)
	if *vpc != "" {
		values := make([]*string, 1)
		values = append(values, vpc)
		filter := ec2.Filter{
			Name:   aws.String("vpc-id"),
			Values: values,
		}
		filters = append(filters, &filter)
	}
	params.SetFilters(filters)
	// if *groupname != "" {
	// 	groupnames := make([]*string, 1)
	// 	groupnames = append(groupnames, groupname)
	// 	params.GroupNames = groupnames
	// }
	resp, err := svc.DescribeSecurityGroups(params)
	if err != nil {
		panic(err)
	}
	rules := make([]sgRule, len(resp.SecurityGroups))
	for _, group := range resp.SecurityGroups {
		for _, permission := range group.IpPermissions {
			for _, ip := range permission.IpRanges {
				if aws.StringValue(ip.CidrIp) == "" {
					fmt.Println(aws.StringValue(group.GroupName))
				}
				rule := sgRule{
					SecurityGroupName: aws.StringValue(group.GroupName),
					FromPort:          aws.Int64Value(permission.FromPort),
					ToPort:            aws.Int64Value(permission.ToPort),
					Source:            aws.StringValue(ip.CidrIp),
				}
				rules = append(rules, rule)
			}
		}
	}
	keys := []string{"SecurityGroup", "Ports", "Source"}
	output := helpers.OutputArray{Keys: keys}
	for _, resource := range rules {
		if resource.SecurityGroupName != "" {

			content := make(map[string]string)
			content["SecurityGroup"] = resource.SecurityGroupName
			if resource.FromPort == resource.ToPort {
				if resource.FromPort == 0 {
					content["Ports"] = "ALL"
				} else {
					content["Ports"] = strconv.FormatInt(resource.FromPort, 10)
				}
			} else {
				content["Ports"] = strconv.FormatInt(resource.FromPort, 10) + " - " + strconv.FormatInt(resource.ToPort, 10)
			}
			content["Source"] = resource.Source
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write(*settings)
}

type sgRule struct {
	SecurityGroupName string
	FromPort          int64
	ToPort            int64
	Source            string
}
