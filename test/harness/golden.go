package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/go-cmp/cmp"
)

// Normalizer is a function that normalizes text before golden file comparison.
// It is applied to both the actual output and the golden file content (or the
// output before writing when updating golden files).
type Normalizer func(string) string

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

// compareWithGoldenNormalized compares text with a golden file after applying normalizers.
// Normalizers are applied to both the actual text and the golden file content.
// When UPDATE_GOLDEN is set, normalizers are applied to the text before writing.
func compareWithGoldenNormalized(text, goldenPath string, normalizers []Normalizer) error {
	normalized := text
	for _, n := range normalizers {
		normalized = n(normalized)
	}

	if os.Getenv("UPDATE_GOLDEN") == "true" {
		return updateGoldenFile(normalized, goldenPath)
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("golden file %s does not exist (run tests with UPDATE_GOLDEN=true to create it)", goldenPath)
		}
		return fmt.Errorf("failed to read golden file %s: %w", goldenPath, err)
	}

	expectedNormalized := string(expected)
	for _, n := range normalizers {
		expectedNormalized = n(expectedNormalized)
	}

	if normalized != expectedNormalized {
		diff := cmp.Diff(expectedNormalized, normalized)
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

var (
	uuidPattern      = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \+\d{4} [A-Z]+`)
)

// NormalizeUID replaces all UUID strings with a deterministic placeholder.
func NormalizeUID(text string) string {
	return uuidPattern.ReplaceAllString(text, "<UID>")
}

// NormalizeTimestamp replaces all Go-formatted timestamps with a deterministic placeholder.
func NormalizeTimestamp(text string) string {
	return timestampPattern.ReplaceAllString(text, "<TIMESTAMP>")
}
