package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

var cachedIAMPolicyDocuments = make(map[string]*IAMPolicyDocument)

// GetRolesAndPolicies returns all the roles and and their attached policies
func GetRolesAndPolicies(verbose bool, svc IAMClient) ([]IAMRole, map[string]IAMPolicyDocument) {
	roles := GetRoleDetails(verbose, svc)
	policies := make(map[string]IAMPolicyDocument)
	for _, role := range roles {
		for policyname, policy := range role.InlinePolicies {
			cleanedname := fmt.Sprintf("%s (inline for %s)", policyname, role.Name)
			policies[cleanedname] = *policy
		}
		for policyname, policy := range role.AttachedPolicies {
			policies[policyname] = *policy
		}
	}
	return roles, policies
}

// GetRoleDetails returns the list of roles in the account
func GetRoleDetails(verbose bool, svc IAMClient) []IAMRole {
	result := []IAMRole{}

	paginator := iam.NewListRolesPaginator(svc, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatal(err.Error())
		}
		for _, role := range page.Roles {
			policydocument := IAMPolicyDocument{Type: IAMPolicyTypeAssumeRole}
			decodeddocument, err := url.QueryUnescape(*role.AssumeRolePolicyDocument)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal([]byte(decodeddocument), &policydocument)
			if err != nil {
				panic(err)
			}
			inlinepolicies := getInlinePoliciesForRole(*role.RoleName, verbose, svc)
			attachedpolicies := getAttachedPoliciesForRole(*role.RoleName, verbose, svc)
			rolestruct := IAMRole{
				Name:             *role.RoleName,
				ID:               *role.RoleId,
				AssumeRolePolicy: policydocument,
				Path:             *role.Path,
				Role:             &role,
				InlinePolicies:   inlinepolicies,
				AttachedPolicies: attachedpolicies,
				Type:             getRoleType(role),
				Verbose:          verbose,
			}
			for _, policy := range rolestruct.InlinePolicies {
				policy.AddRole(&rolestruct)
			}
			for _, policy := range rolestruct.AttachedPolicies {
				policy.AddRole(&rolestruct)
			}
			result = append(result, rolestruct)
		}
	}
	return result
}

func getRoleType(role types.Role) string {
	rolePath := *role.Path
	if rolePath == "/service-role/" || (len(rolePath) > 18 && rolePath[0:18] == "/aws-service-role/") {
		return IAMRoleTypeServiceRole
	}
	if len(rolePath) > 32 && rolePath[0:32] == "/aws-reserved/sso.amazonaws.com/" {
		return IAMRoleTypeSSOManaged
	}
	return IAMRoleTypeUserDefined
}

func getInlinePoliciesForRole(rolename string, verbose bool, svc IAMClient) map[string]*IAMPolicyDocument {
	policies := make(map[string]*IAMPolicyDocument)

	paginator := iam.NewListRolePoliciesPaginator(svc, &iam.ListRolePoliciesInput{RoleName: &rolename})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatal(err.Error())
		}
		for _, policy := range page.PolicyNames {
			cacheKey := rolename + "/" + policy
			if _, ok := cachedIAMPolicyDocuments[cacheKey]; !ok {
				policydocument := IAMPolicyDocument{
					Name: policy,
					Type: IAMPolicyTypeInline,
				}
				if verbose {
					detailinput := &iam.GetRolePolicyInput{
						RoleName:   &rolename,
						PolicyName: &policy,
					}
					detailresp, err := svc.GetRolePolicy(context.TODO(), detailinput)
					if err != nil {
						log.Fatal(err.Error())
					}

					decodeddocument, err := url.QueryUnescape(*detailresp.PolicyDocument)
					if err != nil {
						panic(err)
					}
					err = json.Unmarshal([]byte(decodeddocument), &policydocument)
					if err != nil {
						panic(err)
					}
				}
				cachedIAMPolicyDocuments[cacheKey] = &policydocument
			}
			policies[policy] = cachedIAMPolicyDocuments[cacheKey]
		}
	}
	return policies
}

// getAttachedPoliciesForRole(rolename string, svc *iam.IAM) map[string]
func getAttachedPoliciesForRole(rolename string, verbose bool, svc IAMClient) map[string]*IAMPolicyDocument {
	policies := make(map[string]*IAMPolicyDocument)

	paginator := iam.NewListAttachedRolePoliciesPaginator(svc, &iam.ListAttachedRolePoliciesInput{RoleName: &rolename})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatal(err.Error())
		}
		for _, policy := range page.AttachedPolicies {
			policyname := *policy.PolicyName
			if _, ok := cachedIAMPolicyDocuments[policyname]; !ok {
				policydocument := IAMPolicyDocument{
					Type: IAMPolicyTypeAttached,
					Name: policyname,
				}
				if verbose {
					policystring := getAttachedPolicy(policy.PolicyArn, svc)
					decodeddocument, err := url.QueryUnescape(policystring)
					if err != nil {
						panic(err)
					}
					err = json.Unmarshal([]byte(decodeddocument), &policydocument)
					if err != nil {
						panic(err)
					}
				}
				cachedIAMPolicyDocuments[policyname] = &policydocument
			}
			policies[policyname] = cachedIAMPolicyDocuments[policyname]
		}
	}
	return policies
}
