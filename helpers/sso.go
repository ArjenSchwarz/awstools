package helpers

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssoadmin"
)

var ssoSession = ssoadmin.New(session.New())

// SSOSession returns a shared Ec2Session
func SSOSession() *ssoadmin.SSOAdmin {
	return ssoSession
}

// GetSSOAccountInstance retrieves the SSO Account Instance and all its data
func GetSSOAccountInstance(svc *ssoadmin.SSOAdmin) SSOInstance {
	ssoInstance := getSSOInstance(svc)
	ssoInstance.getPermissionSets(svc)
	return ssoInstance
}

// SSOInstance is the top level representation of an SSO Instance
type SSOInstance struct {
	IdentityStoreID string
	Arn             string
	//PermissionSets contains the permission sets the instance has
	PermissionSets []SSOPermissionSet
	//Accounts contains the accounts with permission sets, those permission sets, and who has access
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

//SSOPolicy represents a Managed Policy
type SSOPolicy struct {
	Arn  string
	Name string
	// Policy string
}

//SSOAccount represents an AWS account managed by AWS
type SSOAccount struct {
	AccountID          string
	AccountAssignments []SSOAccountAssignment
}

//SSOAccountAssignment represents which principals are tied to an account using which permission set
type SSOAccountAssignment struct {
	PrincipalType string
	PrincipalID   string
	PermissionSet *SSOPermissionSet
}

func getSSOInstance(svc *ssoadmin.SSOAdmin) SSOInstance {
	instances, err := svc.ListInstances(&ssoadmin.ListInstancesInput{})
	if err != nil {
		panic(err)
	}
	if len(instances.Instances) < 1 {
		panic("Didn't find any SSO environments")
	}
	if len(instances.Instances) > 1 {
		panic("Found multiple SSO environments. How did you manage that?")
	}
	ssoInstance := SSOInstance{
		IdentityStoreID: *instances.Instances[0].IdentityStoreId,
		Arn:             *instances.Instances[0].InstanceArn,
	}
	return ssoInstance
}

func (instance *SSOInstance) getPermissionSets(svc *ssoadmin.SSOAdmin) []SSOPermissionSet {
	maxresults := int64(100)
	if instance.PermissionSets == nil {
		permissions, err := svc.ListPermissionSets(&ssoadmin.ListPermissionSetsInput{InstanceArn: &instance.Arn, MaxResults: &maxresults})
		if err != nil {
			panic(err)
		}
		permissionsets := []SSOPermissionSet{}
		for _, permissionsetarn := range permissions.PermissionSets {
			permissionset := instance.getPermissionSetDetails(*permissionsetarn, svc)
			permissionsets = append(permissionsets, permissionset)
		}
		instance.PermissionSets = permissionsets
	}
	return instance.PermissionSets
}

func (instance *SSOInstance) getPermissionSetDetails(permissionsetarn string, svc *ssoadmin.SSOAdmin) SSOPermissionSet {
	// Get metadata
	permissionsetdescription, err := svc.DescribePermissionSet(&ssoadmin.DescribePermissionSetInput{InstanceArn: &instance.Arn, PermissionSetArn: &permissionsetarn})
	if err != nil {
		panic(err)
	}
	permissionset := SSOPermissionSet{
		Arn:             permissionsetarn,
		Name:            *permissionsetdescription.PermissionSet.Name,
		CreatedAt:       *permissionsetdescription.PermissionSet.CreatedDate,
		SessionDuration: *permissionsetdescription.PermissionSet.SessionDuration,
		Instance:        instance,
	}
	if permissionsetdescription.PermissionSet.Description != nil {
		permissionset.Description = *permissionsetdescription.PermissionSet.Description
	}
	// Get accounts
	permissionset.addAccountInfo(svc)
	// Get managed policies
	managedpolicies, err := svc.ListManagedPoliciesInPermissionSet(&ssoadmin.ListManagedPoliciesInPermissionSetInput{InstanceArn: &instance.Arn, PermissionSetArn: &permissionsetarn})
	if err != nil {
		panic(err)
	}
	policies := []SSOPolicy{}
	for _, managedpolicy := range managedpolicies.AttachedManagedPolicies {
		policy := SSOPolicy{
			Arn:  *managedpolicy.Arn,
			Name: *managedpolicy.Name,
		}
		policies = append(policies, policy)
	}
	permissionset.ManagedPolicies = policies
	// Get Inline Policy
	inlinepolicy, err := svc.GetInlinePolicyForPermissionSet(&ssoadmin.GetInlinePolicyForPermissionSetInput{InstanceArn: &instance.Arn, PermissionSetArn: &permissionsetarn})
	if err != nil {
		panic(err)
	}
	permissionset.InlinePolicy = *inlinepolicy.InlinePolicy
	return permissionset
}

func (permissionset *SSOPermissionSet) addAccountInfo(svc *ssoadmin.SSOAdmin) []SSOAccount {
	provisionedaccounts, err := svc.ListAccountsForProvisionedPermissionSet(&ssoadmin.ListAccountsForProvisionedPermissionSetInput{
		InstanceArn:      &permissionset.Instance.Arn,
		PermissionSetArn: &permissionset.Arn,
	})
	if err != nil {
		panic(err)
	}
	accounts := []SSOAccount{}
	for _, accountnr := range provisionedaccounts.AccountIds {
		account := SSOAccount{
			AccountID: *accountnr,
		}
		accountassignments, err := svc.ListAccountAssignments(&ssoadmin.ListAccountAssignmentsInput{
			InstanceArn:      &permissionset.Instance.Arn,
			PermissionSetArn: &permissionset.Arn,
			AccountId:        accountnr,
		})
		if err != nil {
			panic(err)
		}
		for _, assignmentraw := range accountassignments.AccountAssignments {
			assignment := SSOAccountAssignment{
				PrincipalType: *assignmentraw.PrincipalType,
				PrincipalID:   *assignmentraw.PrincipalId,
				PermissionSet: permissionset,
			}
			account.addAssignmentToAccount(assignment)
		}
		permissionset.addAccount(account)
	}
	return accounts
}

func (permissionset *SSOPermissionSet) addAccount(account SSOAccount) {
	accounts := []SSOAccount{}
	if len(permissionset.Accounts) != 0 {
		accounts = permissionset.Accounts
	}
	permissionset.Accounts = append(accounts, account)
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

//GetAccountList returns a list of the account numbers in the SSO Instance
func (instance *SSOInstance) GetAccountList() []string {
	accounts := []string{}
	for _, account := range instance.Accounts {
		accounts = append(accounts, account.AccountID)
	}
	return accounts
}

func (account *SSOAccount) addAssignmentToAccount(assignment SSOAccountAssignment) {
	assignments := []SSOAccountAssignment{}
	if len(account.AccountAssignments) != 0 {
		assignments = account.AccountAssignments
	}
	account.AccountAssignments = append(assignments, assignment)
}

//GetManagedPolicyNames returns a slice containing the names of the policies attached to the permission set
func (permissionset *SSOPermissionSet) GetManagedPolicyNames() []string {
	policynames := []string{}
	for _, policy := range permissionset.ManagedPolicies {
		policynames = append(policynames, policy.Name)
	}
	return policynames
}

//GetAssignmentIdsByAccount returns the assigment's principal IDs
func (permissionset *SSOPermissionSet) GetAssignmentIdsByAccount(accountnr string) []string {
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
