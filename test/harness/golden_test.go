package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeUID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"single UUID": {
			input: "resource UID: 550e8400-e29b-41d4-a716-446655440000 found",
			want:  "resource UID: <UID> found",
		},
		"multiple UUIDs": {
			input: "cluster 550e8400-e29b-41d4-a716-446655440000 has machine a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			want:  "cluster <UID> has machine <UID>",
		},
		"no UUIDs": {
			input: "no uids here, just plain text",
			want:  "no uids here, just plain text",
		},
		"partial UUID not matched": {
			input: "short segment 550e8400-e29b-41d4 is not a full UUID",
			want:  "short segment 550e8400-e29b-41d4 is not a full UUID",
		},
		"empty string": {
			input: "",
			want:  "",
		},
		"UUID at start of string": {
			input: "550e8400-e29b-41d4-a716-446655440000 is the id",
			want:  "<UID> is the id",
		},
		"UUID at end of string": {
			input: "the id is 550e8400-e29b-41d4-a716-446655440000",
			want:  "the id is <UID>",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeUID(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeUID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeTimestamp(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"standard timestamp": {
			input: "created at 2024-01-15 10:30:45 +0000 UTC done",
			want:  "created at <TIMESTAMP> done",
		},
		"no timestamps": {
			input: "no timestamps here",
			want:  "no timestamps here",
		},
		"multiple timestamps": {
			input: "start: 2024-01-15 10:30:45 +0000 UTC end: 2025-12-31 23:59:59 +0000 UTC",
			want:  "start: <TIMESTAMP> end: <TIMESTAMP>",
		},
		"empty string": {
			input: "",
			want:  "",
		},
		"partial timestamp not matched": {
			input: "date 2024-01-15 without time",
			want:  "date 2024-01-15 without time",
		},
		"timestamp with non-zero offset": {
			input: "created at 2024-06-15 14:30:00 +0530 IST done",
			want:  "created at <TIMESTAMP> done",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeTimestamp(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeTimestamp(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCompareWithGolden(t *testing.T) {
	t.Parallel()

	t.Run("matching content returns nil", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "match.golden")
		content := "hello world\nline two\n"

		if err := os.WriteFile(goldenPath, []byte(content), filePermissions); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		err := compareWithGolden(content, goldenPath)
		if err != nil {
			t.Errorf("expected nil error for matching content, got: %v", err)
		}
	})

	t.Run("mismatched content returns error with diff", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "mismatch.golden")

		if err := os.WriteFile(goldenPath, []byte("expected output\n"), filePermissions); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		err := compareWithGolden("actual output\n", goldenPath)
		if err == nil {
			t.Fatal("expected error for mismatched content, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "does not match golden file") {
			t.Errorf("error should mention golden file mismatch, got: %v", err)
		}
		if !strings.Contains(errMsg, "-") && !strings.Contains(errMsg, "+") {
			t.Errorf("error should contain diff markers, got: %v", err)
		}
	})

	t.Run("nonexistent golden file returns helpful error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "nonexistent.golden")

		err := compareWithGolden("some text", goldenPath)
		if err == nil {
			t.Fatal("expected error for nonexistent golden file, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "does not exist") {
			t.Errorf("error should mention file does not exist, got: %v", err)
		}
		if !strings.Contains(errMsg, "UPDATE_GOLDEN=true") {
			t.Errorf("error should suggest UPDATE_GOLDEN=true, got: %v", err)
		}
	})
}

func TestCompareWithGoldenNormalized(t *testing.T) {
	t.Parallel()

	t.Run("with normalizer applied to both sides", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "normalized.golden")

		// Golden file has a different UUID than the actual text,
		// but after normalization both should match.
		goldenContent := "resource <UID> is ready\n"
		if err := os.WriteFile(goldenPath, []byte(goldenContent), filePermissions); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		actual := "resource 550e8400-e29b-41d4-a716-446655440000 is ready\n"
		normalizers := []Normalizer{NormalizeUID}

		err := compareWithGoldenNormalized(actual, goldenPath, normalizers)
		if err != nil {
			t.Errorf("expected nil error with normalizer, got: %v", err)
		}
	})

	t.Run("nil normalizers behaves like compareWithGolden", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "plain.golden")
		content := "plain text content\n"

		if err := os.WriteFile(goldenPath, []byte(content), filePermissions); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		err := compareWithGoldenNormalized(content, goldenPath, nil)
		if err != nil {
			t.Errorf("expected nil error with nil normalizers, got: %v", err)
		}
	})

	t.Run("mismatch after normalization returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "mismatch.golden")

		if err := os.WriteFile(goldenPath, []byte("expected\n"), filePermissions); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		err := compareWithGoldenNormalized("different\n", goldenPath, nil)
		if err == nil {
			t.Fatal("expected error for mismatched content, got nil")
		}
	})

	t.Run("multiple normalizers applied in order", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "multi.golden")

		goldenContent := "resource <UID> created at <TIMESTAMP>\n"
		if err := os.WriteFile(goldenPath, []byte(goldenContent), filePermissions); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		actual := "resource 550e8400-e29b-41d4-a716-446655440000 created at 2024-01-15 10:30:45 +0000 UTC\n"
		normalizers := []Normalizer{NormalizeUID, NormalizeTimestamp}

		err := compareWithGoldenNormalized(actual, goldenPath, normalizers)
		if err != nil {
			t.Errorf("expected nil error with multiple normalizers, got: %v", err)
		}
	})
}

func TestUpdateGoldenFile(t *testing.T) {
	t.Parallel()

	t.Run("creates file with correct content", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "output.golden")
		content := "test content\nline two\n"

		err := updateGoldenFile(content, goldenPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(got) != content {
			t.Errorf("file content = %q, want %q", string(got), content)
		}
	})

	t.Run("creates parent directories if needed", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "nested", "deeply", "output.golden")
		content := "nested content\n"

		err := updateGoldenFile(content, goldenPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(got) != content {
			t.Errorf("file content = %q, want %q", string(got), content)
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		goldenPath := filepath.Join(dir, "overwrite.golden")

		if err := os.WriteFile(goldenPath, []byte("old content"), filePermissions); err != nil {
			t.Fatalf("failed to write initial file: %v", err)
		}

		newContent := "new content\n"
		err := updateGoldenFile(newContent, goldenPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(got) != newContent {
			t.Errorf("file content = %q, want %q", string(got), newContent)
		}
	})
}
