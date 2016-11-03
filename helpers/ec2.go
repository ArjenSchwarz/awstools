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
