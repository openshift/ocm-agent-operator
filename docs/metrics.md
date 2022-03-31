# Metrics

OCM Agent Operator will create a metrics `Service` and `ServiceMonitor` named 
`ocm-agent-operator-metrics` hosted on port 8686.

The following metrics are produced.

## ocm_agent_operator_pull_secret_invalid

Type: Gauge

Description: This gauge is set to `1` if OCM Agent Operator cannot retrieve or parse the cluster's
`cloud.openshift.com` pull secret, or `0` if it can do so successfully.

Example:
```
ocm_agent_operator_pull_secret_invalid{ocmagent_name="ocmagent"} = 0
```

## ocm_agent_operator_ocm_agent_resource_absent

Type: Gauge

Description: This gauge is set to `1` if OCM Agent Operator cannot find the `OCM Agent` custom resource, 
or `0` if it can do so successfully.

Example:
```
ocm_agent_operator_ocm_agent_resource_absent = 1
```