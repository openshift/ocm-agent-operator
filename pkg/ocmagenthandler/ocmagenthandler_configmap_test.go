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
	oahconst "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent ConfigMap Handler", func() {
	var (
		mockClient *clientmocks.MockClient
		mockCtrl   *gomock.Controller

		testOcmAgent        ocmagentv1alpha1.OcmAgent
		testOcmAgentHandler ocmAgentHandler
		testClusterId       string
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
		testClusterId = "9345c78b-b6b6-4f42-b242-79bfcc403b0a"
	})

	Context("When building an OCM Agent ConfigMap", func() {
		var cm *corev1.ConfigMap
		BeforeEach(func() {
			cm = buildOCMAgentConfigMap(testOcmAgent, testClusterId)
		})
		It("Sets a correct name", func() {
			Expect(cm.Name).To(Equal(testOcmAgent.Spec.OcmAgentConfig))
		})
		It("Sets the correct data", func() {
			Expect(cm.Data).To(HaveKeyWithValue(oahconst.OCMAgentConfigClusterID, testClusterId))
			Expect(cm.Data).To(HaveKeyWithValue(oahconst.OCMAgentConfigURLKey, testOcmAgent.Spec.AgentConfig.OcmBaseUrl))
			Expect(cm.Data).To(HaveKey(oahconst.OCMAgentConfigServicesKey))
		})
	})

	Context("Managing the OCM Agent ConfigMap", func() {
		var testConfigMap *corev1.ConfigMap
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespacedName = oahconst.BuildNamespacedName(testOcmAgent.Spec.OcmAgentConfig)
			testConfigMap = buildOCMAgentConfigMap(testOcmAgent, testClusterId)
		})
		When("the OCM Agent config already exists", func() {
			When("the config differs from what is expected", func() {
				BeforeEach(func() {
					testConfigMap.Data = map[string]string{"fake": "fake"}
				})
				It("updates the configmap", func() {
					goldenConfig := buildOCMAgentConfigMap(testOcmAgent, testClusterId)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, *testConfigMap),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, d *corev1.ConfigMap, opts ...client.UpdateOptions) error {
								Expect(d.Data).To(Equal(goldenConfig.Data))
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, true)
					Expect(err).To(BeNil())
				})
			})
			When("the configmap matches what is expected", func() {
				It("does not update the configmap", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, *testConfigMap),
					)
					err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, true)
					Expect(err).To(BeNil())
				})
			})

		})
		When("the OCM Agent configmap does not already exist", func() {
			It("creates the configmap", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testConfigMap.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, d *corev1.ConfigMap, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Data, testConfigMap.Data)).To(BeTrue())
							return nil
						}),
				)
				err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, true)
				Expect(err).To(BeNil())
			})
		})
		When("the OCM Agent configmap should be removed", func() {
			When("the configmap is already removed", func() {
				It("does nothing", func() {
					notFound := k8serrs.NewNotFound(schema.GroupResource{}, testConfigMap.Name)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound),
					)
					err := testOcmAgentHandler.ensureConfigMapDeleted(testNamespacedName)
					Expect(err).To(BeNil())
				})
			})
			When("the configmap exists on the cluster", func() {
				It("removes the configmap", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testConfigMap),
						mockClient.EXPECT().Delete(gomock.Any(), testConfigMap),
					)
					err := testOcmAgentHandler.ensureConfigMapDeleted(testNamespacedName)
					Expect(err).To(BeNil())
				})
			})
		})
	})

	Context("Managing the CAMO ConfigMap", func() {
		When("building the CAMO configmap", func() {
			var cm *corev1.ConfigMap
			var err error
			BeforeEach(func() {
				cm, err = buildCAMOConfigMap()
			})
			It("builds one successfully", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(cm.Name).To(Equal(oahconst.CAMOConfigMapNamespacedName.Name))
				Expect(cm.Namespace).To(Equal(oahconst.CAMOConfigMapNamespacedName.Namespace))
				Expect(cm.Data).To(HaveKey(oahconst.OCMAgentServiceURLKey))
			})
		})
	})

	Context("Managing the Trusted CA configmap", func() {
		var testcm *corev1.ConfigMap
		var testNamespacedName types.NamespacedName
		When("building the Trusted CA configmap", func() {
			BeforeEach(func() {
				testcm = buildTrustedCaConfigMap()
			})
			It("builds successfully", func() {
				Expect(testcm.Name).To(Equal("trusted-ca-bundle"))
				Expect(testcm.Namespace).To(Equal(oahconst.OCMAgentNamespace))
				Expect(testcm.ObjectMeta.Labels).Should(HaveKey(oahconst.InjectCaBundleIndicator))
			})
		})
		When("the trusted ca bundle being updated", func() {
			BeforeEach(func() {
				testcm = buildTrustedCaConfigMap()
				testcm.Data = map[string]string{"aaa": "bbb"}
				testNamespacedName = oahconst.BuildNamespacedName(testcm.Name)
			})
			It("does not update the configmap", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, *testcm),
				)
				err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testcm, true)
				Expect(err).To(BeNil())
			})
		})
	})

	Context("When applying a controller reference", func() {
		var testConfigMap *corev1.ConfigMap
		var testNamespacedName types.NamespacedName
		var notFound *k8serrs.StatusError

		BeforeEach(func() {
			testNamespacedName = oahconst.BuildNamespacedName(testOcmAgent.Spec.OcmAgentConfig)
			testConfigMap = buildOCMAgentConfigMap(testOcmAgent, testClusterId)
			notFound = k8serrs.NewNotFound(schema.GroupResource{}, testConfigMap.Name)
		})
		It("Adds one if requested", func() {
			gomock.InOrder(
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound),
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, d *corev1.ConfigMap, opts ...client.CreateOptions) error {
						Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
						Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
						Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
						return nil
					}),
			)
			err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, true)
			Expect(err).To(BeNil())
		})

		It("Does not add one if not requested", func() {
			gomock.InOrder(
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound),
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, d *corev1.ConfigMap, opts ...client.CreateOptions) error {
						Expect(d.ObjectMeta.OwnerReferences).To(BeNil())
						return nil
					}),
			)
			err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, false)
			Expect(err).To(BeNil())
		})

	})
})
