package ocmagenthandler

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOCMAgentHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Agent Handler Suite")
}
