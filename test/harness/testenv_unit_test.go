package harness

import (
	"regexp"
	"testing"
)

func TestProviderInfraRef(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		provider       string
		wantAPIVersion string
		wantKind       string
	}{
		"aws": {
			provider:       "aws",
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			wantKind:       "AWSCluster",
		},
		"azure": {
			provider:       "azure",
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			wantKind:       "AzureCluster",
		},
		"gcp": {
			provider:       "gcp",
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			wantKind:       "GCPCluster",
		},
		"vsphere": {
			provider:       "vsphere",
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			wantKind:       "VSphereCluster",
		},
		"vcd": {
			provider:       "vcd",
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			wantKind:       "VCDCluster",
		},
		"unknown provider returns empty strings": {
			provider:       "unknown",
			wantAPIVersion: "",
			wantKind:       "",
		},
		"empty string returns empty strings": {
			provider:       "",
			wantAPIVersion: "",
			wantKind:       "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotAPIVersion, gotKind := providerInfraRef(tc.provider)

			if gotAPIVersion != tc.wantAPIVersion {
				t.Errorf("providerInfraRef(%q) apiVersion = %q, want %q", tc.provider, gotAPIVersion, tc.wantAPIVersion)
			}
			if gotKind != tc.wantKind {
				t.Errorf("providerInfraRef(%q) kind = %q, want %q", tc.provider, gotKind, tc.wantKind)
			}
		})
	}
}

// uuidRegex matches a standard UUID v4/v5 format (8-4-4-4-12 hex characters).
var uuidRegex = regexp.MustCompile(`^` + uuidPattern.String() + `$`)

func TestDeterministicUID(t *testing.T) {
	t.Parallel()

	t.Run("same name produces same UID", func(t *testing.T) {
		t.Parallel()
		uid1 := deterministicUID("my-resource")
		uid2 := deterministicUID("my-resource")

		if uid1 != uid2 {
			t.Errorf("deterministicUID(\"my-resource\") returned different values: %q vs %q", uid1, uid2)
		}
	})

	t.Run("different names produce different UIDs", func(t *testing.T) {
		t.Parallel()
		uid1 := deterministicUID("resource-a")
		uid2 := deterministicUID("resource-b")

		if uid1 == uid2 {
			t.Errorf("deterministicUID produced same UID for different names: %q", uid1)
		}
	})

	t.Run("result is valid UUID format", func(t *testing.T) {
		t.Parallel()
		names := []string{"test", "my-cluster", "machine-deployment-1", ""}
		for _, name := range names {
			uid := deterministicUID(name)
			if !uuidRegex.MatchString(string(uid)) {
				t.Errorf("deterministicUID(%q) = %q, does not match UUID format", name, uid)
			}
		}
	})

	t.Run("result is types.UID", func(t *testing.T) {
		t.Parallel()
		uid := deterministicUID("test")
		if uid == "" {
			t.Error("deterministicUID returned empty UID")
		}
	})
}

func TestPtr(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		got := ptr(42)
		if got == nil {
			t.Fatal("ptr(42) returned nil")
		}
		if *got != 42 {
			t.Errorf("*ptr(42) = %d, want 42", *got)
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		got := ptr("hello")
		if got == nil {
			t.Fatal("ptr(\"hello\") returned nil")
		}
		if *got != "hello" {
			t.Errorf("*ptr(\"hello\") = %q, want \"hello\"", *got)
		}
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		got := ptr(true)
		if got == nil {
			t.Fatal("ptr(true) returned nil")
		}
		if *got != true {
			t.Errorf("*ptr(true) = %v, want true", *got)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()
		got := ptr(0)
		if got == nil {
			t.Fatal("ptr(0) returned nil")
		}
		if *got != 0 {
			t.Errorf("*ptr(0) = %d, want 0", *got)
		}
	})

	t.Run("returns unique pointer", func(t *testing.T) {
		t.Parallel()
		p1 := ptr(42)
		p2 := ptr(42)
		if p1 == p2 {
			t.Error("ptr(42) called twice should return different pointers")
		}
	})
}
