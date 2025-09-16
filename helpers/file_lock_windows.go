//go:build windows

package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// validateFilePermissions checks if the file has proper permissions for reading and writing (Windows)
func validateFilePermissions(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return NewFileSystemError("failed to get file info", err).
			WithContext("file_path", filePath)
	}

	// Check if file is readable
	if fileInfo.Mode().Perm()&0400 == 0 {
		return NewFileSystemError("config file is not readable", nil).
			WithContext("file_path", filePath).
			WithContext("permissions", fileInfo.Mode().String())
	}

	return nil
}

// validateFilePermissionsForWrite checks if the file has proper permissions for modification (Windows)
func validateFilePermissionsForWrite(filePath string) error {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, check if directory is writable
		dir := filepath.Dir(filePath)
		dirInfo, err := os.Stat(dir)
		if err != nil {
			return NewFileSystemError("failed to get directory info", err).
				WithContext("directory", dir)
		}

		// Check if directory is writable
		if dirInfo.Mode().Perm()&0200 == 0 {
			return NewFileSystemError("directory is not writable", nil).
				WithContext("directory", dir).
				WithContext("permissions", dirInfo.Mode().String())
		}

		return nil
	}

	if err != nil {
		return NewFileSystemError("failed to get file info", err).
			WithContext("file_path", filePath)
	}

	// Check if file is writable
	if fileInfo.Mode().Perm()&0200 == 0 {
		return NewFileSystemError("config file is not writable", nil).
			WithContext("file_path", filePath).
			WithContext("permissions", fileInfo.Mode().String())
	}

	// Note: On Windows, we skip the Unix-specific ownership check which is not applicable
	// The file ownership model is different on Windows

	return nil
}

// acquireFileLock acquires an exclusive lock on the config file for concurrent access protection (Windows)
func acquireFileLock(file *os.File) error {
	// On Windows, we use a different approach for file locking
	// We'll try to create a lock file as an alternative
	lockPath := file.Name() + ".lock"

	// Try to create lock file exclusively
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return NewFileSystemError("config file is locked by another process", err).
				WithContext("file_path", file.Name()).
				WithContext("lock_file", lockPath)
		}
		return NewFileSystemError("failed to acquire file lock", err).
			WithContext("file_path", file.Name()).
			WithContext("lock_file", lockPath)
	}

	// Write process ID to lock file
	_, err = fmt.Fprintf(lockFile, "%d", os.Getpid())
	if err != nil {
		lockFile.Close()
		os.Remove(lockPath)
		return NewFileSystemError("failed to write to lock file", err).
			WithContext("file_path", file.Name()).
			WithContext("lock_file", lockPath)
	}

	lockFile.Close()
	return nil
}

// releaseFileLock releases the exclusive lock on the config file (Windows)
func releaseFileLock(file *os.File) error {
	// Remove the lock file
	lockPath := file.Name() + ".lock"
	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return NewFileSystemError("failed to release file lock", err).
			WithContext("file_path", file.Name()).
			WithContext("lock_file", lockPath)
	}
	return nil
}

// acquireFileLock acquires an exclusive lock on the file with timeout (Windows - method)
func (cf *AWSConfigFile) acquireFileLock(file *os.File) error {
	// Try to acquire lock with timeout using lock file approach
	lockPath := file.Name() + ".lock"
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return NewFileSystemError("timeout acquiring file lock", nil).
				WithContext("file_path", cf.FilePath).
				WithContext("timeout_seconds", 5).
				WithContext("lock_file", lockPath)
		case <-ticker.C:
			// Try to create lock file exclusively
			lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err == nil {
				// Lock acquired successfully
				fmt.Fprintf(lockFile, "%d", os.Getpid())
				lockFile.Close()
				return nil
			}
			if !os.IsExist(err) {
				return NewFileSystemError("failed to acquire file lock", err).
					WithContext("file_path", cf.FilePath).
					WithContext("lock_file", lockPath)
			}
			// Lock is held by another process, continue trying
		}
	}
}

// releaseFileLock releases the file lock (Windows - method)
func (cf *AWSConfigFile) releaseFileLock(file *os.File) error {
	// Remove the lock file
	lockPath := file.Name() + ".lock"
	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return NewFileSystemError("failed to release file lock", err).
			WithContext("file_path", cf.FilePath).
			WithContext("lock_file", lockPath)
	}
	return nil
}
