package fleetnotification_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFleetNotificationRecord(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FleetNotification Controller Suite")
}
