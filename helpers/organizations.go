package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// OrganizationsAPI defines the subset of the Organizations client used by this package.
type OrganizationsAPI interface {
	ListRoots(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options)) (*organizations.ListRootsOutput, error)
	ListChildren(ctx context.Context, params *organizations.ListChildrenInput, optFns ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error)
	DescribeOrganizationalUnit(ctx context.Context, params *organizations.DescribeOrganizationalUnitInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error)
	DescribeAccount(ctx context.Context, params *organizations.DescribeAccountInput, optFns ...func(*organizations.Options)) (*organizations.DescribeAccountOutput, error)
}

func getOrganizationRoot(svc OrganizationsAPI) (OrganizationEntry, error) {
	root, err := svc.ListRoots(context.TODO(), &organizations.ListRootsInput{})
	if err != nil {
		return OrganizationEntry{}, fmt.Errorf("failed to list organization roots: %w", err)
	}
	if len(root.Roots) == 0 {
		return OrganizationEntry{}, fmt.Errorf("no organization roots found")
	}
	rootentry := root.Roots[0]
	entry := OrganizationEntry{
		ID:   aws.ToString(rootentry.Id),
		Arn:  aws.ToString(rootentry.Arn),
		Name: aws.ToString(rootentry.Name),
		Type: string(types.TargetTypeRoot),
	}
	return entry, nil
}

// GetFullOrganization returns the root entry of the organization with all children fleshed out
func GetFullOrganization(svc OrganizationsAPI) (OrganizationEntry, error) {
	root, err := getOrganizationRoot(svc)
	if err != nil {
		return OrganizationEntry{}, err
	}
	children, err := root.findChildren(svc)
	if err != nil {
		return OrganizationEntry{}, err
	}
	root.Children = children
	return root, nil
}

// OrganizationEntry is a helper struct for Organization resources
type OrganizationEntry struct {
	ID       string
	Name     string
	Arn      string
	Type     string
	Children []OrganizationEntry
}

func (entry *OrganizationEntry) findChildren(svc OrganizationsAPI) ([]OrganizationEntry, error) {
	children := []OrganizationEntry{}
	ouinput := &organizations.ListChildrenInput{
		ParentId:  aws.String(entry.ID),
		ChildType: types.ChildType(types.TargetTypeOrganizationalUnit),
	}
	for {
		ouchildren, err := svc.ListChildren(context.TODO(), ouinput)
		if err != nil {
			return nil, fmt.Errorf("failed to list OU children of %s: %w", entry.ID, err)
		}
		for _, child := range ouchildren.Children {
			ouchild, err := formatChild(child, svc)
			if err != nil {
				return nil, err
			}
			ouchildChildren, err := ouchild.findChildren(svc)
			if err != nil {
				return nil, err
			}
			ouchild.Children = ouchildChildren
			children = append(children, ouchild)
		}
		if ouchildren.NextToken == nil {
			break
		}
		ouinput.NextToken = ouchildren.NextToken
	}
	accountinput := &organizations.ListChildrenInput{
		ParentId:  aws.String(entry.ID),
		ChildType: types.ChildType(types.TargetTypeAccount),
	}
	for {
		accountchildren, err := svc.ListChildren(context.TODO(), accountinput)
		if err != nil {
			return nil, fmt.Errorf("failed to list account children of %s: %w", entry.ID, err)
		}
		for _, child := range accountchildren.Children {
			accountchild, err := formatChild(child, svc)
			if err != nil {
				return nil, err
			}
			children = append(children, accountchild)
		}
		if accountchildren.NextToken == nil {
			break
		}
		accountinput.NextToken = accountchildren.NextToken
	}
	return children, nil
}

func (entry *OrganizationEntry) String() string {
	return entry.Name + " (" + entry.ID + ")"
}

func formatChild(raw types.Child, svc OrganizationsAPI) (OrganizationEntry, error) {
	if raw.Type == types.ChildType(types.TargetTypeOrganizationalUnit) {
		input := &organizations.DescribeOrganizationalUnitInput{
			OrganizationalUnitId: raw.Id,
		}
		details, err := svc.DescribeOrganizationalUnit(context.TODO(), input)
		if err != nil {
			return OrganizationEntry{}, fmt.Errorf("failed to describe OU %s: %w", *raw.Id, err)
		}
		return OrganizationEntry{
			Name:     *details.OrganizationalUnit.Name,
			ID:       *details.OrganizationalUnit.Id,
			Type:     string(raw.Type),
			Arn:      *details.OrganizationalUnit.Arn,
			Children: []OrganizationEntry{},
		}, nil
	}
	input := &organizations.DescribeAccountInput{
		AccountId: raw.Id,
	}
	details, err := svc.DescribeAccount(context.TODO(), input)
	if err != nil {
		return OrganizationEntry{}, fmt.Errorf("failed to describe account %s: %w", *raw.Id, err)
	}
	return OrganizationEntry{
		Name:     *details.Account.Name,
		ID:       *details.Account.Id,
		Type:     string(raw.Type),
		Arn:      *details.Account.Arn,
		Children: []OrganizationEntry{},
	}, nil
}
