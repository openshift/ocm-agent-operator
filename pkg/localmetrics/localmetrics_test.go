package localmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
)

var _ = Describe("LocalMetrics", func() {
	var (
		testOcmAgentName string
	)

	BeforeEach(func() {
		testOcmAgentName = testconst.TestOCMAgent.Name
		// Reset all metrics before each test
		ResetMetricPullSecretInvalid(testOcmAgentName)
		ResetMetricOcmAgentResourceAbsent()
	})

	Context("When updating MetricPullSecretInvalid", func() {
		It("should set metric to 1 when pull secret is invalid", func() {
			// Update the metric
			UpdateMetricPullSecretInvalid(testOcmAgentName)

			// Verify the metric value is set to 1
			metricValue := getGaugeValue(MetricPullSecretInvalid, prometheus.Labels{
				nameLabel: testOcmAgentName,
			})
			Expect(metricValue).To(Equal(float64(1)))
		})

	})

	Context("When updating MetricOcmAgentResourceAbsent", func() {
		It("should set metric to 1 when OCM agent resource is absent", func() {
			// Update the metric
			UpdateMetricOcmAgentResourceAbsent()

			// Verify the metric value is set to 1
			metricValue := getGaugeValueNoLabels(MetricOcmAgentResourceAbsent)
			Expect(metricValue).To(Equal(float64(1)))
		})

	})

	Context("When resetting MetricPullSecretInvalid", func() {

		It("should reset metric to 0 from non-zero value", func() {
			// First set the metric to 1
			UpdateMetricPullSecretInvalid(testOcmAgentName)
			initialValue := getGaugeValue(MetricPullSecretInvalid, prometheus.Labels{
				nameLabel: testOcmAgentName,
			})
			Expect(initialValue).To(Equal(float64(1)))

			// Reset the metric
			ResetMetricPullSecretInvalid(testOcmAgentName)

			// Verify the metric value is now 0
			resetValue := getGaugeValue(MetricPullSecretInvalid, prometheus.Labels{
				nameLabel: testOcmAgentName,
			})
			Expect(resetValue).To(Equal(float64(0)))
		})
	})

	Context("When resetting MetricOcmAgentResourceAbsent", func() {
		It("should reset metric to 0 from non-zero value", func() {
			// First set the metric to 1
			UpdateMetricOcmAgentResourceAbsent()
			initialValue := getGaugeValueNoLabels(MetricOcmAgentResourceAbsent)
			Expect(initialValue).To(Equal(float64(1)))

			// Reset the metric
			ResetMetricOcmAgentResourceAbsent()

			// Verify the metric value is now 0
			resetValue := getGaugeValueNoLabels(MetricOcmAgentResourceAbsent)
			Expect(resetValue).To(Equal(float64(0)))
		})
	})
})

// Helper function to get gauge value with labels
func getGaugeValue(gaugeVec *prometheus.GaugeVec, labels prometheus.Labels) float64 {
	gauge := gaugeVec.With(labels)
	metric := &dto.Metric{}
	err := gauge.Write(metric)
	if err != nil {
		return -1 // Return -1 to indicate error
	}
	return metric.GetGauge().GetValue()
}

// Helper function to get gauge value without labels
func getGaugeValueNoLabels(gaugeVec *prometheus.GaugeVec) float64 {
	gauge := gaugeVec.WithLabelValues()
	metric := &dto.Metric{}
	err := gauge.Write(metric)
	if err != nil {
		return -1 // Return -1 to indicate error
	}
	return metric.GetGauge().GetValue()
}
