package helpers

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// GetAccountID returns the ID of the account the command is run from
func GetAccountID(svc *sts.Client) string {
	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(context.TODO(), input)
	if err != nil {
		log.Fatal(err.Error())
	}
	return accountIDFromIdentity(result)
}

// accountIDFromIdentity safely extracts the Account field from an STS
// GetCallerIdentity response. The AWS SDK returns Account as *string
// and in some edge cases (e.g. SSO sessions in specific states) it can
// be nil; aws.ToString converts nil pointers to empty strings rather
// than panicking.
func accountIDFromIdentity(result *sts.GetCallerIdentityOutput) string {
	if result == nil {
		return ""
	}
	return aws.ToString(result.Account)
}
