package ocmagenthandler

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildOCMAgentService(ocmAgent ocmagentv1alpha1.OcmAgent) corev1.Service {
	namespacedName := oah.BuildNamespacedName(oah.OCMAgentServiceName)
	labels := map[string]string{
		"app": oah.OCMAgentName,
	}
	cm := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				TargetPort: intstr.FromInt(oah.OCMAgentPort),
				Name:       oah.OCMAgentPortName,
				Port:       oah.OCMAgentServicePort,
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	return cm
}

// ensureConfigMap ensures that an OCMAgent ConfigMap exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureService(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(oah.OCMAgentServiceName)
	foundResource := &corev1.Service{}
	populationFunc := func() corev1.Service {
		return buildOCMAgentService(ocmAgent)
	}
	// Does the resource already exist?
	o.Log.Info("ensuring service exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent service does not exist; will be created.")
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
		if serviceConfigChanged(foundResource, &resource, o.Log) {
			// Specs aren't equal, update and fix.
			o.Log.Info("An OCMAgent service exists but contains unexpected configuration. Restoring.")
			foundResource.Spec = *resource.Spec.DeepCopy()
			if err = o.Client.Update(o.Ctx, foundResource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureServiceDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(oah.OCMAgentServiceName)
	foundResource := &corev1.Service{}
	// Does the resource already exist?
	o.Log.Info("ensuring service removed", "resource", namespacedName.String())
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

// serviceConfigChanged flags if the two supplied services differ in configuration
// that the OCM Agent Operator manages
func serviceConfigChanged(current, expected *corev1.Service, log logr.Logger) bool {
	changed := false

	if !reflect.DeepEqual(current.Spec.Selector, expected.Spec.Selector) {
		log.V(2).Info(fmt.Sprintf("current service %s/%s did not contain expected selectors", current.Namespace, current.Name))
		changed = true
	}
	if !reflect.DeepEqual(current.Spec.Ports, expected.Spec.Ports) {
		log.V(2).Info(fmt.Sprintf("current service %s/%s did not contain expected ports", current.Namespace, current.Name))
		changed = true
	}
	return changed
}
