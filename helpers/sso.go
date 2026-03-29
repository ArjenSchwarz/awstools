package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

// SSOAdminAPI defines the subset of the SSO Admin client used by this package.
type SSOAdminAPI interface {
	ListInstances(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error)
	ListPermissionSets(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error)
	DescribePermissionSet(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error)
	ListAccountsForProvisionedPermissionSet(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error)
	ListAccountAssignments(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error)
	ListManagedPoliciesInPermissionSet(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error)
	GetInlinePolicyForPermissionSet(ctx context.Context, params *ssoadmin.GetInlinePolicyForPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.GetInlinePolicyForPermissionSetOutput, error)
}

// GetSSOAccountInstance retrieves the SSO Account Instance and all its data
func GetSSOAccountInstance(svc SSOAdminAPI) (SSOInstance, error) {
	ssoInstance, err := getSSOInstance(svc)
	if err != nil {
		return SSOInstance{}, err
	}
	if _, err := ssoInstance.getPermissionSets(svc); err != nil {
		return SSOInstance{}, err
	}
	return ssoInstance, nil
}

// SSOInstance is the top level representation of an SSO Instance
type SSOInstance struct {
	IdentityStoreID string
	Arn             string
	// PermissionSets contains the permission sets the instance has
	PermissionSets []SSOPermissionSet
	// Accounts contains the accounts with permission sets, those permission sets, and who has access
	Accounts map[string]SSOAccount
}

// SSOPermissionSet is the representation of a permission set
type SSOPermissionSet struct {
	Arn             string
	Name            string
	Description     string
	CreatedAt       time.Time
	SessionDuration string
	Accounts        []SSOAccount
	ManagedPolicies []SSOPolicy
	InlinePolicy    string
	Instance        *SSOInstance
}

// SSOPolicy represents a Managed Policy
type SSOPolicy struct {
	Arn  string
	Name string
	// Policy string
}

// SSOAccount represents an AWS account managed by AWS
type SSOAccount struct {
	AccountID          string
	AccountAssignments []SSOAccountAssignment
}

// SSOAccountAssignment represents which principals are tied to an account using which permission set
type SSOAccountAssignment struct {
	PrincipalType string
	PrincipalID   string
	PermissionSet *SSOPermissionSet
}

func getSSOInstance(svc SSOAdminAPI) (SSOInstance, error) {
	instances, err := svc.ListInstances(context.TODO(), &ssoadmin.ListInstancesInput{})
	if err != nil {
		return SSOInstance{}, fmt.Errorf("failed to list SSO instances: %w", err)
	}
	if len(instances.Instances) < 1 {
		return SSOInstance{}, fmt.Errorf("no SSO instances found")
	}
	if len(instances.Instances) > 1 {
		return SSOInstance{}, fmt.Errorf("found multiple SSO instances, expected exactly one")
	}
	ssoInstance := SSOInstance{
		IdentityStoreID: aws.ToString(instances.Instances[0].IdentityStoreId),
		Arn:             aws.ToString(instances.Instances[0].InstanceArn),
	}
	return ssoInstance, nil
}

func (instance *SSOInstance) getPermissionSets(svc SSOAdminAPI) ([]SSOPermissionSet, error) {
	maxresults := int32(100)
	if instance.PermissionSets == nil {
		var allPermissionSetArns []string
		var nextToken *string

		for {
			permissions, err := svc.ListPermissionSets(context.TODO(), &ssoadmin.ListPermissionSetsInput{
				InstanceArn: &instance.Arn,
				MaxResults:  &maxresults,
				NextToken:   nextToken,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list permission sets: %w", err)
			}
			allPermissionSetArns = append(allPermissionSetArns, permissions.PermissionSets...)
			nextToken = permissions.NextToken
			if nextToken == nil {
				break
			}
		}

		permissionsets := []SSOPermissionSet{}
		for _, permissionsetarn := range allPermissionSetArns {
			permissionset, err := instance.getPermissionSetDetails(permissionsetarn, svc)
			if err != nil {
				return nil, err
			}
			permissionsets = append(permissionsets, permissionset)
		}
		instance.PermissionSets = permissionsets
	}
	return instance.PermissionSets, nil
}

func (instance *SSOInstance) getPermissionSetDetails(permissionsetarn string, svc SSOAdminAPI) (SSOPermissionSet, error) {
	// Get metadata
	permissionsetdescription, err := svc.DescribePermissionSet(context.TODO(), &ssoadmin.DescribePermissionSetInput{
		InstanceArn:      &instance.Arn,
		PermissionSetArn: &permissionsetarn,
	})
	if err != nil {
		return SSOPermissionSet{}, fmt.Errorf("failed to describe permission set %s: %w", permissionsetarn, err)
	}
	var createdAt time.Time
	if permissionsetdescription.PermissionSet.CreatedDate != nil {
		createdAt = *permissionsetdescription.PermissionSet.CreatedDate
	}
	permissionset := SSOPermissionSet{
		Arn:             permissionsetarn,
		Name:            aws.ToString(permissionsetdescription.PermissionSet.Name),
		CreatedAt:       createdAt,
		SessionDuration: aws.ToString(permissionsetdescription.PermissionSet.SessionDuration),
		Instance:        instance,
	}
	if permissionsetdescription.PermissionSet.Description != nil {
		permissionset.Description = *permissionsetdescription.PermissionSet.Description
	}
	// Get accounts
	if err := permissionset.addAccountInfo(svc); err != nil {
		return SSOPermissionSet{}, err
	}
	// Get managed policies
	maxresults := int32(100)
	var allPolicies []SSOPolicy
	var nextToken *string

	for {
		managedpolicies, err := svc.ListManagedPoliciesInPermissionSet(context.TODO(), &ssoadmin.ListManagedPoliciesInPermissionSetInput{
			InstanceArn:      &instance.Arn,
			PermissionSetArn: &permissionsetarn,
			MaxResults:       &maxresults,
			NextToken:        nextToken,
		})
		if err != nil {
			return SSOPermissionSet{}, fmt.Errorf("failed to list managed policies for permission set %s: %w", permissionsetarn, err)
		}
		for _, managedpolicy := range managedpolicies.AttachedManagedPolicies {
			policy := SSOPolicy{
				Arn:  aws.ToString(managedpolicy.Arn),
				Name: aws.ToString(managedpolicy.Name),
			}
			allPolicies = append(allPolicies, policy)
		}
		nextToken = managedpolicies.NextToken
		if nextToken == nil {
			break
		}
	}
	permissionset.ManagedPolicies = allPolicies
	// Get Inline Policy
	inlinepolicy, err := svc.GetInlinePolicyForPermissionSet(context.TODO(), &ssoadmin.GetInlinePolicyForPermissionSetInput{
		InstanceArn:      &instance.Arn,
		PermissionSetArn: &permissionsetarn,
	})
	if err != nil {
		return SSOPermissionSet{}, fmt.Errorf("failed to get inline policy for permission set %s: %w", permissionsetarn, err)
	}
	permissionset.InlinePolicy = aws.ToString(inlinepolicy.InlinePolicy)
	return permissionset, nil
}

func (permissionset *SSOPermissionSet) addAccountInfo(svc SSOAdminAPI) error {
	maxresults := int32(100)
	var allAccountIDs []string
	var nextToken *string

	for {
		provisionedaccounts, err := svc.ListAccountsForProvisionedPermissionSet(context.TODO(), &ssoadmin.ListAccountsForProvisionedPermissionSetInput{
			InstanceArn:      &permissionset.Instance.Arn,
			PermissionSetArn: &permissionset.Arn,
			MaxResults:       &maxresults,
			NextToken:        nextToken,
		})
		if err != nil {
			return fmt.Errorf("failed to list accounts for permission set %s: %w", permissionset.Arn, err)
		}
		allAccountIDs = append(allAccountIDs, provisionedaccounts.AccountIds...)
		nextToken = provisionedaccounts.NextToken
		if nextToken == nil {
			break
		}
	}

	for _, accountnr := range allAccountIDs {
		account := SSOAccount{
			AccountID: accountnr,
		}
		var assignmentNextToken *string
		for {
			accountassignments, err := svc.ListAccountAssignments(context.TODO(), &ssoadmin.ListAccountAssignmentsInput{
				InstanceArn:      &permissionset.Instance.Arn,
				PermissionSetArn: &permissionset.Arn,
				AccountId:        aws.String(accountnr),
				MaxResults:       &maxresults,
				NextToken:        assignmentNextToken,
			})
			if err != nil {
				return fmt.Errorf("failed to list account assignments for account %s, permission set %s: %w", accountnr, permissionset.Arn, err)
			}
			for _, assignmentraw := range accountassignments.AccountAssignments {
				assignment := SSOAccountAssignment{
					PrincipalType: string(assignmentraw.PrincipalType),
					PrincipalID:   aws.ToString(assignmentraw.PrincipalId),
					PermissionSet: permissionset,
				}
				account.addAssignmentToAccount(assignment)
			}
			assignmentNextToken = accountassignments.NextToken
			if assignmentNextToken == nil {
				break
			}
		}
		permissionset.addAccount(account)
	}
	return nil
}

func (permissionset *SSOPermissionSet) addAccount(account SSOAccount) {
	accounts := []SSOAccount{}
	if len(permissionset.Accounts) != 0 {
		accounts = permissionset.Accounts
	}
	accounts = append(accounts, account)
	permissionset.Accounts = accounts
	permissionset.Instance.addAssignmentsToAccount(account)
}

func (instance *SSOInstance) addAssignmentsToAccount(account SSOAccount) {
	if instance.Accounts == nil {
		instance.Accounts = make(map[string]SSOAccount)
	}
	if existingaccount, ok := instance.Accounts[account.AccountID]; !ok {
		instance.Accounts[account.AccountID] = account
	} else {
		for _, assignment := range account.AccountAssignments {
			existingaccount.addAssignmentToAccount(assignment)
			instance.Accounts[account.AccountID] = existingaccount
		}
	}
}

// GetAccountList returns a list of the account numbers in the SSO Instance
func (instance *SSOInstance) GetAccountList() []string {
	accounts := []string{}
	for _, account := range instance.Accounts {
		accounts = append(accounts, account.AccountID)
	}
	return accounts
}

// GetPermissionSetList returns a list of the permission sets in the SSO Instance
func (instance *SSOInstance) GetPermissionSetList() []string {
	permissionsets := []string{}
	for _, permissionset := range instance.PermissionSets {
		permissionsets = append(permissionsets, permissionset.Name)
	}
	return permissionsets
}

func (account *SSOAccount) addAssignmentToAccount(assignment SSOAccountAssignment) {
	assignments := []SSOAccountAssignment{}
	if len(account.AccountAssignments) != 0 {
		assignments = account.AccountAssignments
	}
	assignments = append(assignments, assignment)
	account.AccountAssignments = assignments
}

// GetPrincipalIDsForPermissionSet returns the ids of the principals that have been assigned to the provided permission set
func (account *SSOAccount) GetPrincipalIDsForPermissionSet(permissionset SSOPermissionSet) []string {
	accountchildren := []string{}
	for _, assignment := range account.AccountAssignments {
		if assignment.PermissionSet.Name == permissionset.Name {
			accountchildren = append(accountchildren, assignment.PrincipalID)
		}
	}
	return accountchildren
}

// GetManagedPolicyNames returns a slice containing the names of the policies attached to the permission set
func (permissionset *SSOPermissionSet) GetManagedPolicyNames() []string {
	policynames := []string{}
	for _, policy := range permissionset.ManagedPolicies {
		policynames = append(policynames, policy.Name)
	}
	return policynames
}

// GetAssignmentIDsByAccount returns the assigment's principal IDs
func (permissionset *SSOPermissionSet) GetAssignmentIDsByAccount(accountnr string) []string {
	result := []string{}
	for _, account := range permissionset.Accounts {
		if account.AccountID == accountnr {
			for _, assignment := range account.AccountAssignments {
				result = append(result, assignment.PrincipalID)
			}
		}
	}
	return result
}
