package helpers

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// GetAccountID returns the ID of the account the command is run from
func GetAccountID(svc *sts.Client) string {
	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(context.TODO(), input)
	if err != nil {
		log.Fatal(err.Error())
	}
	return *result.Account
}
