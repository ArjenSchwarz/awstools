package helpers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var ec2Session = ec2.New(session.New())

// Ec2Session returns a shared Ec2Session
func Ec2Session() *ec2.EC2 {
	return ec2Session
}

// GetEc2Name returns the name of the provided EC2 Resource
func GetEc2Name(ec2name *string) string {
	svc := Ec2Session()
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{ec2name},
	}
	resp, err := svc.DescribeInstances(params)

	if err != nil {
		panic(err)
	}

	for _, reservation := range resp.Reservations {
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
func GetAllSecurityGroups() []*ec2.SecurityGroup {
	svc := Ec2Session()
	resp, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		panic(err)
	}

	return resp.SecurityGroups
}

// GetEc2BySecurityGroup retrieves all instances attached to a securitygroup
func GetEc2BySecurityGroup(securitygroupID *string) []*ec2.Reservation {
	svc := Ec2Session()
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []*string{securitygroupID},
			},
		},
	}
	resp, err := svc.DescribeInstances(input)
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}

// GetAllEc2Instances retrieves all EC2 instances
func GetAllEc2Instances() []*ec2.Reservation {
	svc := Ec2Session()
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}

// VpcPeering represents a VPC Peering object
type VpcPeering struct {
	RequesterVpc     string
	RequesterAccount string
	AccepterVpc      string
	AccepterAccount  string
	PeeringName      string
}

// GetAllVpcPeers returns the peerings that are present in this region of this account
func GetAllVpcPeers(svc *ec2.EC2) []VpcPeering {
	var result []VpcPeering
	resp, err := svc.DescribeVpcPeeringConnections(&ec2.DescribeVpcPeeringConnectionsInput{})
	if err != nil {
		panic(err)
	}
	for _, connection := range resp.VpcPeeringConnections {
		peering := VpcPeering{
			RequesterVpc:     *connection.RequesterVpcInfo.VpcId,
			RequesterAccount: *connection.RequesterVpcInfo.OwnerId,
			AccepterVpc:      *connection.AccepterVpcInfo.VpcId,
			AccepterAccount:  *connection.AccepterVpcInfo.OwnerId,
			PeeringName:      *connection.VpcPeeringConnectionId,
		}
		result = append(result, peering)
	}
	return result
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
