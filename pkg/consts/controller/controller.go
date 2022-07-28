package controller

import (
	"time"
)

const (
	// SyncPeriodDefault reconciles a sync period for each controller
	SyncPeriodDefault = 5 * time.Minute

	// ReconcileOCMAgentFinalizer defines the finalizer to apply to the OCM Agent resource
	ReconcileOCMAgentFinalizer = "ocmagent.managed.openshift.io"
)