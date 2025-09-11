package ocmagenthandler

import (
	"context"
	"errors"
	"reflect"

	"go.uber.org/mock/gomock"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	Context("When building an OCM Agent ConfigMap ", func() {
		It("Sets correct name and data with cluster ID", func() {
			cm := buildOCMAgentConfigMap(testOcmAgent, testClusterId)
			Expect(cm.Name).To(Equal(testOcmAgent.Name + testconst.TestConfigMapSuffix))
			Expect(cm.Data).To(HaveKeyWithValue(oahconst.OCMAgentConfigClusterID, testClusterId))
			Expect(cm.Data).To(HaveKeyWithValue(oahconst.OCMAgentConfigURLKey, testOcmAgent.Spec.AgentConfig.OcmBaseUrl))
			Expect(cm.Data).To(HaveKey(oahconst.OCMAgentConfigServicesKey))
		})
		It("Should not include cluster ID when empty", func() {
			cm := buildOCMAgentConfigMap(testOcmAgent, "")
			Expect(cm.Data).ToNot(HaveKey(oahconst.OCMAgentConfigClusterID))
			Expect(cm.Data).To(HaveKeyWithValue(oahconst.OCMAgentConfigURLKey, testOcmAgent.Spec.AgentConfig.OcmBaseUrl))
		})
	})

	Context("Managing the OCM Agent ConfigMap", func() {
		var testConfigMap *corev1.ConfigMap
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespacedName = oahconst.BuildNamespacedName(testOcmAgent.Name)
			testNamespacedName.Name = testNamespacedName.Name + testconst.TestConfigMapSuffix
			testConfigMap = buildOCMAgentConfigMap(testOcmAgent, testClusterId)
		})
		When("the OCM Agent config already exists", func() {
			When("the config differs from what is expected", func() {
				BeforeEach(func() {
					testConfigMap.Data = map[string]string{"fake": "fake"}
				})
				It("updates the configmap and handles update failures", func() {
					goldenConfig := buildOCMAgentConfigMap(testOcmAgent, testClusterId)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, *testConfigMap),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, d *corev1.ConfigMap, opts ...client.UpdateOptions) error {
								Expect(d.Data).To(Equal(goldenConfig.Data))
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, goldenConfig, true)
					Expect(err).To(BeNil())

					// Test update failure
					testError := errors.New("update failed")
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, *testConfigMap),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(testError),
					)
					err = testOcmAgentHandler.ensureConfigMap(testOcmAgent, goldenConfig, true)
					Expect(err).To(Equal(testError))
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
			It("creates the configmap and handles create/get failures", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testConfigMap.Name)

				// Test successful creation
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

				// Test create failure
				createError := errors.New("create failed")
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(createError),
				)
				err = testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, true)
				Expect(err).To(Equal(createError))

				// Test unexpected get error
				getError := errors.New("get failed")
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(getError)
				err = testOcmAgentHandler.ensureConfigMap(testOcmAgent, testConfigMap, true)
				Expect(err).To(Equal(getError))
			})
		})
		When("removing configmaps", func() {
			It("handles deletion scenarios and errors", func() {
				// Test: configmap already removed
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testConfigMap.Name)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound)
				err := testOcmAgentHandler.ensureConfigMapDeleted(testNamespacedName)
				Expect(err).To(BeNil())

				// Test: successful deletion
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testConfigMap),
					mockClient.EXPECT().Delete(gomock.Any(), testConfigMap).Return(nil),
				)
				err = testOcmAgentHandler.ensureConfigMapDeleted(testNamespacedName)
				Expect(err).To(BeNil())

				// Test: delete failure
				deleteError := errors.New("delete failed")
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testConfigMap),
					mockClient.EXPECT().Delete(gomock.Any(), testConfigMap).Return(deleteError),
				)
				err = testOcmAgentHandler.ensureConfigMapDeleted(testNamespacedName)
				Expect(err).To(Equal(deleteError))

				// Test: get error during deletion
				getError := errors.New("get failed during delete")
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(getError)
				err = testOcmAgentHandler.ensureConfigMapDeleted(testNamespacedName)
				Expect(err).To(Equal(getError))
			})
		})
	})

	Context("Managing the CAMO ConfigMap", func() {
		It("builds successfully and handles URL errors", func() {
			// Test successful build
			cm, err := buildCAMOConfigMap(testOcmAgent)
			Expect(err).ToNot(HaveOccurred())
			Expect(cm.Name).To(Equal(oahconst.CAMOConfigMapNamespacedName.Name))
			Expect(cm.Namespace).To(Equal(oahconst.CAMOConfigMapNamespacedName.Namespace))
			Expect(cm.Data).To(HaveKey(oahconst.OCMAgentServiceURLKey))

			// Test URL build failure
			invalidAgent := testOcmAgent
			invalidAgent.Name = "invalid name with spaces"
			invalidAgent.Namespace = "invalid\nnamespace"
			cm, err = buildCAMOConfigMap(invalidAgent)
			Expect(err).To(HaveOccurred())
			Expect(cm).To(BeNil())
		})
	})

	Context("Managing the Trusted CA configmap", func() {
		It("builds successfully and skips updates", func() {
			testcm := buildTrustedCaConfigMap()
			Expect(testcm.Name).To(Equal("trusted-ca-bundle"))
			Expect(testcm.Namespace).To(Equal(oahconst.OCMAgentNamespace))
			Expect(testcm.ObjectMeta.Labels).Should(HaveKey(oahconst.InjectCaBundleIndicator))

			// Test that trusted CA bundle updates are skipped
			testcm.Data = map[string]string{"aaa": "bbb"}
			testNamespacedName := oahconst.BuildNamespacedName(testcm.Name)
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, *testcm)
			err := testOcmAgentHandler.ensureConfigMap(testOcmAgent, testcm, true)
			Expect(err).To(BeNil())
		})
	})

	Context("When applying a controller reference", func() {
		var testConfigMap *corev1.ConfigMap
		var testNamespacedName types.NamespacedName
		var notFound *k8serrs.StatusError

		BeforeEach(func() {
			testNamespacedName = oahconst.BuildNamespacedName(testOcmAgent.Name)
			testNamespacedName.Name = testNamespacedName.Name + testconst.TestConfigMapSuffix
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

	Context("Testing remaining functions", func() {
		It("fetchClusterVersion handles success and failure", func() {
			testClusterVersion := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Spec:       configv1.ClusterVersionSpec{ClusterID: "test-cluster-id"},
			}

			// Test successful fetch
			mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "version"}, gomock.Any()).
				SetArg(2, *testClusterVersion).Return(nil)
			clusterVersion, err := testOcmAgentHandler.fetchClusterVersion()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(clusterVersion.Spec.ClusterID)).To(Equal("test-cluster-id"))

			// Test fetch failure
			testError := errors.New("cluster version not found")
			mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "version"}, gomock.Any()).Return(testError)
			clusterVersion, err = testOcmAgentHandler.fetchClusterVersion()
			Expect(err).To(Equal(testError))
			Expect(clusterVersion).To(BeNil())
		})

		It("ensureAllConfigMaps handles fleet mode and errors", func() {
			testClusterVersion := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Spec:       configv1.ClusterVersionSpec{ClusterID: "test-cluster-id"},
			}

			// Test fleet mode (no CAMO, no cluster ID)
			testOcmAgent.Spec.FleetMode = true
			mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "version"}, gomock.Any()).SetArg(2, *testClusterVersion)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(k8serrs.NewNotFound(schema.GroupResource{}, "")).Times(2)
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(2)
			err := testOcmAgentHandler.ensureAllConfigMaps(testOcmAgent)
			Expect(err).ToNot(HaveOccurred())

			// Test cluster version fetch error
			fetchError := errors.New("fetch failed")
			mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "version"}, gomock.Any()).Return(fetchError)
			err = testOcmAgentHandler.ensureAllConfigMaps(testOcmAgent)
			Expect(err).To(Equal(fetchError))
		})

		It("ensureAllConfigMapsDeleted handles deletion scenarios", func() {
			// Test successful deletion
			testCM := &corev1.ConfigMap{}
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testCM).Times(2)
			mockClient.EXPECT().Delete(gomock.Any(), testCM).Return(nil).Times(2)
			err := testOcmAgentHandler.ensureAllConfigMapsDeleted(testOcmAgent)
			Expect(err).ToNot(HaveOccurred())

			// Test deletion error
			deleteError := errors.New("delete failed")
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(deleteError)
			err = testOcmAgentHandler.ensureAllConfigMapsDeleted(testOcmAgent)
			Expect(err).To(Equal(deleteError))
		})
	})
})
