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
}
