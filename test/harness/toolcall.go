package harness

// ToolCall provides a fluent API for MCP tool call testing.
// It accumulates tool name and arguments, then queues operations
// when finalized via AssertContent or bridge methods.
type ToolCall struct {
	harness  *Harness       // parent harness for tool execution
	toolName string         // name of the MCP tool to call
	args     map[string]any // input arguments for the tool
}

// ToolCall starts a new tool call builder.
func (h *Harness) ToolCall(toolName string) *ToolCall {
	return &ToolCall{
		harness:  h,
		toolName: toolName,
		args:     make(map[string]any),
	}
}

// WithArg adds a single argument (chainable)
func (tc *ToolCall) WithArg(key string, value any) *ToolCall {
	tc.args[key] = value
	return tc
}

// WithArgs merges the provided arguments with existing arguments (chainable).
// Arguments set via WithArg are preserved; conflicting keys are overwritten.
func (tc *ToolCall) WithArgs(args map[string]any) *ToolCall {
	for k, v := range args {
		tc.args[k] = v
	}
	return tc
}

// AssertContent queues the tool call and assertion, then returns to the harness.
// This enables continued chaining after the assertion.
// The goldenPath is relative to testdata/<toolName>/.
func (tc *ToolCall) AssertContent(goldenPath string) *Harness {
	// Queue the tool call operation
	tc.harness.operations = append(tc.harness.operations, &toolCallOp{
		toolName: tc.toolName,
		args:     tc.args,
	})
	// Queue the assertion operation
	tc.harness.operations = append(tc.harness.operations, &assertContentOp{
		toolName:   tc.toolName,
		goldenPath: goldenPath,
	})
	return tc.harness
}

// AssertContentNormalized queues the tool call and normalized assertion, then returns to the harness.
// Normalizers are applied to both the actual output and golden file content before comparison,
// allowing non-deterministic fields (e.g. UID, timestamps) to be replaced with placeholders.
// The goldenPath is relative to testdata/<toolName>/.
func (tc *ToolCall) AssertContentNormalized(goldenPath string, normalizers ...Normalizer) *Harness {
	tc.harness.operations = append(tc.harness.operations, &toolCallOp{
		toolName: tc.toolName,
		args:     tc.args,
	})
	tc.harness.operations = append(tc.harness.operations, &assertContentNormalizedOp{
		toolName:    tc.toolName,
		goldenPath:  goldenPath,
		normalizers: normalizers,
	})
	return tc.harness
}

// AssertError queues the tool call and error assertion, then returns to the harness.
// Use this for tool calls that are expected to fail (protocol errors or tool errors).
// The goldenPath is relative to testdata/<toolName>/.
func (tc *ToolCall) AssertError(goldenPath string) *Harness {
	// Queue the tool call operation
	tc.harness.operations = append(tc.harness.operations, &toolCallOp{
		toolName: tc.toolName,
		args:     tc.args,
	})
	// Queue the error assertion operation
	tc.harness.operations = append(tc.harness.operations, &assertErrorOp{
		toolName:   tc.toolName,
		goldenPath: goldenPath,
	})
	return tc.harness
}
