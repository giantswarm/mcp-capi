package harness

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
)

const (
	// dirPermissions is the permission mode for directories (rwxr-xr-x).
	dirPermissions = 0o755
	// filePermissions is the permission mode for golden files (rw-r--r--).
	filePermissions = 0o644
)

// compareWithGolden compares the given text with the content of a golden file.
// If the UPDATE_GOLDEN environment variable is set to "true", it updates the golden file instead.
// Returns an error (rather than failing the test directly) to allow callers to provide
// additional context in failure messages.
// See package documentation for detailed usage information.
func compareWithGolden(text, goldenPath string) error {
	// Check if we should update the golden file
	if os.Getenv("UPDATE_GOLDEN") == "true" {
		return updateGoldenFile(text, goldenPath)
	}

	// Read the golden file
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("golden file %s does not exist (run tests with UPDATE_GOLDEN=true to create it)", goldenPath)
		}
		return fmt.Errorf("failed to read golden file %s: %w", goldenPath, err)
	}

	// Compare text with golden file content
	if text != string(expected) {
		diff := cmp.Diff(string(expected), text)
		return fmt.Errorf("output does not match golden file %s:\n(-expected +actual)\n%s", goldenPath, diff)
	}

	return nil
}

// updateGoldenFile writes the given text to a golden file.
// It creates the directory if it doesn't exist.
func updateGoldenFile(text, goldenPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(goldenPath)
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the golden file
	if err := os.WriteFile(goldenPath, []byte(text), filePermissions); err != nil {
		return fmt.Errorf("failed to write golden file %s: %w", goldenPath, err)
	}

	return nil
}
