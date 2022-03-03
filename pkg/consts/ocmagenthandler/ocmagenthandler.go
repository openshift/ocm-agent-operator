package ocmagenthandler

import (
	"fmt"
	"net/url"

	ns "github.com/openshift/ocm-agent-operator/pkg/util/namespace"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// OCMAgentName is the name of the OCM Agent Deployment and its app label identifier.
	OCMAgentName = "ocm-agent"
	// OCMAgentNamespace is the fall-back namespace to use for OCM Agent Deployments
	OCMAgentNamespace = "openshift-ocm-agent-operator"
	// OCMAgentNetworkPolicyName is the name of the network policy to restrict OCM Agent access
	OCMAgentNetworkPolicyName = "ocm-agent-allow-only-alertmanager"
	// OCMAgentPortName is the name of the OCM Agent service port used in the OCM Agent Deployment
	OCMAgentPortName = "ocm-agent"
	// OCMAgentPort is the container port number used by the agent for exposing its services
	OCMAgentPort = 8081
	// OCMAgentMetricsPort is the container port number used by the agent for exposing metrics
	OCMAgentMetricsPort = 8383
	// OCMAgentLivezPath is the liveliness probe path
	OCMAgentLivezPath = "/livez"
	// OCMAgentReadyzPath is the readyness probe path
	OCMAgentReadyzPath = "/readyz"
	// OCMAgentServiceAccount is the name of the service account that will run the OCM Agent
	OCMAgentServiceAccount = "ocm-agent"
	// OCMAgentCommand is the name of the OCM Agent binary to run in the deployment
	OCMAgentCommand = "ocm-agent"

	// OCMAgentServiceName is the name of the Service that serves the OCM Agent
	OCMAgentServiceName = "ocm-agent"
	// OCMAgentServicePort is the port number to use for the OCM Agent Service
	OCMAgentServicePort = 8081
	// OCMAgentServiceURLKey defines the key in the configure-alertmanager-operator ConfigMap
	// that contains the OCM Agent service URL
	OCMAgentServiceURLKey = "serviceURL"
	// OCMAgentWebhookReceiverPath is the path of the webhook receiver in the OCM Agent
	OCMAgentWebhookReceiverPath = "/alertmanager-receiver"
	// OCMAgentServiceScheme is the protocol that the OCM Agent will use
	OCMAgentServiceScheme = "http"

	// OCMAgentMetricsServiceName is the name of the service that service the OCM Agent metrics
	OCMAgentMetricsServiceName = "ocm-agent-metrics"
	// OCMAgentMetricsServicePort is the port number to use for OCM Agent metrics service
	OCMAgentMetricsServicePort = 8383
	// OCMAgentMetricsPortName is the port name ot use for OCM Agent metrics service
	OCMAgentMetricsPortName = "ocm-agent-metrics"
	// OCMAgentSecretMountPath is the base mount path for secrets in the OCM Agent container
	OCMAgentSecretMountPath = "/secrets"
	// OCMAgentAccessTokenSecretKey is the name of the key used in the access token secret
	OCMAgentAccessTokenSecretKey = "access_token"
	// OCMAgentServiceMonitorName is the name of the ServiceMonitor for OCM Agent
	OCMAgentServiceMonitorName = "ocm-agent-metrics"
	// OCMAgentConfigMountPath is the base mount path for configs in the OCM Agent container
	OCMAgentConfigMountPath = "/configs"
	// OCMAgentConfigServicesKey is the name of the key used for the services configmap entry
	OCMAgentConfigServicesKey = "services"
	// OCMAgentConfigURLKey is the name of the key used for the OCM URL configmap entry
	OCMAgentConfigURLKey = "ocmBaseURL"
	// OCMAgentConfigClusterID is the name of the key used for the Cluster ID configmap entry
	OCMAgentConfigClusterID = "clusterID"
	// PullSecretKey defines the key in the pull secret containing the auth tokens
	PullSecretKey = ".dockerconfigjson"
	// PullSecretAuthTokenKey defines the name of the key in the pull secret containing the auth token
	PullSecretAuthTokenKey = "cloud.openshift.com"

)

var (
	// PullSecretNamespacedName defines the namespaced name of the cluster pull secret
	PullSecretNamespacedName = types.NamespacedName{
		Namespace: "openshift-config",
		Name:      "pull-secret",
	}

	// CAMOConfigMapNamespacedName defines the namespaced name of the CAMO configmap
	CAMOConfigMapNamespacedName = types.NamespacedName{
		Namespace: "openshift-monitoring",
		Name:      "ocm-agent",
	}
)

// BuildNamespacedName returns the name and namespace intended for OCM Agent deployment resources
func BuildNamespacedName(name string) types.NamespacedName {
	namespace, err := ns.GetOperatorNamespace()
	if err != nil {
		namespace = OCMAgentNamespace
	}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	return namespacedName
}

func BuildServiceURL() (string, error) {
	u := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d%s", OCMAgentServiceScheme,
		OCMAgentServiceName,
		OCMAgentNamespace,
		OCMAgentServicePort,
		OCMAgentWebhookReceiverPath)

	if _, err := url.ParseRequestURI(u); err != nil {
		return "", err
	}
	return u, nil
}
