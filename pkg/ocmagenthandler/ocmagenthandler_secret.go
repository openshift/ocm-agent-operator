package ocmagenthandler

import (
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	"github.com/openshift/ocm-agent-operator/pkg/localmetrics"
)

func buildOCMAgentAccessTokenSecret(accessToken []byte, ocmAgent ocmagentv1alpha1.OcmAgent) corev1.Secret {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Spec.TokenSecret)
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Data: map[string][]byte{
			oah.OCMAgentAccessTokenSecretKey: accessToken,
		},
	}
	return secret
}

// ensureAccessTokenSecret ensures that an OCMAgent Secret exists on the cluster
// and that its configuration matches what is expected.
// Returns (wasUpdated bool, error) where wasUpdated indicates if the secret was created or updated.
func (o *ocmAgentHandler) ensureAccessTokenSecret(ocmAgent ocmagentv1alpha1.OcmAgent) (bool, error) {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Spec.TokenSecret)
	foundResource := &corev1.Secret{}

	clusterPullSecret, err := o.fetchAccessTokenPullSecret()
	if err != nil {
		o.Log.Error(err, "Failed to fetch pull-secret")
		localmetrics.UpdateMetricPullSecretInvalid(ocmAgent.Name)
		return false, err
	}
	localmetrics.ResetMetricPullSecretInvalid(ocmAgent.Name)

	populationFunc := func() corev1.Secret {
		return buildOCMAgentAccessTokenSecret(clusterPullSecret, ocmAgent)
	}

	// Does the resource already exist?
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			// Populate the resource with the template
			resource := populationFunc()
			// Set the controller reference
			if err := controllerutil.SetControllerReference(&ocmAgent, &resource, o.Scheme); err != nil {
				o.Log.Error(err, "Failed to set controller reference")
				return false, err
			}
			// and create it
			err = o.Client.Create(o.Ctx, &resource)
			if err != nil {
				o.Log.Error(err, "Failed to create secret")
				return false, err
			}
			o.Log.Info("Created ocm-access-token secret", "secret", namespacedName.String())
			return true, nil // Secret was created
		} else {
			// Return unexpectedly
			o.Log.Error(err, "Unexpected error fetching secret")
			return false, err
		}
	} else {
		// It does exist, check if it is what we expected
		resource := populationFunc()
		if !reflect.DeepEqual(foundResource.Data, resource.Data) {
			// Specs aren't equal, update and fix.
			foundResource = resource.DeepCopy()
			if err = o.Client.Update(o.Ctx, foundResource); err != nil {
				o.Log.Error(err, "Failed to update secret")
				return false, err
			}
			o.Log.Info("Updated ocm-access-token secret", "secret", namespacedName.String())
			return true, nil // Secret was updated
		}
	}
	return false, nil // Secret exists and matches expected state, no update needed
}

func (o *ocmAgentHandler) ensureFleetClientSecret(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Spec.TokenSecret)
	foundResource := &corev1.Secret{}
	// Does the resource already exist?
	o.Log.Info("ensuring fleetmode secret exists", "resource", namespacedName.String())
	if err := o.Client.Get(o.Ctx, namespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// It does not exist, so must be created.
			o.Log.Info("An OCMAgent secret for Hypershift does not exist. Fleet mode OCMAgent will not work as expected")
			return err
		} else {
			// Return unexpectedly
			return err
		}
	}
	return nil
}

func (o *ocmAgentHandler) ensureAccessTokenSecretDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Spec.TokenSecret)
	foundResource := &corev1.Secret{}
	// Does the resource already exist?
	o.Log.Info("ensuring secret removed", "resource", namespacedName.String())
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
	if err := o.Client.Get(o.Ctx, oah.PullSecretNamespacedName, foundResource); err != nil {
		if k8serrors.IsNotFound(err) {
			// There should always be a pull secret, log this
			o.Log.Error(err, "Cluster pull secret was not found on the cluster")
		} else {
			o.Log.Error(err, "Failed to get pull-secret")
		}
		return nil, err
	}

	pullSecret, ok := foundResource.Data[oah.PullSecretKey]
	if !ok {
		o.Log.Error(nil, "Pull secret missing required key", "key", oah.PullSecretKey)
		return nil, fmt.Errorf("pull secret missing required key '%s'", oah.PullSecretKey)
	}

	var dockerConfig map[string]interface{}
	err := json.Unmarshal(pullSecret, &dockerConfig)
	if err != nil {
		o.Log.Error(err, "Unable to interpret decoded pull secret as JSON")
		return nil, err
	}

	authConfig, ok := dockerConfig["auths"]
	if !ok {
		o.Log.Error(nil, "Unable to find auths section in pull secret")
		return nil, fmt.Errorf("unable to find auths section in pull secret")
	}

	apiConfig, ok := authConfig.(map[string]interface{})[oah.PullSecretAuthTokenKey]
	if !ok {
		o.Log.Error(nil, "Unable to find pull secret auth key", "key", oah.PullSecretAuthTokenKey)
		return nil, fmt.Errorf("unable to find pull secret auth key '%s' in pull secret", oah.PullSecretAuthTokenKey)
	}

	accessToken, ok := apiConfig.(map[string]interface{})["auth"]
	if !ok {
		o.Log.Error(nil, "Unable to find access auth token in pull secret")
		return nil, fmt.Errorf("unable to find access auth token in pull secret")
	}
	strAccessToken := fmt.Sprintf("%v", accessToken)

	return []byte(strAccessToken), nil
}
