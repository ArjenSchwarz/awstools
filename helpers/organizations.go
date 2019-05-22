package helpers

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
)

var organizationsSession = organizations.New(session.New())

// OrganizationsSession returns a shared organizationsSession
func OrganizationsSession() *organizations.Organizations {
	return organizationsSession
}

func getOrganizationRoot(svc *organizations.Organizations) OrganizationEntry {
	root, err := svc.ListRoots(&organizations.ListRootsInput{})
	if err != nil {
		fmt.Print(err)
	}
	rootentry := root.Roots[0]
	entry := OrganizationEntry{
		ID:   *rootentry.Id,
		Arn:  *rootentry.Arn,
		Name: *rootentry.Name,
		Type: organizations.TargetTypeRoot,
	}
	return entry
}

// GetFullOrganization returns the root entry of the organization with all children fleshed out
func GetFullOrganization(svc *organizations.Organizations) OrganizationEntry {
	root := getOrganizationRoot(svc)
	root.Children = root.findChildren(svc)
	return root
}

// OrganizationEntry is a helper struct for Organization resources
type OrganizationEntry struct {
	ID       string
	Name     string
	Arn      string
	Type     string
	Children []OrganizationEntry
}

func (entry *OrganizationEntry) findChildren(svc *organizations.Organizations) []OrganizationEntry {
	children := []OrganizationEntry{}
	ouinput := &organizations.ListChildrenInput{
		ParentId:  aws.String(entry.ID),
		ChildType: aws.String(organizations.TargetTypeOrganizationalUnit),
	}
	ouchildren, err := svc.ListChildren(ouinput)
	if err != nil {
		fmt.Println(err)
	}
	for _, child := range ouchildren.Children {
		ouchild := formatChild(*child, svc)
		ouchild.Children = ouchild.findChildren(svc)
		children = append(children, ouchild)
	}
	accountinput := &organizations.ListChildrenInput{
		ParentId:  aws.String(entry.ID),
		ChildType: aws.String(organizations.TargetTypeAccount),
	}
	accountchildren, err := svc.ListChildren(accountinput)
	if err != nil {
	}
	for _, child := range accountchildren.Children {
		accountchild := formatChild(*child, svc)
		children = append(children, accountchild)
	}
	return children
}

func (entry *OrganizationEntry) String() string {
	if entry.Type == organizations.TargetTypeRoot {
		return entry.Type
	} else if entry.Type == organizations.TargetTypeOrganizationalUnit {
		return entry.Type + ": " + entry.Name
	} else {
		return entry.Name + " (" + entry.ID + ")"
	}
}

func formatChild(raw organizations.Child, svc *organizations.Organizations) OrganizationEntry {
	if *raw.Type == organizations.TargetTypeOrganizationalUnit {
		input := &organizations.DescribeOrganizationalUnitInput{
			OrganizationalUnitId: raw.Id,
		}
		details, err := svc.DescribeOrganizationalUnit(input)
		if err != nil {
			fmt.Println(err)
		}
		return OrganizationEntry{
			Name:     *details.OrganizationalUnit.Name,
			ID:       *details.OrganizationalUnit.Id,
			Type:     *raw.Type,
			Arn:      *details.OrganizationalUnit.Arn,
			Children: []OrganizationEntry{},
		}
	}
	input := &organizations.DescribeAccountInput{
		AccountId: raw.Id,
	}
	details, err := svc.DescribeAccount(input)
	if err != nil {
		fmt.Println(err)
	}
	return OrganizationEntry{
		Name:     *details.Account.Name,
		ID:       *details.Account.Id,
		Type:     *raw.Type,
		Arn:      *details.Account.Arn,
		Children: []OrganizationEntry{},
	}
}
