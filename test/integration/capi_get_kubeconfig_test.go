package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiGetKubeconfig(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-existent cluster kubeconfig", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("retrieves kubeconfig successfully", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		kubeconfigData := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://my-cluster-api.example.com:6443
  name: my-cluster
contexts:
- context:
    cluster: my-cluster
    user: my-cluster-admin
  name: my-cluster
current-context: my-cluster
users:
- name: my-cluster-admin
  user:
    client-certificate-data: dGVzdC1jZXJ0
    client-key-data: dGVzdC1rZXk=`

		harness.New(t).
			CreateNamespace(namespace).
			CreateSecret(namespace, "my-cluster-kubeconfig", map[string][]byte{
				"value": []byte(kubeconfigData),
			}).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			WithArg("name", "my-cluster").
			AssertContent("success.golden").
			Execute()
	})

	t.Run("retrieves kubeconfig from data key", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		kubeconfigData := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://data-cluster-api.example.com:6443
  name: data-cluster`

		harness.New(t).
			CreateNamespace(namespace).
			CreateSecret(namespace, "data-cluster-kubeconfig", map[string][]byte{
				"data": []byte(kubeconfigData),
			}).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			WithArg("name", "data-cluster").
			AssertContent("data_key.golden").
			Execute()
	})

	t.Run("returns error without namespace argument", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_kubeconfig").
			WithArg("name", "some-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("returns error without name argument", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("returns error when secret has no recognized key", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateSecret(namespace, "no-key-cluster-kubeconfig", map[string][]byte{
				"cert": []byte("some-cert-data"),
			}).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			WithArg("name", "no-key-cluster").
			AssertError("no_recognized_key.golden").
			Execute()
	})

	t.Run("returns empty kubeconfig when value key is empty", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateSecret(namespace, "empty-cluster-kubeconfig", map[string][]byte{
				"value": []byte(""),
			}).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			WithArg("name", "empty-cluster").
			AssertContent("empty_value.golden").
			Execute()
	})

	t.Run("prefers value key over data key", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateSecret(namespace, "both-keys-cluster-kubeconfig", map[string][]byte{
				"value": []byte("value-kubeconfig-content"),
				"data":  []byte("data-kubeconfig-content"),
			}).
			ToolCall("capi_get_kubeconfig").
			WithArg("namespace", namespace).
			WithArg("name", "both-keys-cluster").
			AssertContent("value_takes_precedence.golden").
			Execute()
	})
}
