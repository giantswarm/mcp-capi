package harness

import (
	"fmt"
	"strings"
	"testing"
)

// mockT implements TestingT for unit testing the harness itself.
// It captures calls to error/fatal methods instead of actually failing.
type mockT struct {
	name     string
	cleanups []func()
	errors   []string
	fatals   []string
	logs     []string
}

func (m *mockT) Name() string     { return m.name }
func (m *mockT) Helper()          {}
func (m *mockT) Cleanup(f func()) { m.cleanups = append(m.cleanups, f) }
func (m *mockT) Log(args ...any)  { m.logs = append(m.logs, fmt.Sprint(args...)) }
func (m *mockT) Logf(format string, args ...any) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}
func (m *mockT) Fatal(args ...any) { m.fatals = append(m.fatals, fmt.Sprint(args...)) }
func (m *mockT) Fatalf(format string, args ...any) {
	m.fatals = append(m.fatals, fmt.Sprintf(format, args...))
}
func (m *mockT) Error(args ...any) { m.errors = append(m.errors, fmt.Sprint(args...)) }
func (m *mockT) Errorf(format string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}
func (m *mockT) TempDir() string { return "" }

// runCleanups runs all registered cleanups in LIFO order (matching testing.T behavior).
func (m *mockT) runCleanups() {
	for i := len(m.cleanups) - 1; i >= 0; i-- {
		m.cleanups[i]()
	}
}

func TestHarness_ForgottenExecute(t *testing.T) {
	t.Parallel()

	mt := &mockT{name: "TestForgotten"}
	h := New(mt)
	h.CreateNamespace("test")

	// Simulate test completion: run registered cleanups without calling Execute.
	mt.runCleanups()

	if len(mt.errors) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(mt.errors), mt.errors)
	}
	if !strings.Contains(mt.errors[0], "queued operations") {
		t.Errorf("error should mention queued operations, got: %q", mt.errors[0])
	}
	if !strings.Contains(mt.errors[0], "Execute() was never called") {
		t.Errorf("error should mention Execute() was never called, got: %q", mt.errors[0])
	}
}

func TestHarness_DoubleExecute(t *testing.T) {
	t.Parallel()

	mt := &mockT{name: "TestDouble"}
	h := New(mt)

	// Simulate that Execute() has already been called by setting the flag directly.
	// Calling Execute() for real would trigger k8senv initialization which requires
	// the test environment manager.
	h.executed = true

	// Second Execute should trigger a fatal.
	h.Execute()

	if len(mt.fatals) != 1 {
		t.Fatalf("expected exactly 1 fatal, got %d: %v", len(mt.fatals), mt.fatals)
	}
	if !strings.Contains(mt.fatals[0], "Execute() called twice") {
		t.Errorf("fatal should mention Execute() called twice, got: %q", mt.fatals[0])
	}
}

func TestHarness_NoOperationsNoWarning(t *testing.T) {
	t.Parallel()

	mt := &mockT{name: "TestNoOps"}
	_ = New(mt)

	// No operations queued, Execute() not called. Cleanup should be silent.
	mt.runCleanups()

	if len(mt.errors) != 0 {
		t.Errorf("expected no errors when no operations queued, got: %v", mt.errors)
	}
	if len(mt.fatals) != 0 {
		t.Errorf("expected no fatals when no operations queued, got: %v", mt.fatals)
	}
}
