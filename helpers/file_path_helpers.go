package helpers

import (
	"os"
	"path/filepath"
)

// nearestExistingDir walks up the directory tree starting at dir and returns
// the first ancestor that exists on disk. If dir itself exists it is returned
// unchanged. This is used by write-permission validation so that writing to a
// brand-new config path inside not-yet-created directories is allowed: the
// write path creates the missing directories with os.MkdirAll, so it is enough
// to confirm that the nearest existing ancestor is writable.
//
// The loop terminates because filepath.Dir eventually returns a fixed point
// (the filesystem root or "." for relative paths), which always exists.
func nearestExistingDir(dir string) string {
	for {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root / fixed point; return it even if Stat failed
			// so the caller surfaces a meaningful error for that path.
			return dir
		}
		dir = parent
	}
}
