package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// organizationsListRootsAPI is the interface for the ListRoots AWS Organizations API call.
type organizationsListRootsAPI interface {
	ListRoots(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options)) (*organizations.ListRootsOutput, error)
}

func getOrganizationRoot(svc organizationsListRootsAPI) (OrganizationEntry, error) {
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
func GetFullOrganization(svc *organizations.Client) (OrganizationEntry, error) {
	root, err := getOrganizationRoot(svc)
	if err != nil {
		return OrganizationEntry{}, err
	}
	root.Children = root.findChildren(svc)
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

func (entry *OrganizationEntry) findChildren(svc *organizations.Client) []OrganizationEntry {
	children := []OrganizationEntry{}
	ouinput := &organizations.ListChildrenInput{
		ParentId:  aws.String(entry.ID),
		ChildType: types.ChildType(types.TargetTypeOrganizationalUnit),
	}
	ouchildren, err := svc.ListChildren(context.TODO(), ouinput)
	if err != nil {
		fmt.Println(err)
	}
	for _, child := range ouchildren.Children {
		ouchild := formatChild(child, svc)
		ouchild.Children = ouchild.findChildren(svc)
		children = append(children, ouchild)
	}
	accountinput := &organizations.ListChildrenInput{
		ParentId:  aws.String(entry.ID),
		ChildType: types.ChildType(types.TargetTypeAccount),
	}
	accountchildren, _ := svc.ListChildren(context.TODO(), accountinput)

	for _, child := range accountchildren.Children {
		accountchild := formatChild(child, svc)
		children = append(children, accountchild)
	}
	return children
}

func (entry *OrganizationEntry) String() string {
	return entry.Name + " (" + entry.ID + ")"
}

func formatChild(raw types.Child, svc *organizations.Client) OrganizationEntry {
	if raw.Type == types.ChildType(types.TargetTypeOrganizationalUnit) {
		input := &organizations.DescribeOrganizationalUnitInput{
			OrganizationalUnitId: raw.Id,
		}
		details, err := svc.DescribeOrganizationalUnit(context.TODO(), input)
		if err != nil {
			fmt.Println(err)
		}
		return OrganizationEntry{
			Name:     *details.OrganizationalUnit.Name,
			ID:       *details.OrganizationalUnit.Id,
			Type:     string(raw.Type),
			Arn:      *details.OrganizationalUnit.Arn,
			Children: []OrganizationEntry{},
		}
	}
	input := &organizations.DescribeAccountInput{
		AccountId: raw.Id,
	}
	details, err := svc.DescribeAccount(context.TODO(), input)
	if err != nil {
		fmt.Println(err)
	}
	return OrganizationEntry{
		Name:     *details.Account.Name,
		ID:       *details.Account.Id,
		Type:     string(raw.Type),
		Arn:      *details.Account.Arn,
		Children: []OrganizationEntry{},
	}
}
