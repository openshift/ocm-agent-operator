package ocmagenthandler

import (
	"reflect"

	monitorv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildOCMAgentServiceMonitor(ocmAgent ocmagentv1alpha1.OcmAgent) monitorv1.ServiceMonitor {
	namespacedName := oah.BuildNamespacedName(oah.OCMAgentServiceMonitorName)
	labels := map[string]string{
		"app": oah.OCMAgentName,
	}
	sm := monitorv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: monitorv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Endpoints: []monitorv1.Endpoint{{
				Port: oah.OCMAgentMetricsPortName,
				Path: "/metrics",
			}},
		},
	}
	return sm
}

// ensureServiceMonitor ensures that an OCMAgent serviceMonitor exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureServiceMonitor(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(oah.OCMAgentServiceMonitorName)
	foundResource := &monitorv1.ServiceMonitor{}

	populationFunc := func() monitorv1.ServiceMonitor {
		return buildOCMAgentServiceMonitor(ocmAgent)
	}

	// Does the resource already exist?
	o.Log.Info("ensuring serviceMonitor exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent serviceMonitor does not exist; will be created.")
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
			o.Log.Info("An OCMAgent serviceMonitor exists but contains unexpected configuration. Restoring.")
			foundResource = resource.DeepCopy()
			if err = o.Client.Update(o.Ctx, foundResource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureServiceMonitorDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(oah.OCMAgentServiceMonitorName)
	foundResource := &monitorv1.ServiceMonitor{}
	// Does the resource already exist?
	o.Log.Info("ensuring serviceMonitor removed", "resource", namespacedName.String())
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
