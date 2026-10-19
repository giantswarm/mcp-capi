package capi

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const rolloutMD = "workers"

// TestRolloutMachineDeploymentSetsRolloutAfter: a rollout sets
// spec.rolloutAfter, which makes CAPI replace the machines; template
// annotations alone are propagated in place and replace none.
func TestRolloutMachineDeploymentSetsRolloutAfter(t *testing.T) {
	scheme, err := newScheme()
	if err != nil {
		t.Fatal(err)
	}
	md := &clusterv1.MachineDeployment{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: rolloutMD}}
	ctrl := ctrlfake.NewClientBuilder().WithScheme(scheme).WithObjects(md).Build()
	c := &Client{}
	WithWritePolicy(WritePolicy{})(c)
	c.SetClients(nil, ctrl)

	before := time.Now().Add(-time.Second)
	if err := c.RolloutMachineDeployment(context.Background(), RolloutMachineDeploymentOptions{Namespace: testNamespace, Name: rolloutMD, Reason: "new AMI"}); err != nil {
		t.Fatalf("RolloutMachineDeployment() error = %v", err)
	}

	got := &clusterv1.MachineDeployment{}
	if err := ctrl.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: rolloutMD}, got); err != nil {
		t.Fatal(err)
	}
	if got.Spec.RolloutAfter == nil || got.Spec.RolloutAfter.Before(&metav1.Time{Time: before}) {
		t.Fatalf("spec.rolloutAfter = %v, want now: without it CAPI replaces no machine", got.Spec.RolloutAfter)
	}
	if got.Annotations[RolloutReasonAnnotation] != "new AMI" {
		t.Errorf("annotation %s = %q, want the reason", RolloutReasonAnnotation, got.Annotations[RolloutReasonAnnotation])
	}
	if len(got.Spec.Template.Annotations) != 0 {
		t.Errorf("machine template annotations = %v, want none", got.Spec.Template.Annotations)
	}
}
