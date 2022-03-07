package ocmagenthandler

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	configv1 "github.com/openshift/api/config/v1"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildOCMAgentConfigMap(ocmAgent ocmagentv1alpha1.OcmAgent, clusterId string) *corev1.ConfigMap {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Spec.OcmAgentConfig)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Data: map[string]string{
			oah.OCMAgentConfigServicesKey: strings.Join(ocmAgent.Spec.AgentConfig.Services, ","),
			oah.OCMAgentConfigURLKey:      ocmAgent.Spec.AgentConfig.OcmBaseUrl,
			oah.OCMAgentConfigClusterID:   clusterId,
		},
	}
	return cm
}

func buildTrustedCaConfigMap() *corev1.ConfigMap {
	namespacedName := oah.BuildNamespacedName(oah.TrustedCaBundleConfigMapName)
	labels := map[string]string{
		oah.InjectCaBundleIndicator: "true",
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels:    labels,
		},
	}
	return cm
}

func buildCAMOConfigMap() (*corev1.ConfigMap, error) {
	oaServiceURL, err := oah.BuildServiceURL()
	if err != nil {
		return nil, err
	}
	camoCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oah.CAMOConfigMapNamespacedName.Name,
			Namespace: oah.CAMOConfigMapNamespacedName.Namespace,
		},
		Data: map[string]string{
			oah.OCMAgentServiceURLKey: oaServiceURL,
		},
	}
	return camoCM, nil
}

// ensureAllConfigMaps calls the ensureConfigMap on all the OCM Agent
// managed configmaps
func (o *ocmAgentHandler) ensureAllConfigMaps(ocmAgent ocmagentv1alpha1.OcmAgent) error {

	// Ensure the OCM Agent ConfigMap
	// Determine the cluster ID, used as a configmap value
	cv, err := o.fetchClusterVersion()
	if err != nil {
		o.Log.Error(err, "unable to fetch cluster ID for creating configmap")
		return err
	}
	clusterID := string(cv.Spec.ClusterID)

	oaCM := buildOCMAgentConfigMap(ocmAgent, clusterID)
	err = o.ensureConfigMap(ocmAgent, oaCM, true)
	if err != nil {
		return err
	}

	// Ensure the CAMO ConfigMap
	camoCM, err := buildCAMOConfigMap()
	if err != nil {
		return err
	}
	err = o.ensureConfigMap(ocmAgent, camoCM, false)
	if err != nil {
		return err
	}

	// Ensure the trusted-ca-build ConfigMap
	trustedCACM := buildTrustedCaConfigMap()
	err = o.ensureConfigMap(ocmAgent, trustedCACM, true)
	if err != nil {
		return err
	}

	return nil
}

// ensureConfigMaps ensures that the OCM Agent Operator-managed configmap
// exists on the cluster and that the configuration matches what is expected.
// And apply the ownerReference to the configmaps if needed
func (o *ocmAgentHandler) ensureConfigMap(ocmAgent ocmagentv1alpha1.OcmAgent, cm *corev1.ConfigMap, manager bool) error {

	foundResource := &corev1.ConfigMap{}
	namespacedName := types.NamespacedName{
		Namespace: cm.Namespace,
		Name:      cm.Name,
	}

	// Does the resource already exist?
	o.Log.Info("ensuring configmap exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info(fmt.Sprintf("configmap does not exist, %s/%s; will be created.",
				cm.Namespace,
				cm.Name))
			// Set the controller reference if needed
			if manager {
				if err := controllerutil.SetControllerReference(&ocmAgent, cm, o.Scheme); err != nil {
					return err
				}
			}
			// and create it
			err = o.Client.Create(o.Ctx, cm)
			if err != nil {
				return err
			}
		} else {
			// Return unexpectedly
			return err
		}
	} else {
		// skip update the configmap for trusted-ca-bundle to avoid the race with CNO
		if cm.Name != oah.TrustedCaBundleConfigMapName {
			// It does exist, check if it is what we expected
			if !reflect.DeepEqual(foundResource.Data, cm.Data) {
				// Specs aren't equal, update and fix.
				o.Log.Info(fmt.Sprintf("configmap exists but contains unexpected configuration, %s/%s. Restoring.",
					cm.Namespace, cm.Name))
				foundResource = cm.DeepCopy()
				if err = o.Client.Update(context.TODO(), foundResource); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureAllConfigMapsDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {

	cmsToDelete := []types.NamespacedName{
		oah.BuildNamespacedName(ocmAgent.Spec.OcmAgentConfig),
		oah.CAMOConfigMapNamespacedName,
	}

	for _, cm := range cmsToDelete {
		err := o.ensureConfigMapDeleted(cm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *ocmAgentHandler) ensureConfigMapDeleted(n types.NamespacedName) error {
	foundResource := &corev1.ConfigMap{}
	o.Log.Info("ensuring configmap removed", "resource", n.String())
	// Does the resource already exist?
	if err := o.Client.Get(o.Ctx, n, foundResource); err != nil {
		if !k8serrors.IsNotFound(err) {
			// Return unexpected error
			return err
		} else {
			// Resource deleted
			return nil
		}
	}
	// It does, so remove it
	err := o.Client.Delete(o.Ctx, foundResource)
	if err != nil {
		return err
	}
	return nil
}

func (o *ocmAgentHandler) fetchClusterVersion() (*configv1.ClusterVersion, error) {
	cv := &configv1.ClusterVersion{}
	err := o.Client.Get(o.Ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return nil, err
	}
	return cv, nil
}
