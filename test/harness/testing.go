package harness

// TestingT is an interface that abstracts the testing framework.
// It is satisfied by *testing.T and any compatible test framework.
//
// This interface defines only the methods actually used by the harness,
// allowing the harness to work with any compatible test framework.
type TestingT interface {
	// Name returns the name of the running test or benchmark.
	Name() string

	// Helper marks the calling function as a test helper function.
	// When printing file and line information, that function will be skipped.
	Helper()

	// Cleanup registers a function to be called when the test completes.
	// Cleanup functions are called in last-in-first-out order.
	Cleanup(f func())

	// Log formats its arguments using default formatting and records the text in the log.
	Log(args ...any)

	// Logf formats its arguments according to the format, analogous to Printf,
	// and records the text in the log.
	Logf(format string, args ...any)

	// Fatal is equivalent to Log followed by FailNow.
	Fatal(args ...any)

	// Fatalf is equivalent to Logf followed by FailNow.
	Fatalf(format string, args ...any)

	// Error is equivalent to Log followed by Fail.
	Error(args ...any)

	// Errorf is equivalent to Logf followed by Fail.
	Errorf(format string, args ...any)

	// TempDir returns a temporary directory for the test to use.
	// The directory is automatically removed when the test completes.
	TempDir() string
}
