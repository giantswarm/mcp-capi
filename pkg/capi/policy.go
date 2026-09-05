package capi

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Labels and annotations that mark an object as owned by a declarative
// controller. A direct change to such an object is drift: the controller
// reverts it on its next reconciliation, and the change belongs in the Git
// source (or the release values) the controller applies.
const (
	// Flux kustomize-controller stamps every object it applies.
	fluxKustomizationNameLabel      = "kustomize.toolkit.fluxcd.io/name"
	fluxKustomizationNamespaceLabel = "kustomize.toolkit.fluxcd.io/namespace"
	// Flux helm-controller stamps the objects of a HelmRelease.
	fluxHelmReleaseNameLabel      = "helm.toolkit.fluxcd.io/name"
	fluxHelmReleaseNamespaceLabel = "helm.toolkit.fluxcd.io/namespace"
	// Argo CD tracks objects by label (default) or annotation.
	argoInstanceLabel      = "argocd.argoproj.io/instance"
	argoTrackingAnnotation = "argocd.argoproj.io/tracking-id"
	// Helm 3 marks every object of a release. On Giant Swarm management
	// clusters the workload clusters are rendered by the cluster charts
	// (cluster-aws, ...) from an App or HelmRelease that lives in Git.
	helmReleaseNameAnnotation      = "meta.helm.sh/release-name"
	helmReleaseNamespaceAnnotation = "meta.helm.sh/release-namespace"
	managedByLabel                 = "app.kubernetes.io/managed-by"
	managedByHelm                  = "Helm"

	// PreventDeletionLabel marks an object that must not be deleted. Giant
	// Swarm's deletion blocker denies the request; mcp-capi refuses it up
	// front, whatever the write policy, so an agent gets the reason instead
	// of an admission error.
	PreventDeletionLabel = "giantswarm.io/prevent-deletion"
)

// ErrReadOnly is returned for every mutating call while the client runs
// read-only.
var ErrReadOnly = errors.New("mcp-capi is read-only")

// ErrManagedResource is returned when a mutating call targets an object owned
// by a GitOps controller or a Helm release and the GitOps guard is on.
var ErrManagedResource = errors.New("resource is managed declaratively")

// ErrDeletionPrevented is returned when the target carries
// PreventDeletionLabel.
var ErrDeletionPrevented = errors.New("resource is protected against deletion")

// ErrCredentialExport is returned when a call would hand out a workload
// cluster's credentials (its kubeconfig Secret, or Secret data in a backup)
// and ExposeKubeconfig is off.
var ErrCredentialExport = errors.New("workload cluster credentials are not exported")

// WritePolicy decides whether a mutating call may reach the API server, and
// whether a workload cluster's credentials may leave it. The zero value
// permits everything except the credential export; the serve command turns
// ReadOnly and GitOpsGuard on by default and leaves ExposeKubeconfig off.
type WritePolicy struct {
	// ReadOnly refuses every mutating call (create, update, patch, delete).
	ReadOnly bool
	// GitOpsGuard refuses mutating calls on objects that a GitOps controller
	// (Flux, Argo CD) or a Helm release owns: the change would be reverted
	// and belongs in the Git source instead. Creates are not affected, there
	// is no object to inspect.
	GitOpsGuard bool
	// ExposeKubeconfig allows a workload cluster's credentials to leave the
	// server: the admin kubeconfig Secret through GetKubeconfig, and Secret
	// data in BackupCluster. Off by default and independent of ReadOnly:
	// reading a kubeconfig is not a write, but handing it out is the power to
	// do anything to the workload cluster, so it takes an explicit opt-in.
	ExposeKubeconfig bool
}

// Owner names the declarative controller that owns an object.
type Owner struct {
	// Kind is the owner type, e.g. "Flux Kustomization" or "Helm release".
	Kind string
	// Name is the owner's namespace/name where known.
	Name string
	// Marker is the label or annotation that identified the owner.
	Marker string
	// Advice tells the caller where the change belongs.
	Advice string
}

func (o Owner) String() string {
	if o.Name == "" {
		return fmt.Sprintf("%s (%s)", o.Kind, o.Marker)
	}
	return fmt.Sprintf("%s %s (%s)", o.Kind, o.Name, o.Marker)
}

// ManagedBy reports the declarative owner of obj, if any. The first marker
// found wins, most specific first: Flux Kustomization, Flux HelmRelease,
// Argo CD, Helm.
func ManagedBy(obj metav1.Object) (Owner, bool) {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	if name := labels[fluxKustomizationNameLabel]; name != "" {
		return Owner{
			Kind:   "Flux Kustomization",
			Name:   qualified(labels[fluxKustomizationNamespaceLabel], name),
			Marker: "label " + fluxKustomizationNameLabel,
			Advice: "change the manifests in the Kustomization's Git source",
		}, true
	}
	if name := labels[fluxHelmReleaseNameLabel]; name != "" {
		return Owner{
			Kind:   "Flux HelmRelease",
			Name:   qualified(labels[fluxHelmReleaseNamespaceLabel], name),
			Marker: "label " + fluxHelmReleaseNameLabel,
			Advice: "change the HelmRelease values in Git",
		}, true
	}
	if name := labels[argoInstanceLabel]; name != "" {
		return Owner{
			Kind:   "Argo CD Application",
			Name:   name,
			Marker: "label " + argoInstanceLabel,
			Advice: "change the manifests in the Application's Git source",
		}, true
	}
	if id := annotations[argoTrackingAnnotation]; id != "" {
		return Owner{
			Kind:   "Argo CD Application",
			Name:   id,
			Marker: "annotation " + argoTrackingAnnotation,
			Advice: "change the manifests in the Application's Git source",
		}, true
	}
	if name := annotations[helmReleaseNameAnnotation]; name != "" {
		return Owner{
			Kind:   "Helm release",
			Name:   qualified(annotations[helmReleaseNamespaceAnnotation], name),
			Marker: "annotation " + helmReleaseNameAnnotation,
			Advice: "change the release values (the App or HelmRelease in Git)",
		}, true
	}
	if labels[managedByLabel] == managedByHelm {
		return Owner{
			Kind:   "Helm release",
			Marker: fmt.Sprintf("label %s=%s", managedByLabel, managedByHelm),
			Advice: "change the release values (the App or HelmRelease in Git)",
		}, true
	}
	return Owner{}, false
}

// CheckCredentialExport applies the policy to the export of the credentials
// of Cluster namespace/name; what names them ("the kubeconfig", "Secrets in
// the backup"). Only ExposeKubeconfig applies: the export is a read, so
// neither ReadOnly nor the GitOps guard has a say.
func (p WritePolicy) CheckCredentialExport(what, namespace, name string) error {
	if p.ExposeKubeconfig {
		return nil
	}
	return fmt.Errorf("refusing to export %s of Cluster %s: %w (start the server with --expose-kubeconfig to allow it)", what, qualified(namespace, name), ErrCredentialExport)
}

// CheckCreate applies the policy to the creation of a new object of kind in
// namespace. Only ReadOnly applies: there is no existing object to inspect.
func (p WritePolicy) CheckCreate(kind, namespace, name string) error {
	if p.ReadOnly {
		return fmt.Errorf("refusing to create %s %s: %w (start the server with --read-only=false to allow mutating tools)", kind, qualified(namespace, name), ErrReadOnly)
	}
	return nil
}

// CheckUpdate applies the policy to an update or patch of obj.
func (p WritePolicy) CheckUpdate(kind string, obj metav1.Object) error {
	return p.check("update", kind, obj)
}

// CheckDelete applies the policy to the deletion of obj. An object carrying
// PreventDeletionLabel is refused whatever the policy says.
func (p WritePolicy) CheckDelete(kind string, obj metav1.Object) error {
	if err := p.check("delete", kind, obj); err != nil {
		return err
	}
	if _, protected := obj.GetLabels()[PreventDeletionLabel]; protected {
		return fmt.Errorf("refusing to delete %s %s: %w (label %s is set; remove it first if the deletion is intended)", kind, qualified(obj.GetNamespace(), obj.GetName()), ErrDeletionPrevented, PreventDeletionLabel)
	}
	return nil
}

func (p WritePolicy) check(verb, kind string, obj metav1.Object) error {
	target := fmt.Sprintf("%s %s", kind, qualified(obj.GetNamespace(), obj.GetName()))
	if p.ReadOnly {
		return fmt.Errorf("refusing to %s %s: %w (start the server with --read-only=false to allow mutating tools)", verb, target, ErrReadOnly)
	}
	if !p.GitOpsGuard {
		return nil
	}
	owner, managed := ManagedBy(obj)
	if !managed {
		return nil
	}
	return fmt.Errorf("refusing to %s %s: %w: it is owned by %s and a direct change would be reverted on the next reconciliation; %s", verb, target, ErrManagedResource, owner, owner.Advice)
}

func qualified(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}
