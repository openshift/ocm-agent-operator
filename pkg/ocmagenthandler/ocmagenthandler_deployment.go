package ocmagenthandler

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oahconst "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildOCMAgentDeployment(ocmAgent ocmagentv1alpha1.OcmAgent) appsv1.Deployment {
	namespacedName := oahconst.BuildNamespacedName()
	labels := map[string]string{
		"app": oahconst.OCMAgentName,
	}
	labelSelectors := metav1.LabelSelector{
		MatchLabels: labels,
	}

	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	// Define a volume/volume mount for the access token secret
	var secretVolumeSourceDefaultMode int32 = 0600
	tokenSecretVolumeName := ocmAgent.Spec.TokenSecret
	volumes = append(volumes, corev1.Volume{
		Name: tokenSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  ocmAgent.Spec.TokenSecret,
				DefaultMode: &secretVolumeSourceDefaultMode,
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      tokenSecretVolumeName,
		MountPath: filepath.Join(oahconst.OCMAgentSecretMountPath, tokenSecretVolumeName),
	})

	// Define a volume/volume mount for the config
	configVolumeName := ocmAgent.Spec.OcmAgentConfig
	var configVolumeSourceDefaultMode int32 = 0600
	volumes = append(volumes, corev1.Volume{
		Name: configVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: ocmAgent.Spec.OcmAgentConfig,
				},
				DefaultMode: &configVolumeSourceDefaultMode,
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      configVolumeName,
		MountPath: filepath.Join(oahconst.OCMAgentConfigMountPath, configVolumeName),
	})
	// Sort volume slices by name to keep the sequence stable.
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})
	sort.Slice(volumeMounts, func(i, j int) bool {
		return volumeMounts[i].Name < volumeMounts[j].Name
	})

	replicas := int32(ocmAgent.Spec.Replicas)
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &labelSelectors,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes:            volumes,
					ServiceAccountName: oahconst.OCMAgentServiceAccount,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
								Preference: corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      "node-role.kubernetes.io/infra",
										Operator: corev1.NodeSelectorOpExists,
									}},
								},
								Weight: 1,
							}},
						},
					},
					Tolerations: []corev1.Toleration{{
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
						Key:      "node-role.kubernetes.io/infra",
					}},
					Containers: []corev1.Container{{
						VolumeMounts: volumeMounts,
						Image:        ocmAgent.Spec.OcmAgentImage,
						Command:      []string{"ocm-agent"},
						Name:         oahconst.OCMAgentName,
						Ports: []corev1.ContainerPort{{
							ContainerPort: oahconst.OCMAgentPort,
							Name:          oahconst.OCMAgentPortName,
						}},
					}},
				},
			},
		},
	}
	return dep
}

// ensureDeployment ensures that an OCMAgent Deployment exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureDeployment(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oahconst.BuildNamespacedName()
	foundResource := &appsv1.Deployment{}
	populationFunc := func() appsv1.Deployment {
		return buildOCMAgentDeployment(ocmAgent)
	}
	// Does the resource already exist?
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent deployment does not exist; will be created.")
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
		if deploymentConfigChanged(foundResource, &resource, o.Log) {
			// Specs aren't equal, update and fix.
			o.Log.Info("An OCMAgent deployment exists but contains unexpected configuration. Restoring.")
			foundResource.Spec = *resource.Spec.DeepCopy()
			if err = o.Client.Update(o.Ctx, foundResource); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureDeploymentDeleted removes the deployment from the cluster
func (o *ocmAgentHandler) ensureDeploymentDeleted() error {
	namespacedName := oahconst.BuildNamespacedName()
	foundResource := &appsv1.Deployment{}
	// Does the resource already exist?
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

// deploymentConfigChanged flags if the two supplied deployments differ in configuration
// that the OCM Agent Operator manages
func deploymentConfigChanged(current, expected *appsv1.Deployment, log logr.Logger) bool {
	changed := false

	// There may be multiple containers eventually, so let's do a loop
	for _, name := range []string{oahconst.OCMAgentName} {
		var curImage, expImage string

		for i, c := range current.Spec.Template.Spec.Containers {
			if name == c.Name {
				curImage = current.Spec.Template.Spec.Containers[i].Image
				break
			}
		}
		for i, c := range expected.Spec.Template.Spec.Containers {
			if name == c.Name {
				expImage = expected.Spec.Template.Spec.Containers[i].Image
				break
			}
		}

		if len(curImage) == 0 {
			log.V(2).Info(fmt.Sprintf("current deployment %s/%s did not contain expected %s container", current.Namespace, current.Name, name))
			changed = true
			break
		} else if curImage != expImage {
			changed = true
		}
	}

	// Compare replicas
	if *(current.Spec.Replicas) != *(expected.Spec.Replicas) {
		log.V(2).Info(fmt.Sprintf("current deployment %s/%s did not contain expected replicas: %v", current.Namespace, current.Name, *(expected.Spec.Replicas)))
		changed = true
	}

	// Compare affinity
	if !reflect.DeepEqual(current.Spec.Template.Spec.Affinity, expected.Spec.Template.Spec.Affinity) {
		log.V(2).Info(fmt.Sprintf("current deployment %s/%s did not contain expected affinity", current.Namespace, current.Name))
		changed = true
	}

	// Compare tolerations
	if !reflect.DeepEqual(current.Spec.Template.Spec.Tolerations, expected.Spec.Template.Spec.Tolerations) {
		log.V(2).Info(fmt.Sprintf("current deployment %s/%s did not contain expected tolerations", current.Namespace, current.Name))
		changed = true
	}

	// TODO compare more things if needed

	return changed
}
