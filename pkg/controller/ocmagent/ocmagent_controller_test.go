package ocmagent_test

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	ctrlconst "github.com/openshift/ocm-agent-operator/pkg/consts/controller"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	"github.com/openshift/ocm-agent-operator/pkg/controller/ocmagent"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"
	"github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/ocmagenthandler"
)

var _ = Describe("OCMAgent Controller", func() {
	var (
		mockClient          *clientmocks.MockClient
		mockCtrl            *gomock.Controller
		mockOcmAgentHandler *ocmagenthandler.MockOCMAgentHandler
		ocmAgentReconciler  ocmagent.ReconcileOCMAgent
		testOcmAgent        *ocmagentv1alpha1.OcmAgent
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		mockOcmAgentHandler = ocmagenthandler.NewMockOCMAgentHandler(mockCtrl)
		ocmAgentReconciler = ocmagent.ReconcileOCMAgent{
			Client:          mockClient,
			Ctx:             testconst.Context,
			Log:             testconst.Logger,
			Scheme:          testconst.Scheme,
			OCMAgentHandler: mockOcmAgentHandler,
		}
	})

	Context("Reconciling an OCM Agent CR", func() {
		BeforeEach(func() {
			testOcmAgent = &ocmagentv1alpha1.OcmAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testconst.OCMAgentNamespacedName.Name,
					Namespace: testconst.OCMAgentNamespacedName.Namespace,
				},
				Spec:   ocmagentv1alpha1.OcmAgentSpec{},
				Status: ocmagentv1alpha1.OcmAgentStatus{},
			}
		})

		When("An OCM Agent needs to be created", func() {
			It("Creates an OCM Agent", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testconst.OCMAgentNamespacedName, gomock.Any()).Times(1).SetArg(2, *testOcmAgent),
					mockOcmAgentHandler.EXPECT().EnsureOCMAgentResourcesExist(*testOcmAgent).Times(1),
					mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, o *ocmagentv1alpha1.OcmAgent, opts ...client.UpdateOptions) error {
							Expect(o.Finalizers).To(ContainElement(ctrlconst.ReconcileOCMAgentFinalizer))
							return nil
						}),
				)
				_, err := ocmAgentReconciler.Reconcile(testconst.Context, reconcile.Request{NamespacedName: testconst.OCMAgentNamespacedName})
				Expect(err).To(BeNil())
				Expect(ocmAgentReconciler.Log.Enabled()).To(BeFalse())
			})
		})

		When("An OCM Agent needs to be deleted", func() {
			BeforeEach(func() {
				testOcmAgent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				testOcmAgent.Finalizers = []string{
					ctrlconst.ReconcileOCMAgentFinalizer,
				}
			})
			It("Deletes an OCM Agent", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testconst.OCMAgentNamespacedName, gomock.Any()).Times(1).SetArg(2, *testOcmAgent),
					mockOcmAgentHandler.EXPECT().EnsureOCMAgentResourcesAbsent(gomock.Any()).Times(1),
					mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
					func(ctx context.Context, o *ocmagentv1alpha1.OcmAgent, opts ...client.UpdateOptions) error {
						Expect(o.Finalizers).NotTo(ContainElement(ctrlconst.ReconcileOCMAgentFinalizer))
						return nil
					}),
				)
				_, err := ocmAgentReconciler.Reconcile(testconst.Context, reconcile.Request{NamespacedName: testconst.OCMAgentNamespacedName})
				Expect(err).To(BeNil())
				Expect(ocmAgentReconciler.Log.Enabled()).To(BeFalse())
			})
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

})
