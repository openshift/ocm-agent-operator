package ocmagenthandler

import (
	ns "github.com/openshift/ocm-agent-operator/pkg/util/namespace"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// OCMAgentName is the name of the OCM Agent Deployment and its app label identifier.
	OCMAgentName = "ocm-agent"
	// OCMAgentNamespace is the fall-back namespace to use for OCM Agent Deployments
	OCMAgentNamespace = "openshift-ocm-agent-operator"
	// OCMAgentPortName is the name of the OCM Agent service port used in the OCM Agent Deployment
	OCMAgentPortName = "ocm-agent"
	// OCMAgentPort is the container port number used by the agent for exposing its services
	OCMAgentPort = 8081
	// OCMAgentServiceAccount is the name of the service account that will run the OCM Agent
	OCMAgentServiceAccount = "ocm-agent"
	// OCMAgentCommand is the name of the OCM Agent binary to run in the deployment
	OCMAgentCommand = "ocm-agent"

	// OCMAgentServiceName is the name of the Service that serves the OCM Agent
	OCMAgentServiceName = "ocm-agent"
	// OCMAgentServicePort is the port number to use for the OCM Agent Service
	OCMAgentServicePort = 8081

	// OCMAgentSecretMountPath is the base mount path for secrets in the OCM Agent container
	OCMAgentSecretMountPath = "/secrets"
	// OCMAgentAccessTokenSecretKey is the name of the key used in the access token secret
	OCMAgentAccessTokenSecretKey = "access_token"

	// OCMAgentConfigMountPath is the base mount path for configs in the OCM Agent container
	OCMAgentConfigMountPath = "/configs"
	// OCMAgentConfigServicesKey is the name of the key used for the services configmap entry
	OCMAgentConfigServicesKey = "services"
	// OCMAgentConfigURLKey is the name of the key used for the OCM URL configmap entry
	OCMAgentConfigURLKey = "ocmBaseURL"

	// PullSecretKey defines the key in the pull secret containing the auth tokens
	PullSecretKey = ".dockerconfigjson"
	// PullSecretAuthTokenKey defines the name of the key in the pull secret containing the auth token
	PullSecretAuthTokenKey = "cloud.openshift.com"
)

// BuildPullSecretNamespacedName returns the namespaced name of the pull secret
func BuildPullSecretNamespacedName() types.NamespacedName {
	pullSecretNamespacedName := types.NamespacedName{
		Name:      "pull-secret",
		Namespace: "openshift-config",
	}
	return pullSecretNamespacedName
}

// BuildNamespacedName returns the name and namespace intended for OCM Agent deployment resources
func BuildNamespacedName() types.NamespacedName {
	namespace, err := ns.GetOperatorNamespace()
	if err != nil {
		namespace = OCMAgentNamespace
	}
	namespacedName := types.NamespacedName{Name: OCMAgentName, Namespace: namespace}
	return namespacedName
}
