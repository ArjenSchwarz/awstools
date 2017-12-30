package helpers

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// GetEc2Name returns the name of the provided EC2 Resource
func GetEc2Name(ec2name string, config aws.Config) string {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2name},
	}
	reservations := describeInstances(params, config)

	for _, reservation := range reservations {
		for _, instance := range reservation.Instances {
			for _, tag := range instance.Tags {
				if aws.StringValue(tag.Key) == "Name" {
					return aws.StringValue(tag.Value)
				}
			}
		}
	}
	return ""
}

// GetAllSecurityGroups returns a list of all securitygroups in the region
func GetAllSecurityGroups(config aws.Config) []ec2.SecurityGroup {
	svc := ec2.New(config)
	req := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{})
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}

	return resp.SecurityGroups
}

// GetAllSecurityGroupsForVPC returns a list of all securitygroups for the provided VPC
func GetAllSecurityGroupsForVPC(vpc string, config aws.Config) []ec2.SecurityGroup {
	svc := ec2.New(config)
	params := &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpc},
			},
		},
	}

	req := svc.DescribeSecurityGroupsRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}

	return resp.SecurityGroups
}

// GetEc2BySecurityGroup retrieves all instances attached to a securitygroup
func GetEc2BySecurityGroup(securitygroupID string, config aws.Config) []ec2.RunInstancesOutput {
	params := &ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []string{securitygroupID},
			},
		},
	}
	return describeInstances(params, config)
}

// GetAllEc2Instances retrieves all EC2 instances
func GetAllEc2Instances(config aws.Config) []ec2.RunInstancesOutput {
	return describeInstances(&ec2.DescribeInstancesInput{}, config)
}

// IsLatestInstanceFamily checks if an instance is part of the la
// test family is running in the latest instance family.
// TODO: Automate this to work properly
func IsLatestInstanceFamily(instanceFamily string) bool {
	family := instanceFamily[0:1]
	version := instanceFamily[1:]
	switch family {
	case "c":
		return version == "4"
	case "d":
		return version == "2"
	case "f":
		return version == "1"
	case "g":
		return version == "3"
	case "p":
		return version == "2"
	case "i":
		return version == "3"
	case "m":
		return version == "4"
	case "r":
		return version == "4"
	case "t":
		return version == "2"
	case "x":
		return version == "1"
	default:
		return false
	}
}

func describeInstances(params *ec2.DescribeInstancesInput, config aws.Config) []ec2.RunInstancesOutput {
	svc := ec2.New(config)
	req := svc.DescribeInstancesRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}
