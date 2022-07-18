package ocmagenthandler

import (
	"context"
	"reflect"

	"github.com/golang/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent Service Handler", func() {
	var (
		mockClient *clientmocks.MockClient
		mockCtrl   *gomock.Controller

		testOcmAgent        ocmagentv1alpha1.OcmAgent
		testOcmAgentHandler ocmAgentHandler
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		testOcmAgent = testconst.TestOCMAgent
		testOcmAgentHandler = ocmAgentHandler{
			Client: mockClient,
			Log:    testconst.Logger,
			Ctx:    testconst.Context,
			Scheme: testconst.Scheme,
		}
	})

	Context("When building an OCM Agent Service", func() {
		It("Sets a correct name", func() {
			cm := buildOCMAgentService(testOcmAgent)
			metricsSvc := buildOCMAgentMetricsService(testOcmAgent)
			Expect(cm.Name).To(Equal("ocm-agent"))
			Expect(metricsSvc.Name).To(Equal("ocm-agent-metrics"))
		})
	})

	Context("Managing the OCM Agent Service", func() {
		var testService, testMetricsService corev1.Service
		var testNamespacedName, testMetricsNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespacedName = oah.BuildNamespacedName(oah.OCMAgentServiceName)
			testService = buildOCMAgentService(testOcmAgent)
			testMetricsService = buildOCMAgentMetricsService(testOcmAgent)
			testMetricsNamespacedName = oah.BuildNamespacedName(oah.OCMAgentMetricsServiceName)
		})
		When("the OCM Agent service already exists", func() {
			When("the service differs from what is expected", func() {
				BeforeEach(func() {
					testService.Spec.Ports[0].Port = int32(9999)
				})
				It("updates the Service", func() {
					goldenService := buildOCMAgentService(testOcmAgent)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testService),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
							func(ctx context.Context, d *corev1.Service, opts ...client.UpdateOptions) error {
								Expect(reflect.DeepEqual(d.Spec, goldenService.Spec)).To(BeTrue())
								return nil
							}),
						mockClient.EXPECT().Get(gomock.Any(), testMetricsNamespacedName, gomock.Any()).Times(1).SetArg(2, testMetricsService),
					)
					err := testOcmAgentHandler.ensureService(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the Service matches what is expected", func() {
				It("does not update the Service", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testService),
						mockClient.EXPECT().Get(gomock.Any(), testMetricsNamespacedName, gomock.Any()).Times(1).SetArg(2, testMetricsService),
					)
					err := testOcmAgentHandler.ensureService(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})
		When("the OCM Agent Service does not already exist", func() {
			It("creates the Service", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testService.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, d *corev1.Service, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Spec, testService.Spec)).To(BeTrue())
							Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
							Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
							Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
							return nil
						}),
					mockClient.EXPECT().Get(gomock.Any(), testMetricsNamespacedName, gomock.Any()).Times(1).SetArg(2, testMetricsService),
				)
				err := testOcmAgentHandler.ensureService(testOcmAgent)
				Expect(err).To(BeNil())
			})
		})
		When("the OCM Agent Service should be removed", func() {
			When("the Service is already removed", func() {
				It("does nothing", func() {
					notFound := k8serrs.NewNotFound(schema.GroupResource{}, testService.Name)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(notFound),
					)
					err := testOcmAgentHandler.ensureServiceDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the Service exists on the cluster", func() {
				It("removes the Service", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testService),
						mockClient.EXPECT().Delete(gomock.Any(), &testService),
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testMetricsService),
						mockClient.EXPECT().Delete(gomock.Any(), &testMetricsService),
					)
					err := testOcmAgentHandler.ensureServiceDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})
	})

	Context("When comparing two services", func() {
		var currentService, expectedService corev1.Service
		BeforeEach(func() {
			currentService = buildOCMAgentService(testOcmAgent)
			expectedService = buildOCMAgentService(testOcmAgent)
		})
		Context("When the labels are different", func() {
			BeforeEach(func() {
				currentService.Labels = map[string]string{"different": "value"}
			})
			It("flags them as different", func() {
				r := serviceConfigChanged(&currentService, &expectedService, testconst.Logger)
				Expect(r).To(BeTrue())
			})
		})
		Context("When the ports are different", func() {
			BeforeEach(func() {
				currentService.Spec.Ports[0].Port = int32(9999)
			})
			It("flags them as different", func() {
				r := serviceConfigChanged(&currentService, &expectedService, testconst.Logger)
				Expect(r).To(BeTrue())
			})
		})
		Context("When there are no differences", func() {
			It("flags that there are none", func() {
				r := serviceConfigChanged(&currentService, &expectedService, testconst.Logger)
				Expect(r).To(BeFalse())
			})
		})
	})
})
