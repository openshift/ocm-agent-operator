package ocmagenthandler

import (
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/types"

	ns "github.com/openshift/ocm-agent-operator/pkg/util/namespace"
)

const (
	// OCMAgentNamespace is the fall-back namespace to use for OCM Agent Deployments
	OCMAgentNamespace = "openshift-ocm-agent-operator"
	// OCMAgentDefaultNetworkPolicySuffix is the name of the network policy to restrict OCM Agent access
	OCMAgentDefaultNetworkPolicySuffix = "-allow-only-alertmanager"
	// OCMAgentRHOBSNetworkPolicySuffix is the name of the network policy to restrict OA for rhobs
	OCMAgentRHOBSNetworkPolicySuffix = "-allow-rhobs-alertmanager"
	// OCMAgentOBONetworkPolicySuffix is the name of the network policy to restrict OA for obo
	OCMAgentOBONetworkPolicySuffix = "-allow-obo-alertmanager"
	// OCMAgentMUONetworkPolicySuffix is the name of the network policy to restrict OA for MUO
	OCMAgentMUONetworkPolicySuffix = "-allow-muo-communication"
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

	// OCMAgentServicePort is the port number to use for the OCM Agent Service
	OCMAgentServicePort = 8081
	// OCMAgentServiceURLKey defines the key in the configure-alertmanager-operator ConfigMap
	// that contains the OCM Agent service URL
	OCMAgentServiceURLKey = "serviceURL"
	// OCMAgentWebhookReceiverPath is the path of the webhook receiver in the OCM Agent
	OCMAgentWebhookReceiverPath = "/alertmanager-receiver"
	// OCMAgentServiceScheme is the protocol that the OCM Agent will use
	OCMAgentServiceScheme = "http"

	// OCMAgentMetricsServicePort is the port number to use for OCM Agent metrics service
	OCMAgentMetricsServicePort = 8383
	// OCMAgentMetricsPortName is the port name ot use for OCM Agent metrics service
	OCMAgentMetricsPortName = "ocm-agent-metrics"
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
	// OCMAgentConfigClusterID is the name of the key used for the Cluster ID configmap entry
	OCMAgentConfigClusterID = "clusterID"
	// PullSecretKey defines the key in the pull secret containing the auth tokens
	PullSecretKey = ".dockerconfigjson" //#nosec G101 -- This is a false positive
	// PullSecretAuthTokenKey defines the name of the key in the pull secret containing the auth token
	PullSecretAuthTokenKey = "cloud.openshift.com"
	// InjectCaBundleIndicator defines the name of the key for the label of trusted CA bundle configmap
	InjectCaBundleIndicator = "config.openshift.io/inject-trusted-cabundle"
	// TrustedCaBundleConfigMapName TrustedCaBundleConfigMap defines the name of trusted CA bundle configmap
	TrustedCaBundleConfigMapName = "trusted-ca-bundle"
	// ResourceLimitsCPU and ResourceLimitsMemory defines the cpu and memory limits for OA deployment
	ResourceLimitsCPU    = "50m"
	ResourceLimitsMemory = "64Mi"
	// ResourceRequestsCPU and ResourceRequestsMemory defines the cpu and memory requests for OA deployment
	ResourceRequestsCPU    = "1m"
	ResourceRequestsMemory = "30Mi"
	// ConfigMapSuffix is the suffix added to configmap name to always make it unique compared to secret name
	ConfigMapSuffix = "-cm"
	// PDBSuffix is the suffix added to PDB name to always make it unique
	PDBSuffix          = "-pdb"
	NamespaceMonitorng = "openshift-monitoring"
	NamespaceMUO       = "openshift-managed-upgrade-operator"
	NamespaceRHOBS     = "observatorium-mst-production"
	NamespaceOBO       = "openshift-observability-operator"
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

	ProxyNamespacedName = types.NamespacedName{
		Namespace: "",
		Name:      "cluster",
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

func BuildServiceURL(ocmAgentSvcName, ocmAgentNamespace string) (string, error) {
	u := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d%s", OCMAgentServiceScheme,
		ocmAgentSvcName,
		ocmAgentNamespace,
		OCMAgentServicePort,
		OCMAgentWebhookReceiverPath)

	if _, err := url.ParseRequestURI(u); err != nil {
		return "", err
	}
	return u, nil
}
