package ocmagent_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOCMAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCMAgent Controller Suite")
}
