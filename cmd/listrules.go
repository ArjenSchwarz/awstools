package cmd

import (
	"fmt"
	"strconv"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	awsConfig := config.DefaultAwsConfig()
	var securitygroups []ec2.SecurityGroup
	if *vpc == "" {
		securitygroups = helpers.GetAllSecurityGroups(awsConfig)
	} else {
		securitygroups = helpers.GetAllSecurityGroupsForVPC(*vpc, awsConfig)
	}

	rules := make([]sgRule, len(securitygroups))
	for _, group := range securitygroups {
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
