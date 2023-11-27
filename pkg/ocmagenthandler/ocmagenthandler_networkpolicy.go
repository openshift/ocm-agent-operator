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

func buildNetworkPolicy(ocmAgent ocmagentv1alpha1.OcmAgent) netv1.NetworkPolicy {
	var (
		namespacedName    types.NamespacedName
		namespaceSelector *metav1.LabelSelector
	)
	if ocmAgent.Spec.FleetMode {
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMFleetAgentNetworkPolicySuffix)
		namespaceSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "name",
				Operator: "In",
				Values:   []string{"observatorium-mst-production", "openshift-monitoring"},
			}},
		}
	} else {
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentNetworkPolicySuffix)
		namespaceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"name": "openshift-monitoring"},
		}
	}

	np := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
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

func buildNetworkPolicyForMUO(ocmAgent ocmagentv1alpha1.OcmAgent) netv1.NetworkPolicy {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name + "-allow-muo-communication")

	np := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": ocmAgent.Name},
			},
			Ingress: []netv1.NetworkPolicyIngressRule{
				{
					From: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-managed-upgrade-operator"},
							},
						},
					},
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	}

	return np
}

func (o *ocmAgentHandler) ensureAllNetworkPolicies(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	err := o.ensureNetworkPolicy(ocmAgent)
	if err != nil {
		return err
	}

	// Ensure MUO network policy
	err = o.ensureNetworkPolicyForMUO(ocmAgent)
	if err != nil {
		return err
	}

	return nil
}

// ensureNetworkPolicy ensures that an OCMAgent NetworkPolicy exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureNetworkPolicy(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	var namespacedName types.NamespacedName
	if ocmAgent.Spec.FleetMode {
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMFleetAgentNetworkPolicySuffix)
	} else {
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentNetworkPolicySuffix)
	}
	foundResource := &netv1.NetworkPolicy{}
	populationFunc := func() netv1.NetworkPolicy {
		return buildNetworkPolicy(ocmAgent)
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

func (o *ocmAgentHandler) ensureNetworkPolicyForMUO(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name + "-allow-muo-communication")
	foundResource := &netv1.NetworkPolicy{}

	// Check if the resource already exist
	o.Log.Info("ensuring MUO networkpolicy exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// If Network policy doesn't exist, create it.
			o.Log.Info("MUO NetworkPolicy does not exist; will be created.")
			resource := buildNetworkPolicyForMUO(ocmAgent)
			if err := controllerutil.SetControllerReference(&ocmAgent, &resource, o.Scheme); err != nil {
				return err
			}
			return o.Client.Create(o.Ctx, &resource)
		}
		return err
	} else {
		// If Network policy exists, check if it needs updating.
		resource := buildNetworkPolicyForMUO(ocmAgent)
		if !reflect.DeepEqual(foundResource.Spec, resource.Spec) {
			o.Log.Info("MUO NetworkPolicy exists but is outdated. Updating.")
			foundResource.Spec = *resource.Spec.DeepCopy()
			return o.Client.Update(context.TODO(), foundResource)
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureAllNetworkPoliciesDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	err := o.ensureNetworkPolicyDeleted(ocmAgent)
	if err != nil {
		return err
	}

	// Delete MUO network policy
	err = o.ensureNetworkPolicyForMUODeleted(ocmAgent)
	if err != nil {
		return err
	}

	return nil
}

func (o *ocmAgentHandler) ensureNetworkPolicyDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	var namespacedName types.NamespacedName
	if ocmAgent.Spec.FleetMode {
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMFleetAgentNetworkPolicySuffix)
	} else {
		namespacedName = oah.BuildNamespacedName(ocmAgent.Name + oah.OCMAgentNetworkPolicySuffix)
	}
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

func (o *ocmAgentHandler) ensureNetworkPolicyForMUODeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name + "-allow-muo-communication")
	foundResource := &netv1.NetworkPolicy{}

	// Check if the network policy exists
	o.Log.Info("ensuring MUO networkpolicy is removed", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	// Delete the network policy
	return o.Client.Delete(o.Ctx, foundResource)
}
