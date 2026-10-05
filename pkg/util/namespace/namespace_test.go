package namespace

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNamespace(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Namespace Suite")
}

var _ = Describe("ValidateNamespace", func() {
	Context("when namespace name is valid", func() {
		It("should accept lowercase alphanumeric names", func() {
			err := ValidateNamespace("my-namespace")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should accept names with numbers", func() {
			err := ValidateNamespace("namespace-123")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should accept single character names", func() {
			err := ValidateNamespace("a")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when namespace name is invalid", func() {
		It("should reject empty namespace", func() {
			err := ValidateNamespace("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be empty"))
		})

		It("should reject namespace longer than 63 characters", func() {
			longName := "this-is-a-very-long-namespace-name-that-exceeds-the-maximum-length-of-63-characters"
			err := ValidateNamespace(longName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot exceed 63 characters"))
		})

		It("should reject namespace with uppercase letters", func() {
			err := ValidateNamespace("MyNamespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid character"))
		})

		It("should reject namespace starting with hyphen", func() {
			err := ValidateNamespace("-namespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot start or end with hyphen"))
		})

		It("should reject namespace ending with hyphen", func() {
			err := ValidateNamespace("namespace-")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot start or end with hyphen"))
		})

		It("should reject namespace with special characters", func() {
			err := ValidateNamespace("namespace_test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid character"))
		})
	})
})

var _ = Describe("IsSystemNamespace", func() {
	Context("when checking Kubernetes system namespaces", func() {
		It("should identify kube-system as system namespace", func() {
			result := IsSystemNamespace("kube-system")
			Expect(result).To(BeTrue())
		})

		It("should identify kube-public as system namespace", func() {
			result := IsSystemNamespace("kube-public")
			Expect(result).To(BeTrue())
		})

		It("should identify kube-node-lease as system namespace", func() {
			result := IsSystemNamespace("kube-node-lease")
			Expect(result).To(BeTrue())
		})

		It("should identify default as system namespace", func() {
			result := IsSystemNamespace("default")
			Expect(result).To(BeTrue())
		})
	})

	Context("when checking OpenShift system namespaces", func() {
		It("should identify openshift-prefixed namespaces as system", func() {
			result := IsSystemNamespace("openshift-monitoring")
			Expect(result).To(BeTrue())
		})

		It("should identify openshift-console as system namespace", func() {
			result := IsSystemNamespace("openshift-console")
			Expect(result).To(BeTrue())
		})
	})

	Context("when checking user namespaces", func() {
		It("should not identify regular namespace as system", func() {
			result := IsSystemNamespace("my-application")
			Expect(result).To(BeFalse())
		})

		It("should not identify namespace with openshift in middle", func() {
			result := IsSystemNamespace("my-openshift-app")
			Expect(result).To(BeFalse())
		})
	})
})
