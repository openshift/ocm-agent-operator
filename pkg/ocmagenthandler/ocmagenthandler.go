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

type ensureResource func(agent ocmagentv1alpha1.OcmAgent) error

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

	ensureFuncs := []ensureResource{
		o.ensureDeployment,
		o.ensureAllConfigMaps,
		o.ensureAccessTokenSecret,
		o.ensureService,
		o.ensureNetworkPolicy,
		o.ensureServiceMonitor,
	}

	for _, fn := range ensureFuncs {
		err := fn(ocmAgent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *ocmAgentHandler) EnsureOCMAgentResourcesAbsent(ocmAgent ocmagentv1alpha1.OcmAgent) error {

	ensureFuncs := []ensureResource{
		o.ensureDeploymentDeleted,
		o.ensureServiceDeleted,
		o.ensureAllConfigMapsDeleted,
		o.ensureAccessTokenSecretDeleted,
		o.ensureNetworkPolicyDeleted,
		o.ensureServiceMonitorDeleted,
	}

	for _, fn := range ensureFuncs {
		err := fn(ocmAgent)
		if err != nil {
			return err
		}
	}

	return nil
}
