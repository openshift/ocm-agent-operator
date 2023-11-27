package ocmagenthandler

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -source $GOFILE -destination ../../pkg/util/test/generated/mocks/$GOPACKAGE/interfaces.go -package mocks

type OcmAgentHandlerBuilder interface {
	New() (OCMAgentHandler, error)
}

type ocmAgentHandlerBuilder struct {
	Client client.Client
}

func NewBuilder(c client.Client) OcmAgentHandlerBuilder {
	return &ocmAgentHandlerBuilder{Client: c}
}

func (oab *ocmAgentHandlerBuilder) New() (OCMAgentHandler, error) {
	log := ctrl.Log.WithName("handler").WithName("OCMAgent")
	ctx := context.Background()
	oaohandler := &ocmAgentHandler{
		Client: oab.Client,
		Log:    log,
		Ctx:    ctx,
		Scheme: oab.Client.Scheme(),
	}
	return oaohandler, nil
}

type OCMAgentHandler interface {
	// EnsureOCMAgentResourcesExist ensures that an OCM Agent is deployed on the cluster.
	EnsureOCMAgentResourcesExist(ocmagentv1alpha1.OcmAgent) error
	// EnsureOCMAgentResourcesAbsent ensures that all OCM Agent resources are removed on the cluster.
	EnsureOCMAgentResourcesAbsent(ocmagentv1alpha1.OcmAgent) error
}

type ensureResource func(agent ocmagentv1alpha1.OcmAgent) error

type ocmAgentHandler struct {
	Client client.Client
	Log    logr.Logger
	Ctx    context.Context
	Scheme *runtime.Scheme
}

func (o *ocmAgentHandler) EnsureOCMAgentResourcesExist(ocmAgent ocmagentv1alpha1.OcmAgent) error {

	var ensureFuncs []ensureResource
	var ensureSecretFunc ensureResource
	if ocmAgent.Spec.FleetMode {
		ensureSecretFunc = o.ensureFleetClientSecret
	} else {
		ensureSecretFunc = o.ensureAccessTokenSecret
	}
	ensureFuncs = []ensureResource{
		o.ensureDeployment,
		o.ensureAllConfigMaps,
		ensureSecretFunc,
		o.ensureService,
		o.ensureAllNetworkPolicies,
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
		o.ensureAllNetworkPoliciesDeleted,
		o.ensureServiceMonitorDeleted,
	}

	if !ocmAgent.Spec.FleetMode {
		ensureFuncs = append(ensureFuncs, o.ensureAccessTokenSecretDeleted)
	}

	for _, fn := range ensureFuncs {
		err := fn(ocmAgent)
		if err != nil {
			return err
		}
	}

	return nil
}
