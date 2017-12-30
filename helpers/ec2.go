package helpers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
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

func GetAllSecurityGroups() []*ec2.SecurityGroup {
	svc := Ec2Session()
	resp, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		panic(err)
	}

	return resp.SecurityGroups
}

func GetEc2BySecurityGroup(securitygroupId *string) []*ec2.Reservation {
	svc := Ec2Session()
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []*string{securitygroupId},
			},
		},
	}
	resp, err := svc.DescribeInstances(input)
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}

func GetAllEc2Instances() []*ec2.Reservation {
	svc := Ec2Session()
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}

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

func GetMagneticVolumes() []*ec2.Volume {
	svc := Ec2Session()
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("volume-type"),
				Values: []*string{aws.String("st1"), aws.String("sc1"), aws.String("standard")},
			},
		},
	}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		panic(err)
	}
	return result.Volumes
}

func GetUnattachedVolumes() []*ec2.Volume {
	svc := Ec2Session()
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("status"),
				Values: []*string{aws.String("available")},
			},
		},
	}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		panic(err)
	}
	return result.Volumes
}

func GetAllASGs() []*autoscaling.Group {
	svc := autoscaling.New(session.New())
	result, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		panic(err)
	}
	return result.AutoScalingGroups
}

func GetAllELBs() []*elb.LoadBalancerDescription {
	svc := elb.New(session.New())
	result, err := svc.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		panic(err)
	}
	return result.LoadBalancerDescriptions
}

func GetNrOfUnhealthyELBInstances(elbname *string) int {
	svc := elb.New(session.New())
	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: elbname,
	}
	unhealthy := 0
	result, err := svc.DescribeInstanceHealth(input)
	if err != nil {
		panic(err)
	}
	for _, status := range result.InstanceStates {
		if aws.StringValue(status.State) == "OutOfService" {
			unhealthy++
		}
	}
	return unhealthy
}

func GetELBsByName(elbnames []*string) ([]*elb.LoadBalancerDescription, error) {
	svc := elb.New(session.New())
	result, err := svc.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
		LoadBalancerNames: elbnames,
	})
	if err != nil {
		return nil, err
	}
	return result.LoadBalancerDescriptions, nil
}

func DoesELBExist(elbname string) bool {
	var search []*string
	search = append(search, &elbname)
	_, err := GetELBsByName(search)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				return false
			}
		}
		panic(err)
	}
	return true
}
