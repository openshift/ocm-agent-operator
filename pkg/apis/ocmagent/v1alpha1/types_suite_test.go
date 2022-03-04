package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAPITypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Types Suite")
}
