module github.com/ArjenSchwarz/awstools

go 1.15

require (
	github.com/aws/aws-sdk-go v1.37.6
	github.com/aws/aws-sdk-go-v2 v1.2.0
	github.com/aws/aws-sdk-go-v2/config v1.1.0
	github.com/aws/aws-sdk-go-v2/service/appmesh v1.1.0
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.1.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.1.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.1.1
	github.com/emicklei/dot v0.14.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
