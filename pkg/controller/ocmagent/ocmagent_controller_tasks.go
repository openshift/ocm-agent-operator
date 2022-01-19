package ocmagent

import (
	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
)

func (r *ReconcileOCMAgent) EnsureFinalizerSet(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	return nil
}

func (r *ReconcileOCMAgent) EnsureServiceMonitorExists(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	return nil
}

func (r *ReconcileOCMAgent) EnsureConfigMapExists(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	return nil
}

func (r *ReconcileOCMAgent) EnsureAgentRemoved(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	return nil
}

