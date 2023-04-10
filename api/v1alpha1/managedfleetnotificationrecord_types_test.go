package v1alpha1_test

import (
	"time"

	"github.com/openshift/ocm-agent-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCMAgent Controller MFNR Type", func() {

	var (
		testMNFR *v1alpha1.ManagedFleetNotificationRecord
	)

	BeforeEach(func() {
		testMNFR = &v1alpha1.ManagedFleetNotificationRecord{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-mc-id",
				Namespace: "openshift-ocm-agent-operator",
			},
			Status: v1alpha1.ManagedFleetNotificationRecordStatus{
				ManagementCluster: "test-mc-id",
				NotificationRecordByName: []v1alpha1.NotificationRecordByName{
					{
						NotificationName: "test-notification-1",
						ResendWait:       1,
						NotificationRecordItems: []v1alpha1.NotificationRecordItem{
							{
								HostedClusterID:     "test-hc-1-1",
								ServiceLogSentCount: 0,
								LastTransitionTime:  nil,
							},
							{
								HostedClusterID:     "test-hc-1-2",
								ServiceLogSentCount: 1,
								LastTransitionTime:  &metav1.Time{Time: time.Now().Add(time.Duration(-5) * time.Hour)},
							},
							{
								HostedClusterID:     "test-hc-1-3",
								ServiceLogSentCount: 0,
								LastTransitionTime:  nil,
							},
						},
					},
					{
						NotificationName: "test-notification-2",
						ResendWait:       24,
						NotificationRecordItems: []v1alpha1.NotificationRecordItem{
							{
								HostedClusterID:     "test-hc-2-1",
								ServiceLogSentCount: 0,
								LastTransitionTime:  nil,
							},
						},
					},
				},
			},
		}
	})

	Context("When updating a notification record item", func() {
		Context("When the notification record item already exists", func() {
			It("will update it correctly", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				nri := testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[1]
				nri2, err := testMNFR.UpdateNotificationRecordItem(nr.NotificationName, nri.HostedClusterID)
				Expect(err).To(BeNil())
				Expect(nri2.ServiceLogSentCount).To(Equal(2))
				Expect(testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[1].ServiceLogSentCount).To(Equal(2))
			})
		})
		Context("When the notification does not exist", func() {
			It("will return an error", func() {
				_, err := testMNFR.UpdateNotificationRecordItem("nope", "nope")
				Expect(err).NotTo(BeNil())
			})
		})
		Context("When the notification record item does not exist", func() {
			It("will return an error", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				_, err := testMNFR.UpdateNotificationRecordItem(nr.NotificationName, "nope")
				Expect(err).NotTo(BeNil())
			})
		})
	})

	Context("When adding a notification record item", func() {
		Context("If the notification record item doesn't exists", func() {
			It("will add the item", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				nfi, err := testMNFR.AddNotificationRecordItem("test-hc-1-4", &nr)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems)).To(Equal(4))
				Expect(nfi.HostedClusterID).To(Equal("test-hc-1-4"))
			})
		})
		Context("If the notification record item already exists", func() {
			It("will error", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				nfi, err := testMNFR.AddNotificationRecordItem("test-hc-1-3", &nr)
				Expect(err).To(HaveOccurred())
				Expect(nfi).To(BeNil())
			})
		})
		Context("If the notification record doesn't exist", func() {
			It("will error", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				// intentionally bork the notification record name - this won't change the MNFR
				nr.NotificationName = "wont-find-this"
				nfi, err := testMNFR.AddNotificationRecordItem("test-hc-1-3", &nr)
				Expect(err).To(HaveOccurred())
				Expect(nfi).To(BeNil())
			})
		})
	})

	Context("When removing a notification record item", func() {
		Context("When the notification record item already exists", func() {
			It("will update it correctly", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				nri := testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[1]
				nr2, err := testMNFR.RemoveNotificationRecordItem(nr.NotificationName, nri.HostedClusterID)
				Expect(err).To(BeNil())
				Expect(len(testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems)).To(Equal(2))
				Expect(testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[0].HostedClusterID).To(Equal("test-hc-1-1"))
				Expect(testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[1].HostedClusterID).To(Equal("test-hc-1-3"))
				Expect(nr2.NotificationRecordItems[0].HostedClusterID).To(Equal("test-hc-1-1"))
				Expect(nr2.NotificationRecordItems[1].HostedClusterID).To(Equal("test-hc-1-3"))
			})
		})
		Context("When the notification does not exist", func() {
			It("will return an error", func() {
				_, err := testMNFR.RemoveNotificationRecordItem("nope", "nope")
				Expect(err).NotTo(BeNil())
			})
		})
		Context("When the notification record item does not exist", func() {
			It("will return an error", func() {
				nr := testMNFR.Status.NotificationRecordByName[0]
				_, err := testMNFR.RemoveNotificationRecordItem(nr.NotificationName, "nope")
				Expect(err).NotTo(BeNil())
			})
		})
	})

	Context("When checking if a firing notification can be sent", func() {

		When("there is no defined notification", func() {
			It("will raise an error", func() {
				cansend, err := testMNFR.CanBeSent("test-mc-id-1", "test-notification-1", "test-hc-1-1")
				Expect(cansend).To(BeFalse())
				Expect(err).To(HaveOccurred())
			})
		})

		When("there is no notification history for the notification", func() {
			BeforeEach(func() {
				testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems = []v1alpha1.NotificationRecordItem{}
			})
			It("will send", func() {
				cansend, err := testMNFR.CanBeSent("test-mc-id","test-notification-1", "test-hc-1-1")
				Expect(cansend).To(BeTrue())
				Expect(err).To(BeNil())
			})
		})

		When("the current time is within the dont-resend window", func() {
			BeforeEach(func() {
				testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[0] = v1alpha1.NotificationRecordItem{
					LastTransitionTime: &metav1.Time{Time: time.Now().Add(time.Duration(-5) * time.Minute)},
					HostedClusterID: "test-hc-1-1" ,
				} 
			})
			It("will not resend", func() {
				cansend, err := testMNFR.CanBeSent("test-mc-id","test-notification-1", "test-hc-1-1")
				Expect(cansend).To(BeFalse())
				Expect(err).To(BeNil())
			})
		})

		When("the current time is outside the dont-resend window", func() {
			BeforeEach(func() {
				testMNFR.Status.NotificationRecordByName[0].NotificationRecordItems[2] = v1alpha1.NotificationRecordItem{
					LastTransitionTime: &metav1.Time{Time: time.Now().Add(time.Duration(-5) * time.Hour)},
				} 
			})
			It("will resend notification", func() {
				cansend, err := testMNFR.CanBeSent("test-mc-id","test-notification-1", "test-hc-1-1")
				Expect(cansend).To(BeTrue())
				Expect(err).To(BeNil())
			})
		})
	})

	
})
