package ocmagenthandler

import (
	"context"
	"reflect"

	monitorv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"

	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent ServiceMonitor Handler", func() {
	var (
		mockClient *clientmocks.MockClient
		mockCtrl   *gomock.Controller

		testOcmAgent        ocmagentv1alpha1.OcmAgent
		testOcmAgentHandler ocmAgentHandler
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		testOcmAgent = ocmagentv1alpha1.OcmAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ocm-agent",
			},
			Spec:   ocmagentv1alpha1.OcmAgentSpec{},
			Status: ocmagentv1alpha1.OcmAgentStatus{},
		}
		testOcmAgentHandler = ocmAgentHandler{
			Client: mockClient,
			Scheme: testconst.Scheme,
			Log:    testconst.Logger,
			Ctx:    testconst.Context,
		}
	})

	Context("When building an OCM Agent ServiceMonitor", func() {
		It("Sets a correct name", func() {
			sm := buildOCMAgentServiceMonitor(testOcmAgent)
			Expect(sm.Name).To(Equal("ocm-agent-metrics"))
		})
	})

	Context("Managing the OCM Agent ServiceMonitor", func() {
		var testServiceMonitor monitorv1.ServiceMonitor
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespacedName = oah.BuildNamespacedName(oah.OCMAgentServiceMonitorName)
			testServiceMonitor = buildOCMAgentServiceMonitor(testOcmAgent)
		})
		When("the OCM Agent serviceMonitor already exists", func() {
			When("the serviceMonitor differs from what is expected", func() {
				BeforeEach(func() {
					testServiceMonitor.Spec.Endpoints[0].Port = "test-port"
				})
				It("updates the ServiceMonitor", func() {
					goldenSM := buildOCMAgentServiceMonitor(testOcmAgent)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testServiceMonitor),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
							func(ctx context.Context, d *monitorv1.ServiceMonitor, opts ...client.UpdateOptions) error {
								Expect(reflect.DeepEqual(d.Spec, goldenSM.Spec)).To(BeTrue())
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureServiceMonitor(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the ServiceMonitor matches what is expected", func() {
				It("does not update the ServiceMonitor", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testServiceMonitor),
					)
					err := testOcmAgentHandler.ensureServiceMonitor(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})
		When("the OCM Agent ServiceMonitor does not already exist", func() {
			It("creates the ServiceMonitor", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testServiceMonitor.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, d *monitorv1.ServiceMonitor, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Spec, testServiceMonitor.Spec)).To(BeTrue())
							Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
							Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
							Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
							return nil
						}),
				)
				err := testOcmAgentHandler.ensureServiceMonitor(testOcmAgent)
				Expect(err).To(BeNil())
			})
		})
		When("the OCM Agent ServiceMonitor should be removed", func() {
			When("the ServiceMonitor is already removed", func() {
				It("does nothing", func() {
					notFound := k8serrs.NewNotFound(schema.GroupResource{}, testServiceMonitor.Name)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(notFound),
					)
					err := testOcmAgentHandler.ensureServiceMonitorDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the ServiceMonitor exists on the cluster", func() {
				It("removes the ServiceMonitor", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testServiceMonitor),
						mockClient.EXPECT().Delete(gomock.Any(), &testServiceMonitor),
					)
					err := testOcmAgentHandler.ensureServiceMonitorDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})
	})
})
