module github.com/ArjenSchwarz/awstools

go 1.15

require (
	github.com/aws/aws-sdk-go-v2 v1.2.0
	github.com/aws/aws-sdk-go-v2/config v1.1.0
	github.com/aws/aws-sdk-go-v2/service/appmesh v1.1.0
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.1.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.1.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.1.1
	github.com/aws/aws-sdk-go-v2/service/organizations v1.1.1
	github.com/aws/aws-sdk-go-v2/service/rds v1.1.1
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.1.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.1.0
	github.com/emicklei/dot v0.14.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	golang.org/x/text v0.3.3 // indirect
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
