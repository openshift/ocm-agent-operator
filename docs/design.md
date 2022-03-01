# OCM Agent Operator Design

## Controllers

### OCMAgent Controller

The [OCMAgent Controller](https://github.com/openshift/ocm-agent-operator/tree/master/pkg/controller/ocmagent/ocmagent_controller.go) is responsible for ensuring the deployment or removal of an OCM Agent based upon the presence of an `OCMAgent` Custom Resource.

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

