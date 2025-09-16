//go:build unix

package helpers

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// validateFilePermissions checks if the file has proper permissions for reading and writing (Unix)
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

// validateFilePermissionsForWrite checks if the file has proper permissions for modification (Unix)
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

	// Check if file is owned by current user (security check)
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		currentUID := os.Getuid()
		if int(stat.Uid) != currentUID {
			return NewFileSystemError("config file is not owned by current user", nil).
				WithContext("file_path", filePath).
				WithContext("file_uid", stat.Uid).
				WithContext("current_uid", currentUID)
		}
	}

	return nil
}

// acquireFileLock acquires an exclusive lock on the config file for concurrent access protection (Unix)
func acquireFileLock(file *os.File) error {
	// Use flock for file locking on Unix systems
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return NewFileSystemError("config file is locked by another process", err).
				WithContext("file_path", file.Name())
		}
		return NewFileSystemError("failed to acquire file lock", err).
			WithContext("file_path", file.Name())
	}
	return nil
}

// releaseFileLock releases the exclusive lock on the config file (Unix)
func releaseFileLock(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return NewFileSystemError("failed to release file lock", err).
			WithContext("file_path", file.Name())
	}
	return nil
}

// acquireFileLock acquires an exclusive lock on the file with timeout (Unix - method)
func (cf *AWSConfigFile) acquireFileLock(file *os.File) error {
	// Try to acquire lock with timeout
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return NewFileSystemError("timeout acquiring file lock", nil).
				WithContext("file_path", cf.FilePath).
				WithContext("timeout_seconds", 5)
		case <-ticker.C:
			// Try to acquire exclusive lock
			err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				return nil // Lock acquired successfully
			}
			if err != syscall.EWOULDBLOCK {
				return NewFileSystemError("failed to acquire file lock", err).
					WithContext("file_path", cf.FilePath)
			}
			// Lock is held by another process, continue trying
		}
	}
}

// releaseFileLock releases the file lock (Unix - method)
func (cf *AWSConfigFile) releaseFileLock(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return NewFileSystemError("failed to release file lock", err).
			WithContext("file_path", cf.FilePath)
	}
	return nil
}
