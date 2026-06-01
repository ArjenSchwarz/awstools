package helpers

import (
	"testing"
)

// TestGetUserDetails_PopulatesID is a regression test for T-1132.
// GetUserDetails must copy the AWS UserId into IAMUser.ID so that
// IAMUser.GetID() (used for the Draw.io identity column) returns a
// non-empty, unique value. Before the fix, ID was never set and GetID()
// returned an empty string, causing duplicate/blank Draw.io node identities.
func TestGetUserDetails_PopulatesID(t *testing.T) {
	cachedUsers = nil
	defer func() { cachedUsers = nil }()

	users := makeUsers(3)
	mock := &mockIAMClient{users: users}

	result := GetUserDetails(mock)
	if len(result) != len(users) {
		t.Fatalf("GetUserDetails() returned %d users, want %d", len(result), len(users))
	}

	seen := make(map[string]bool)
	for _, user := range result {
		id := user.GetID()
		if id == "" {
			t.Errorf("user %q has empty GetID(); expected the AWS UserId", user.GetName())
			continue
		}
		if seen[id] {
			t.Errorf("duplicate GetID() %q across users", id)
		}
		seen[id] = true
	}
}

// TestGetUserDetails_IDMatchesUserId verifies the ID equals the AWS UserId.
func TestGetUserDetails_IDMatchesUserId(t *testing.T) {
	cachedUsers = nil
	defer func() { cachedUsers = nil }()

	users := makeUsers(2)
	mock := &mockIAMClient{users: users}

	wantIDs := make(map[string]bool)
	for _, u := range users {
		wantIDs[*u.UserId] = true
	}

	for _, user := range GetUserDetails(mock) {
		if !wantIDs[user.GetID()] {
			t.Errorf("user %q GetID() = %q, not a known AWS UserId", user.GetName(), user.GetID())
		}
	}
}

// TestGetGroupDetails_PopulatesID is a regression test for T-1132.
// GetGroupDetails must copy the AWS GroupId into IAMGroup.ID so that
// IAMGroup.GetID() returns a non-empty, unique Draw.io identity value.
func TestGetGroupDetails_PopulatesID(t *testing.T) {
	groups := makeGroups(3)
	mock := &mockIAMClient{groups: groups}

	result := GetGroupDetails(mock)
	if len(result) != len(groups) {
		t.Fatalf("GetGroupDetails() returned %d groups, want %d", len(result), len(groups))
	}

	seen := make(map[string]bool)
	for _, group := range result {
		id := group.GetID()
		if id == "" {
			t.Errorf("group %q has empty GetID(); expected the AWS GroupId", group.GetName())
			continue
		}
		if seen[id] {
			t.Errorf("duplicate GetID() %q across groups", id)
		}
		seen[id] = true
	}
}

// TestGetGroupDetails_IDMatchesGroupId verifies the ID equals the AWS GroupId.
func TestGetGroupDetails_IDMatchesGroupId(t *testing.T) {
	groups := makeGroups(2)
	mock := &mockIAMClient{groups: groups}

	wantIDs := make(map[string]bool)
	for _, g := range groups {
		wantIDs[*g.GroupId] = true
	}

	for _, group := range GetGroupDetails(mock) {
		if !wantIDs[group.GetID()] {
			t.Errorf("group %q GetID() = %q, not a known AWS GroupId", group.GetName(), group.GetID())
		}
	}
}
