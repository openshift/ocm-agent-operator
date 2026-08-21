package ocmagenthandler

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"

	netv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildNetworkPolicyName(ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) types.NamespacedName {
	var namespacedName types.NamespacedName

	switch namespace {
	case oah.NamespaceMonitorng:
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentDefaultNetworkPolicySuffix)
	case oah.NamespaceRHOBS:
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentRHOBSNetworkPolicySuffix)
	case oah.NamespaceMUO:
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentMUONetworkPolicySuffix)
	case oah.NamespaceOBO:
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentOBONetworkPolicySuffix)
	}

	return namespacedName
}

// callerPodSelector returns the pod label selector for the known caller pod in namespace.
// A NetworkPolicyPeer with only a NamespaceSelector matches every pod in that namespace, so
// there is no safe "not yet verified" fallback here: namespace must be one of the constants
// wired into ensureAllNetworkPolicies/ensureAllNetworkPoliciesDeleted. Reaching default means a
// namespace was added to one of those lists without a case added here, so it errors out instead
// of silently degrading to a namespace-wide policy.
func callerPodSelector(namespace string) (*metav1.LabelSelector, error) {
	switch namespace {
	case oah.NamespaceMonitorng:
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{oah.AlertmanagerPodLabelKey: oah.AlertmanagerPodLabelValue},
		}, nil
	case oah.NamespaceMUO:
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{oah.MUOPodLabelKey: oah.MUOPodLabelValue},
		}, nil
	case oah.NamespaceRHOBS:
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{oah.RHOBSPodLabelKey: oah.RHOBSPodLabelValue},
		}, nil
	case oah.NamespaceOBO:
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{oah.OBOPodLabelKey: oah.OBOPodLabelValue},
		}, nil
	default:
		return nil, fmt.Errorf("callerPodSelector: no pod selector defined for namespace %q", namespace)
	}
}

// callerNamespace returns the real k8s namespace that hosts the pods callerPodSelector(namespace)
// matches. This is namespace itself for every caller except RHOBS: RHOBS's fleet-mode
// Alertmanager runs in NamespaceOBO alongside the OBO Alertmanager, not in a namespace of its
// own (oah.NamespaceRHOBS is a dispatch key, not a literal namespace - see its doc comment).
func callerNamespace(namespace string) string {
	if namespace == oah.NamespaceRHOBS {
		return oah.NamespaceOBO
	}
	return namespace
}

func buildNetworkPolicy(ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) (netv1.NetworkPolicy, error) {
	namespacedName := buildNetworkPolicyName(ocmAgent, namespace)

	podSelector, err := callerPodSelector(namespace)
	if err != nil {
		return netv1.NetworkPolicy{}, err
	}

	peer := netv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"kubernetes.io/metadata.name": callerNamespace(namespace)},
		},
		PodSelector: podSelector,
	}

	np := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels: map[string]string{
				"app": ocmAgent.Name,
			},
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": ocmAgent.Name},
			},
			Ingress: []netv1.NetworkPolicyIngressRule{{
				From: []netv1.NetworkPolicyPeer{peer},
			}},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	}

	return np, nil
}

func (o *ocmAgentHandler) ensureAllNetworkPolicies(ctx context.Context, ocmAgent ocmagentv1alpha1.OcmAgent) error {
	var namespaces []string
	if ocmAgent.Spec.FleetMode {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceRHOBS, oah.NamespaceOBO)
	} else {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceMUO)
	}
	for _, ns := range namespaces {
		err := o.ensureNetworkPolicy(ctx, ocmAgent, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

// ensureNetworkPolicy ensures that an OCMAgent NetworkPolicy exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureNetworkPolicy(ctx context.Context, ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) error {

	namespacedName := buildNetworkPolicyName(ocmAgent, namespace)

	foundResource := &netv1.NetworkPolicy{}
	populationFunc := func() (netv1.NetworkPolicy, error) {
		return buildNetworkPolicy(ocmAgent, namespace)
	}
	// Does the resource already exist?
	o.Log.Info("ensuring networkpolicy exists", "resource", namespacedName.String())
	if err := o.Client.Get(ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent NetworkPolicy does not exist; will be created.")
			// Populate the resource with the template
			resource, err := populationFunc()
			if err != nil {
				return err
			}
			// Set the controller reference
			if err := controllerutil.SetControllerReference(&ocmAgent, &resource, o.Scheme); err != nil {
				return err
			}
			// and create it
			err = o.Client.Create(ctx, &resource)
			if err != nil {
				return err
			}
		} else {
			// Return unexpectedly
			return err
		}
	} else {
		// It does exist, check if it is what we expected
		resource, err := populationFunc()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(foundResource.Spec, resource.Spec) {
			// Specs aren't equal, update and fix.
			o.Log.Info("An OCMAgent network policy exists but contains unexpected configuration. Restoring.")
			foundResource.Spec = *resource.Spec.DeepCopy()
			if err = o.Client.Update(ctx, foundResource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureAllNetworkPoliciesDeleted(ctx context.Context, ocmAgent ocmagentv1alpha1.OcmAgent) error {
	var namespaces []string
	if ocmAgent.Spec.FleetMode {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceRHOBS, oah.NamespaceOBO)
	} else {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceMUO)
	}
	for _, ns := range namespaces {
		err := o.ensureNetworkPolicyDeleted(ctx, ocmAgent, ns)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureNetworkPolicyDeleted(ctx context.Context, ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) error {

	namespacedName := buildNetworkPolicyName(ocmAgent, namespace)

	foundResource := &netv1.NetworkPolicy{}
	// Does the resource already exist?
	o.Log.Info("ensuring networkpolicy removed", "resource", namespacedName.String())
	if err := o.Client.Get(ctx, namespacedName, foundResource); err != nil {
		if !k8serrors.IsNotFound(err) {
			// Return unexpected error
			return err
		} else {
			// Resource deleted
			return nil
		}
	}
	err := o.Client.Delete(ctx, foundResource)
	if err != nil {
		return err
	}
	return nil
}
