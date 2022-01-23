package ocmagenthandler

import (
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oahconst "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func buildOCMAgentAccessTokenSecret(accessToken []byte, ocmAgent ocmagentv1alpha1.OcmAgent) corev1.Secret {
	namespacedName := oahconst.BuildNamespacedName()
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ocmAgent.Spec.TokenSecret,
			Namespace: namespacedName.Namespace,
		},
		Data: map[string][]byte{
			oahconst.OCMAgentAccessTokenSecretKey: accessToken,
		},
	}
	return secret
}

// ensureAccessTokenSecret ensures that an OCMAgent Secret exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensureAccessTokenSecret(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oahconst.BuildNamespacedName()
	namespacedName.Name = ocmAgent.Spec.TokenSecret
	foundResource := &corev1.Secret{}

	clusterPullSecret, err := o.fetchAccessTokenPullSecret()
	if err != nil {
		return err
	}
	populationFunc := func() corev1.Secret {
		return buildOCMAgentAccessTokenSecret(clusterPullSecret, ocmAgent)
	}
	// Does the resource already exist?
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent access token secret does not exist; will be created.")
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
		if !reflect.DeepEqual(foundResource.Data, resource.Data) {
			// Specs aren't equal, update and fix.
			o.Log.Info("An OCMAgent access token secret exists but contains unexpected configuration. Restoring.")
			foundResource = resource.DeepCopy()
			if err = o.Client.Update(o.Ctx, foundResource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureAccessTokenSecretDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oahconst.BuildNamespacedName()
	namespacedName.Name = ocmAgent.Spec.TokenSecret
	foundResource := &corev1.Secret{}
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

func (o *ocmAgentHandler) fetchAccessTokenPullSecret() ([]byte, error) {
	foundResource := &corev1.Secret{}
	if err := o.Client.Get(o.Ctx, oahconst.BuildPullSecretNamespacedName(), foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// There should always be a pull secret, log this
			o.Log.Error(err, "Cluster pull secret was not found on the cluster.")
		}
		return nil, err
	}

	pullSecret, ok := foundResource.Data[oahconst.PullSecretKey]
	if !ok {
		return nil, fmt.Errorf("pull secret missing required key '%s'", oahconst.PullSecretKey)
	}

	var dockerConfig map[string]interface{}
	err := json.Unmarshal(pullSecret, &dockerConfig)
	if err != nil {
		o.Log.Error(err, "Unable to interpret decoded pull secret as JSON")
		return nil, err
	}

	authConfig, ok := dockerConfig["auths"]
	if !ok {
		return nil, fmt.Errorf("unable to find auths section in pull secret")
	}
	apiConfig, ok := authConfig.(map[string]interface{})[oahconst.PullSecretAuthTokenKey]
	if !ok {
		return nil, fmt.Errorf("unable to find pull secret auth key '%s' in pull secret", oahconst.PullSecretAuthTokenKey)
	}
	accessToken, ok := apiConfig.(map[string]interface{})["auth"]
	if !ok {
		return nil, fmt.Errorf("unable to find access auth token in pull secret")
	}
	strAccessToken := fmt.Sprintf("%v", accessToken)

	return []byte(strAccessToken), nil
}
