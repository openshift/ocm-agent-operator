package localmetrics

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocalMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Metrics Suite")
}
