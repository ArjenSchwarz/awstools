package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// captureStderr runs f while redirecting os.Stderr to a pipe and returns the
// bytes written. It restores os.Stderr before returning.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()

	f()

	_ = w.Close()
	os.Stderr = origStderr
	return <-done
}

// resetViperState returns a function that restores viper's singleton and the
// package-level cfgFile to their pre-test values. This is needed because
// initConfig mutates both.
func resetViperState(t *testing.T) func() {
	t.Helper()
	origCfgFile := cfgFile
	origViper := viper.GetViper()
	viper.Reset()
	return func() {
		cfgFile = origCfgFile
		// Restore viper by overwriting its singleton via Reset + replay is
		// overkill; tests that need a clean viper should call viper.Reset().
		_ = origViper
		viper.Reset()
	}
}

// TestInitConfig_PrintsOnSuccess_T694 verifies that when viper successfully
// reads a config file, the "Using config file:" message is printed to stderr.
//
// Expected: message appears on success (err == nil).
// Bug (T-694): message was printed on failure instead of success.
func TestInitConfig_PrintsOnSuccess_T694(t *testing.T) {
	restore := resetViperState(t)
	defer restore()

	// Create a real config file the Viper loader can parse successfully.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "awstools-test.yaml")
	if err := os.WriteFile(cfgPath, []byte("output:\n  format: json\n"), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	cfgFile = cfgPath

	out := captureStderr(t, initConfig)

	if !strings.Contains(out, "Using config file:") {
		t.Errorf("expected 'Using config file:' message on successful read, got stderr=%q", out)
	}
	if !strings.Contains(out, cfgPath) {
		t.Errorf("expected config path %q in stderr output, got %q", cfgPath, out)
	}
}

// TestInitConfig_SilentOnReadFailure_T694 verifies that when viper fails to
// read the config file (e.g. because it does not exist), the success message
// is NOT printed. A missing config file is acceptable — Viper configs are
// optional — so there should be no misleading "Using config file:" output.
//
// Expected: no "Using config file:" message on failure.
// Bug (T-694): message was incorrectly printed on failure.
func TestInitConfig_SilentOnReadFailure_T694(t *testing.T) {
	restore := resetViperState(t)
	defer restore()

	// Point cfgFile at a path that definitely does not exist so ReadInConfig
	// returns an error.
	dir := t.TempDir()
	cfgFile = filepath.Join(dir, "does-not-exist.yaml")

	out := captureStderr(t, initConfig)

	if strings.Contains(out, "Using config file:") {
		t.Errorf("did not expect 'Using config file:' on failed read, got stderr=%q", out)
	}
}
