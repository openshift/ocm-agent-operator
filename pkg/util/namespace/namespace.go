package namespace

import (
	"fmt"
	"os"
	"strings"
)

// GetOperatorNamespace retrieves the operator namespace from the running environment or error if unavailable
func GetOperatorNamespace() (string, error) {
	envVarOperatorNamespace := "OPERATOR_NAMESPACE"
	ns, found := os.LookupEnv(envVarOperatorNamespace)
	if !found {
		return "", fmt.Errorf("%s must be set", envVarOperatorNamespace)
	}
	return ns, nil
}

// ValidateNamespace checks if a namespace name follows Kubernetes naming conventions
func ValidateNamespace(ns string) error {
	if ns == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	if len(ns) > 63 {
		return fmt.Errorf("namespace length cannot exceed 63 characters")
	}

	// Check if namespace contains only lowercase alphanumeric characters and hyphens
	for _, char := range ns {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return fmt.Errorf("namespace contains invalid character: %c", char)
		}
	}

	// Namespace cannot start or end with hyphen
	if strings.HasPrefix(ns, "-") || strings.HasSuffix(ns, "-") {
		return fmt.Errorf("namespace cannot start or end with hyphen")
	}

	return nil
}

// IsSystemNamespace checks if a namespace is a Kubernetes system namespace
func IsSystemNamespace(ns string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
	}

	for _, sysNs := range systemNamespaces {
		if ns == sysNs {
			return true
		}
	}

	// Check for openshift system namespaces
	if strings.HasPrefix(ns, "openshift-") {
		return true
	}

	return false
}
