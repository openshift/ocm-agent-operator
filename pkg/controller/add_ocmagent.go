package controller

import (
	"github.com/openshift/ocm-agent-operator/pkg/controller/ocmagent"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, ocmagent.Add)
}
