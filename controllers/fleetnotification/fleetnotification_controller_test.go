package fleetnotification_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"

	"github.com/golang/mock/gomock"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	"github.com/openshift/ocm-agent-operator/controllers/fleetnotification"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"
)

var _ = Describe("FleetNotification Controller", func() {
	var (
		mockClient                  *clientmocks.MockClient
		mockStatusWriter            *clientmocks.MockStatusWriter
		mockCtrl                    *gomock.Controller
		fleetNotificationReconciler *fleetnotification.ManagedFleetNotificationReconciler
		testFleetNotificationRecord *ocmagentv1alpha1.ManagedFleetNotificationRecord
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		mockStatusWriter = clientmocks.NewMockStatusWriter(mockCtrl)
		fleetNotificationReconciler = &fleetnotification.ManagedFleetNotificationReconciler{
			Client: mockClient,
			Scheme: testconst.Scheme,
		}
	})

	Context("Reconcile ManagedFleetNotificationRecord CR", func() {
		BeforeEach(func() {
			testFleetNotificationRecord = &ocmagentv1alpha1.ManagedFleetNotificationRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testconst.MfnrNamespacedName.Name,
					Namespace: testconst.MfnrNamespacedName.Namespace,
				},
				Status: ocmagentv1alpha1.ManagedFleetNotificationRecordStatus{},
			}
		})

		When("There is no notificationrecord items", func() {
			It("Won't need to do the garbage collection", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testconst.MfnrNamespacedName, gomock.Any()).Times(1).SetArg(2, *testFleetNotificationRecord),
				)
				_, err := fleetNotificationReconciler.Reconcile(testconst.Context, reconcile.Request{NamespacedName: testconst.MfnrNamespacedName})
				Expect(err).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("All the notificationrecord items are updated recently", func() {
			BeforeEach(func() {
				testFleetNotificationRecord = &ocmagentv1alpha1.ManagedFleetNotificationRecord{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testconst.MfnrNamespacedName.Name,
						Namespace: testconst.MfnrNamespacedName.Namespace,
					},
					Status: ocmagentv1alpha1.ManagedFleetNotificationRecordStatus{
						NotificationRecordByName: []ocmagentv1alpha1.NotificationRecordByName{
							{
								NotificationRecordItems: []ocmagentv1alpha1.NotificationRecordItem{
									{
										HostedClusterID:     "1234-5678-12345678",
										ServiceLogSentCount: 1,
										LastTransitionTime:  &metav1.Time{Time: time.Now()},
									},
								},
							},
						},
					},
				}
			})
			It("Won't need to do the garbage collection", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testconst.MfnrNamespacedName, gomock.Any()).Times(1).SetArg(2, *testFleetNotificationRecord),
				)
				_, err := fleetNotificationReconciler.Reconcile(testconst.Context, reconcile.Request{NamespacedName: testconst.MfnrNamespacedName})
				Expect(err).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("There is notification record which was sent before and stale", func() {
			BeforeEach(func() {
				testFleetNotificationRecord = &ocmagentv1alpha1.ManagedFleetNotificationRecord{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testconst.MfnrNamespacedName.Name,
						Namespace: testconst.MfnrNamespacedName.Namespace,
					},
					Status: ocmagentv1alpha1.ManagedFleetNotificationRecordStatus{
						NotificationRecordByName: []ocmagentv1alpha1.NotificationRecordByName{
							{
								NotificationRecordItems: []ocmagentv1alpha1.NotificationRecordItem{
									{
										HostedClusterID:     "1234-5678-12345678",
										ServiceLogSentCount: 2,
										LastTransitionTime:  &metav1.Time{Time: time.Date(2022, time.November, 10, 23, 0, 0, 0, time.UTC)},
									},
								},
							},
						},
					},
				}
			})
			It("Will need to do the garbage collection for it", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testconst.MfnrNamespacedName, gomock.Any()).Times(1).SetArg(2, *testFleetNotificationRecord),
					mockClient.EXPECT().Status().Return(mockStatusWriter),
					mockStatusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(1, *testFleetNotificationRecord),
				)
				_, err := fleetNotificationReconciler.Reconcile(testconst.Context, reconcile.Request{NamespacedName: testconst.MfnrNamespacedName})
				Expect(err).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
