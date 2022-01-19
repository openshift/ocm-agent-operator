package ocmagenthandler

import (
	"context"
	"github.com/go-logr/logr"
	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -source $GOFILE -destination ../../pkg/util/test/generated/mocks/$GOPACKAGE/interfaces.go -package $GOPACKAGE

type OCMAgentHandler interface {
	// EnsureOCMAgentResourcesExist ensures that an OCM Agent is deployed on the cluster.
	EnsureOCMAgentResourcesExist(ocmagentv1alpha1.OcmAgent) error
	// EnsureOCMAgentResourcesAbsent ensures that all OCM Agent resources are removed on the cluster.
	EnsureOCMAgentResourcesAbsent(ocmagentv1alpha1.OcmAgent) error
}

type ocmAgentHandler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	Ctx    context.Context
}

func New(client client.Client, scheme *runtime.Scheme, log logr.Logger, ctx context.Context) OCMAgentHandler {
	return &ocmAgentHandler{
		Client: client,
		Scheme: scheme,
		Log:    log,
		Ctx:    ctx,
	}
}

func (o *ocmAgentHandler) EnsureOCMAgentResourcesExist(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	// Ensure we have a Deployment
	o.Log.V(2).Info("Entering ensureDeployment")
	err := o.ensureDeployment(ocmAgent)
	if err != nil {
		return err
	}

	// Ensure we have a ConfigMap
	o.Log.V(2).Info("Entering ensureConfigMap")
	err = o.ensureConfigMap(ocmAgent)
	if err != nil {
		return err
	}

	// Ensure we have a Secret
	o.Log.V(2).Info("Entering ensureAccessTokenSecret")
	err = o.ensureAccessTokenSecret(ocmAgent)
	if err != nil {
		return err
	}

	// Ensure we have a service
	o.Log.V(2).Info("Entering ensureService")
	err = o.ensureService(ocmAgent)
	if err != nil {
		return err
	}

	// TODO: Ensure we have a ServiceMonitor

	return nil
}

func (o *ocmAgentHandler) EnsureOCMAgentResourcesAbsent(ocmAgent ocmagentv1alpha1.OcmAgent) error {

	// Ensure the deployment is removed
	o.Log.V(2).Info("Entering ensureDeploymentDeleted")
	err := o.ensureDeploymentDeleted()
	if err != nil {
		return err
	}

	// Ensure the service is removed
	o.Log.V(2).Info("Entering ensureServiceDeleted")
	err = o.ensureServiceDeleted()
	if err != nil {
		return err
	}

	// Ensure the configmap is removed
	o.Log.V(2).Info("Entering ensureConfigMapDeleted")
	err = o.ensureConfigMapDeleted(ocmAgent)
	if err != nil {
		return err
	}

	// Ensure the access token secret is removed
	o.Log.V(2).Info("Entering ensureAccessTokenSecretDeleted")
	err = o.ensureAccessTokenSecretDeleted(ocmAgent)
	if err != nil {
		return err
	}

	return nil
}
