package helpers

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAppendProfilesCreatesMissingNestedFile is a regression test for T-1010:
// profile generation to a brand-new --output-file path (including one inside a
// directory that does not yet exist) must succeed and create the file, rather
// than failing because the write-permission pre-check stats a non-existent
// parent directory.
func TestAppendProfilesCreatesMissingNestedFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Parent directory "newdir" does not exist yet.
	path := filepath.Join(tmpDir, "newdir", "config")

	cf, err := LoadAWSConfigFile(path)
	if err != nil {
		t.Fatalf("LoadAWSConfigFile: unexpected error: %v", err)
	}

	if err := cf.AppendProfiles([]GeneratedProfile{
		{Name: "fresh", Region: "us-east-1"},
	}); err != nil {
		t.Fatalf("AppendProfiles to non-existent nested path should succeed, got: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected config file to be created at %s: %v", path, statErr)
	}

	loaded, err := LoadAWSConfigFile(path)
	if err != nil {
		t.Fatalf("reload: unexpected error: %v", err)
	}
	if _, ok := loaded.Profiles["fresh"]; !ok {
		t.Errorf("expected profile 'fresh' to be present after creating new file")
	}
}

// TestValidateFilePermissionsForWrite_MissingNestedDir verifies the underlying
// permission pre-check tolerates a target whose parent directory does not yet
// exist, validating the nearest existing ancestor instead.
func TestValidateFilePermissionsForWrite_MissingNestedDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "a", "b", "c", "config")

	if err := validateFilePermissionsForWrite(path); err != nil {
		t.Fatalf("validateFilePermissionsForWrite for missing nested dir should succeed, got: %v", err)
	}
}
