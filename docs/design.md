# OCM Agent Operator Design

## OCM Agent Controller

The [OCM Agent Controller](https://github.com/openshift/ocm-agent-operator/tree/master/pkg/controller/ocmagent/ocmagent_controller.go) is responsible for ensuring the deployment or removal of an OCM Agent based upon the presence of an `OCMAgent` Custom Resource.

The controller is responsible for managing several resources, outlined below.

### OCM Agent resources

An `OcmAgent` deployment consists of:
- A `ServiceAccount` (named `ocm-agent`)
- A `Role` and `RoleBinding` (both named `ocm-agent`) that defines the OCM Agent's API  permissions.
- A `Deployment` (named `ocm-agent`) which runs the [ocm-agent](https://quay.io/openshift/ocm-agent)
- A `ConfigMap` (name defined in the `OcmAgent` CR) which contains the agent's configuration.
- A `Secret` (name defined in the `OcmAgent` CR) which contains the agent's OCM access token.
- A `Service` (named `ocm-agent`) which serves the OCM Agent API
- A `NetworkPolicy` to only grant ingress from specific cluster clients.
- A `ServiceMonitor` (named `ocm-agent-metrics`) which makes sure that the OCM Agent metrics can be exposed to Prometheus

The controller watches for changes to the above resources in its deployed namespace, in addition to changes to the cluster pull secret (`openshift-config/pull-secret`) which contains the OCM Agent's auth token.

### configure-alertmanager-operator resources

The OCM Agent Controller is also responsible for creating/removing `ConfigMap` resource (named `ocm-agent`) in the `openshift-monitoring` namespace.

This resource is used by the [configure-alertmanager-operator](https://github.com/openshift/configure-alertmanager-operator) to appropriately configure AlertManager to communicate to OCM Agent.

The `ConfigMap` contains the following items:

| Key | Description | Example |
| --- | --- | --- |
| `serviceURL` | OCM Agent service URI | http://ocm-agent.openshift-ocm-agent-operator.svc.cluster.local:8081/alertmanager-receiver |

