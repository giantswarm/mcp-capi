package capi

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testKubeconfig = "apiVersion: v1\nkind: Config\n"

// kubeconfigClient returns a Client over a fake clientset that holds the
// kubeconfig Secret of testCluster, with the given policy.
func kubeconfigClient(policy WritePolicy) *Client {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: testCluster + "-kubeconfig"},
		Data:       map[string][]byte{"value": []byte(testKubeconfig)},
	}
	c := &Client{}
	WithWritePolicy(policy)(c)
	c.SetClients(fake.NewSimpleClientset(secret), nil)
	return c
}

// TestGetKubeconfigRefusedByDefault is the library-level guarantee behind the
// hidden tool: without ExposeKubeconfig the kubeconfig Secret is never read,
// whatever the write switches say.
func TestGetKubeconfigRefusedByDefault(t *testing.T) {
	for name, policy := range map[string]WritePolicy{
		caseZeroPolicy:    {},
		caseReadOnly:      {ReadOnly: true, GitOpsGuard: true},
		"writes allowed":  {GitOpsGuard: true},
		"nothing guarded": {ReadOnly: false, GitOpsGuard: false},
	} {
		t.Run(name, func(t *testing.T) {
			c := kubeconfigClient(policy)
			got, err := c.GetKubeconfig(context.Background(), testNamespace, testCluster)
			if !errors.Is(err, ErrCredentialExport) {
				t.Fatalf("GetKubeconfig() error = %v, want ErrCredentialExport", err)
			}
			if got != "" {
				t.Fatalf("GetKubeconfig() returned %q on refusal, want empty", got)
			}
			if actions := c.k8sClient.(*fake.Clientset).Actions(); len(actions) != 0 {
				t.Fatalf("GetKubeconfig() touched the API server %d time(s) before refusing: %v", len(actions), actions)
			}
		})
	}
}

func TestGetKubeconfigWithExposeKubeconfig(t *testing.T) {
	c := kubeconfigClient(WritePolicy{ReadOnly: true, GitOpsGuard: true, ExposeKubeconfig: true})
	got, err := c.GetKubeconfig(context.Background(), testNamespace, testCluster)
	if err != nil {
		t.Fatalf("GetKubeconfig() error = %v", err)
	}
	if got != testKubeconfig {
		t.Fatalf("GetKubeconfig() = %q, want %q", got, testKubeconfig)
	}
}

// TestBackupClusterIncludeSecretsRefusedByDefault: a backup never carries
// Secret data unless the export is allowed. The refusal comes before the
// cluster is fetched, so no client is needed.
func TestBackupClusterIncludeSecretsRefusedByDefault(t *testing.T) {
	for name, policy := range map[string]WritePolicy{
		caseZeroPolicy:   {},
		caseReadOnly:     {ReadOnly: true, GitOpsGuard: true},
		"writes allowed": {GitOpsGuard: true},
	} {
		t.Run(name, func(t *testing.T) {
			c := &Client{}
			WithWritePolicy(policy)(c)
			got, err := c.BackupCluster(context.Background(), BackupClusterOptions{Namespace: testNamespace, Name: testCluster, IncludeSecrets: true})
			if !errors.Is(err, ErrCredentialExport) {
				t.Fatalf("BackupCluster(include secrets) error = %v, want ErrCredentialExport", err)
			}
			if got != "" {
				t.Fatalf("BackupCluster() returned %q on refusal, want empty", got)
			}
		})
	}
}
