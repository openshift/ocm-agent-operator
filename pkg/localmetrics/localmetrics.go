package localmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricsTag = "ocm_agent_operator"
	nameLabel  = "ocmagent_name"
)

var (
	MetricPullSecretInvalid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "pull_secret_invalid",
		Help:      "Failed to obtain a valid pull secret",
	}, []string{nameLabel})

	MetricFleetSecretAbsent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "fleet_secret_absent",
		Help:      "Failed to obtain a vault secret for OCM service account",
	}, []string{nameLabel})

	MetricOcmAgentResourceAbsent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: metricsTag,
		Name:      "ocm_agent_resource_absent",
		Help:      "No OCM Agent resource found",
	}, []string{})

	MetricsList = []prometheus.Collector{
		MetricPullSecretInvalid,
		MetricOcmAgentResourceAbsent,
		MetricFleetSecretAbsent,
	}
)

func UpdateMetricPullSecretInvalid(ocmAgentName string) {
	MetricPullSecretInvalid.With(prometheus.Labels{
		nameLabel: ocmAgentName}).Set(float64(1))
}

func UpdateMetricFleetSecretAbsent(ocmAgentName string) {
	MetricFleetSecretAbsent.With(prometheus.Labels{
		nameLabel: ocmAgentName}).Set(float64(1))
}

func UpdateMetricOcmAgentResourceAbsent() {
	MetricOcmAgentResourceAbsent.WithLabelValues().Set(
		float64(1))
}

func ResetMetricPullSecretInvalid(ocmAgentName string) {
	MetricPullSecretInvalid.With(prometheus.Labels{
		nameLabel: ocmAgentName}).Set(float64(0))
}

func ResetMetricFleetSecretAbsent(ocmAgentName string) {
	MetricFleetSecretAbsent.With(prometheus.Labels{
		nameLabel: ocmAgentName}).Set(float64(0))
}

func ResetMetricOcmAgentResourceAbsent() {
	MetricOcmAgentResourceAbsent.WithLabelValues().Set(float64(0))
}
