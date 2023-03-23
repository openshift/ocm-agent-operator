package ocmagenthandler

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/go-logr/logr"

	oconfigv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	"github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildOCMAgentDeployment(ocmAgent ocmagentv1alpha1.OcmAgent) appsv1.Deployment {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name)
	labels := map[string]string{
		"app": ocmAgent.Name,
	}
	labelSelectors := metav1.LabelSelector{
		MatchLabels: labels,
	}

	// Define a volumes for the config
	tokenSecretVolumeName := ocmAgent.Spec.TokenSecret
	configVolumeName := ocmAgent.Name + ocmagenthandler.ConfigMapSuffix
	trustedCaVolumeName := oah.TrustedCaBundleConfigMapName
	var secretVolumeSourceDefaultMode int32 = 0640
	var configVolumeSourceDefaultMode int32 = 0644
	var trustedCaVolumeSourceDefaultMode int32 = 0644

	volumes := []corev1.Volume{
		{
			Name: tokenSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  ocmAgent.Spec.TokenSecret,
					DefaultMode: &secretVolumeSourceDefaultMode,
				},
			},
		},
		{
			Name: configVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ocmAgent.Name + ocmagenthandler.ConfigMapSuffix,
					},
					DefaultMode: &configVolumeSourceDefaultMode,
				},
			},
		},
		{
			Name: trustedCaVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: oah.TrustedCaBundleConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "ca-bundle.crt",
							Path: "tls-ca-bundle.pem",
						},
					},
					DefaultMode: &trustedCaVolumeSourceDefaultMode,
				},
			},
		},
	}

	// define the volume mounts for the deployment
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      tokenSecretVolumeName,
			MountPath: filepath.Join(oah.OCMAgentSecretMountPath, tokenSecretVolumeName),
		},
		{
			Name:      configVolumeName,
			MountPath: filepath.Join(oah.OCMAgentConfigMountPath, configVolumeName),
		},
		{
			Name:      trustedCaVolumeName,
			MountPath: "/etc/pki/ca-trust/extracted/pem",
			ReadOnly:  true,
		},
	}

	envVars := []corev1.EnvVar{
		{Name: "HTTP_PROXY"},
		{Name: "HTTPS_PROXY"},
		{Name: "NO_PROXY"},
		{Name: "OCM_AGENT_SECRET_NAME", Value: ocmAgent.Spec.TokenSecret},
		{Name: "OCM_AGENT_CONFIGMAP_NAME", Value: ocmAgent.Name + ocmagenthandler.ConfigMapSuffix},
	}

	// Sort volume slices by name to keep the sequence stable.
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})
	sort.Slice(volumeMounts, func(i, j int) bool {
		return volumeMounts[i].Name < volumeMounts[j].Name
	})

	// Define resource limits for the config
	resourceLimits := corev1.ResourceList{
		corev1.ResourceCPU:    k8sresource.MustParse(oah.ResourceLimitsCPU),
		corev1.ResourceMemory: k8sresource.MustParse(oah.ResourceLimitsMemory),
	}
	// Define resource requests for the config
	resourceRequests := corev1.ResourceList{
		corev1.ResourceCPU:    k8sresource.MustParse(oah.ResourceRequestsCPU),
		corev1.ResourceMemory: k8sresource.MustParse(oah.ResourceRequestsMemory),
	}

	// Construct the command arguments of the agent
	ocmAgentCommand := buildOCMAgentArgs(ocmAgent)

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
					ServiceAccountName: oah.OCMAgentServiceAccount,
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
						Env:          envVars,
						VolumeMounts: volumeMounts,
						Image:        ocmAgent.Spec.OcmAgentImage,
						Command:      ocmAgentCommand,
						Name:         ocmAgent.Name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: oah.OCMAgentPort,
							Name:          oah.OCMAgentPortName,
						}},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Path:   oah.OCMAgentReadyzPath,
									Port:   intstr.FromInt(oah.OCMAgentPort),
								},
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Path:   oah.OCMAgentLivezPath,
									Port:   intstr.FromInt(oah.OCMAgentPort),
								},
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits:   resourceLimits,
							Requests: resourceRequests,
						},
					}},
				},
			},
		},
	}
	return dep
}

// buildOCMAgentArgs returns the full command argument list to run the OCM Agent
// in a deployment.
func buildOCMAgentArgs(ocmAgent ocmagentv1alpha1.OcmAgent) []string {

	configmapName := ocmAgent.Name + ocmagenthandler.ConfigMapSuffix

	accessTokenPath := filepath.Join(oah.OCMAgentSecretMountPath, ocmAgent.Spec.TokenSecret,
		oah.OCMAgentAccessTokenSecretKey)
	configServicesPath := filepath.Join(oah.OCMAgentConfigMountPath, configmapName,
		oah.OCMAgentConfigServicesKey)
	configURLPath := filepath.Join(oah.OCMAgentConfigMountPath, configmapName,
		oah.OCMAgentConfigURLKey)
	clusterIDPath := filepath.Join(oah.OCMAgentConfigMountPath, configmapName,
		oah.OCMAgentConfigClusterID)
	command := []string{
		oah.OCMAgentCommand,
		"serve",
		fmt.Sprintf("--services=@%s", configServicesPath),
		fmt.Sprintf("--ocm-url=@%s", configURLPath),
	}
	if !ocmAgent.Spec.FleetMode {
		command = append(command, fmt.Sprintf("--cluster-id=@%s", clusterIDPath), fmt.Sprintf("--access-token=@%s", accessTokenPath))
	}
	if ocmAgent.Spec.FleetMode {
		command = append(command, "--fleet-mode")
	}

	return command
}

// ensureDeployment ensures that an OCMAgent Deployment exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureDeployment(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name)
	foundResource := &appsv1.Deployment{}
	populationFunc := func() appsv1.Deployment {
		return buildOCMAgentDeployment(ocmAgent)
	}

	envVars, err := o.buildEnvVars(ocmAgent)
	if err != nil {
		return err
	}

	// Populate the resource with template and append the env vars
	resource := populationFunc()
	resource.Spec.Template.Spec.Containers[0].Env = envVars

	// Does the resource already exist?
	o.Log.Info("ensuring deployment exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent deployment does not exist; will be created.")
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
		if deploymentConfigChanged(foundResource, &resource, ocmAgent, o.Log) {
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
func (o *ocmAgentHandler) ensureDeploymentDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name)
	foundResource := &appsv1.Deployment{}
	// Does the resource already exist?
	o.Log.Info("ensuring deployment removed", "resource", namespacedName.String())
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
func deploymentConfigChanged(current, expected *appsv1.Deployment, ocmAgent ocmagentv1alpha1.OcmAgent, log logr.Logger) bool {
	changed := false

	// Compare labels
	if !reflect.DeepEqual(current.Labels, expected.Labels) {
		return true
	}
	if !reflect.DeepEqual(current.Spec.Template.Labels, expected.Spec.Template.Labels) {
		return true
	}

	// There may be multiple containers eventually, so let's do a loop
	for _, name := range []string{ocmAgent.Name} {
		var curImage, expImage string
		var curReadinessProbeHTTPGet, curLivenessProbeHTTPGet, expReadinessProbeHTTPGet, expLivenessProbeHTTPGet *corev1.HTTPGetAction
		var curEnvs, expEnvs []corev1.EnvVar
		// Assign current container spec
		for i, c := range current.Spec.Template.Spec.Containers {
			if name == c.Name {
				curImage = current.Spec.Template.Spec.Containers[i].Image
				// get current readiness probe HTTPGetter only if ReadinessProbe is set
				if current.Spec.Template.Spec.Containers[i].ReadinessProbe != nil {
					curReadinessProbeHTTPGet = current.Spec.Template.Spec.Containers[i].ReadinessProbe.HTTPGet
				}
				// get current liveness probe HTTPGetter only if LivenessProbe is set
				if current.Spec.Template.Spec.Containers[i].LivenessProbe != nil {
					curLivenessProbeHTTPGet = current.Spec.Template.Spec.Containers[i].LivenessProbe.HTTPGet
				}
				curEnvs = current.Spec.Template.Spec.Containers[i].Env
				break
			}
		}
		// Assign expected container spec
		for i, c := range expected.Spec.Template.Spec.Containers {
			if name == c.Name {
				expImage = expected.Spec.Template.Spec.Containers[i].Image
				expReadinessProbeHTTPGet = expected.Spec.Template.Spec.Containers[i].ReadinessProbe.HTTPGet
				expLivenessProbeHTTPGet = expected.Spec.Template.Spec.Containers[i].LivenessProbe.HTTPGet
				expEnvs = expected.Spec.Template.Spec.Containers[i].Env
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

		// Compare readiness probe change
		if !reflect.DeepEqual(curReadinessProbeHTTPGet, expReadinessProbeHTTPGet) {
			log.V(2).Info(fmt.Sprintf("current readiness probe http getter %s/%s did not match expected readiness http getter", curReadinessProbeHTTPGet, expReadinessProbeHTTPGet))
			changed = true
		}

		// Compare readiness probe change
		if !reflect.DeepEqual(curLivenessProbeHTTPGet, expLivenessProbeHTTPGet) {
			log.V(2).Info(fmt.Sprintf("current liveness probe http getter %s/%s did not match expected liveness probe http getter", curLivenessProbeHTTPGet, expLivenessProbeHTTPGet))
			changed = true
		}

		if !reflect.DeepEqual(curEnvs, expEnvs) {
			log.V(2).Info(fmt.Sprintf("current env vars %s/%s did not match expected env vars", curEnvs, expEnvs))
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

// buildEnvVars build the slice of environments to set to the OCM Agent deployment
func (o *ocmAgentHandler) buildEnvVars(ocmAgent ocmagentv1alpha1.OcmAgent) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
	proxy := oconfigv1.Proxy{}
	err := o.Client.Get(o.Ctx, oah.ProxyNamespacedName, &proxy)
	if err != nil {
		return nil, err
	}

	proxyStatus := proxy.Status
	envVars = append(envVars, corev1.EnvVar{Name: "HTTP_PROXY", Value: proxyStatus.HTTPProxy})
	envVars = append(envVars, corev1.EnvVar{Name: "HTTPS_PROXY", Value: proxyStatus.HTTPSProxy})
	envVars = append(envVars, corev1.EnvVar{Name: "NO_PROXY", Value: proxyStatus.NoProxy})
	envVars = append(envVars, corev1.EnvVar{Name: "OCM_AGENT_SECRET_NAME", Value: ocmAgent.Spec.TokenSecret})
	envVars = append(envVars, corev1.EnvVar{Name: "OCM_AGENT_CONFIGMAP_NAME", Value: ocmAgent.Name + ocmagenthandler.ConfigMapSuffix})

	return envVars, nil
}
