package ocmagenthandler

import (
	"context"
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

func buildNetworkPolicy(ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) netv1.NetworkPolicy {
	var (
		namespacedName    types.NamespacedName
		namespaceSelector *metav1.LabelSelector
	)

	namespacedName = buildNetworkPolicyName(ocmAgent, namespace)

	namespaceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": namespace},
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
				From: []netv1.NetworkPolicyPeer{{
					NamespaceSelector: namespaceSelector},
				}},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	}

	return np
}

func (o *ocmAgentHandler) ensureAllNetworkPolicies(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	var namespaces []string
	if ocmAgent.Spec.FleetMode {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceRHOBS, oah.NamespaceOBO)
	} else {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceMUO)
	}
	for _, ns := range namespaces {
		err := o.ensureNetworkPolicy(ocmAgent, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

// ensureNetworkPolicy ensures that an OCMAgent NetworkPolicy exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureNetworkPolicy(ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) error {

	namespacedName := buildNetworkPolicyName(ocmAgent, namespace)

	foundResource := &netv1.NetworkPolicy{}
	populationFunc := func() netv1.NetworkPolicy {
		return buildNetworkPolicy(ocmAgent, namespace)
	}
	// Does the resource already exist?
	o.Log.Info("ensuring networkpolicy exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent NetworkPolicy does not exist; will be created.")
			// Populate the resource with the template
			resource := populationFunc()
			// Set the controller reference
			if err := controllerutil.SetControllerReference(&ocmAgent, &resource, o.Scheme); err != nil {
				return err
			}
			// and create it
			err = o.Client.Create(o.Ctx, &resource)
			if err != nil {
				return err
			}
		} else {
			// Return unexpectedly
			return err
		}
	} else {
		// It does exist, check if it is what we expected
		resource := populationFunc()
		if !reflect.DeepEqual(foundResource.Spec, resource.Spec) {
			// Specs aren't equal, update and fix.
			o.Log.Info("An OCMAgent network policy exists but contains unexpected configuration. Restoring.")
			foundResource.Spec = *resource.Spec.DeepCopy()
			if err = o.Client.Update(context.TODO(), foundResource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureAllNetworkPoliciesDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	var namespaces []string
	if ocmAgent.Spec.FleetMode {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceRHOBS, oah.NamespaceOBO)
	} else {
		namespaces = append(namespaces, oah.NamespaceMonitorng, oah.NamespaceMUO)
	}
	for _, ns := range namespaces {
		err := o.ensureNetworkPolicyDeleted(ocmAgent, ns)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureNetworkPolicyDeleted(ocmAgent ocmagentv1alpha1.OcmAgent, namespace string) error {

	namespacedName := buildNetworkPolicyName(ocmAgent, namespace)

	foundResource := &netv1.NetworkPolicy{}
	// Does the resource already exist?
	o.Log.Info("ensuring networkpolicy removed", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if !k8serrors.IsNotFound(err) {
			// Return unexpected error
			return err
		} else {
			// Resource deleted
			return nil
		}
	}
	err := o.Client.Delete(o.Ctx, foundResource)
	if err != nil {
		return err
	}
	return nil
}
