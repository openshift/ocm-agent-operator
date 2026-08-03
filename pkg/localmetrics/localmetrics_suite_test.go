package localmetrics

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLocalMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Metrics Suite")
}
