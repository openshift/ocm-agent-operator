package ocmagenthandler

import (
	"context"
	"reflect"

	oconfigv1 "github.com/openshift/api/config/v1"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"
	appsv1 "k8s.io/api/apps/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/golang/mock/gomock"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	"github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent Deployment Handler", func() {
	var (
		mockClient *clientmocks.MockClient
		mockCtrl   *gomock.Controller

		testOcmAgent        ocmagentv1alpha1.OcmAgent
		testOcmAgentHandler ocmAgentHandler
		testHSOcmAgent      ocmagentv1alpha1.OcmAgent
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		testOcmAgent = testconst.TestOCMAgent
		testHSOcmAgent = testconst.TestHSOCMAgent
		testOcmAgentHandler = ocmAgentHandler{
			Client: mockClient,
			Log:    testconst.Logger,
			Ctx:    testconst.Context,
			Scheme: testconst.Scheme,
		}
	})

	Context("When building an OCM Agent Deployment", func() {
		It("deploys with the expected configured values", func() {
			deployment := buildOCMAgentDeployment(testOcmAgent)
			Expect(deployment.Name).To(Equal(testOcmAgent.Name))
			Expect(deployment.Namespace).To(Equal(ocmagenthandler.OCMAgentNamespace))
			Expect(*deployment.Spec.Replicas).To(Equal(testOcmAgent.Spec.Replicas))
			Expect(deployment.Spec.Template.Spec.Containers).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(testOcmAgent.Spec.OcmAgentImage))
			Expect(deployment.Spec.Template.Spec.Volumes).NotTo(BeEmpty())
			// This is a little brittle based on the naming conventions used in the testOcmAgent
			Expect(deployment.Spec.Template.Spec.Volumes[0].Name).To(Equal(testOcmAgent.Spec.TokenSecret))
			Expect(deployment.Spec.Template.Spec.Volumes[1].Name).To(Equal(testOcmAgent.Name + testconst.TestConfigMapSuffix))
			Expect(deployment.Spec.Template.Spec.Volumes[2].Name).To(Equal(ocmagenthandler.TrustedCaBundleConfigMapName))
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name).To(Equal(testOcmAgent.Spec.TokenSecret))
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name).To(Equal(testOcmAgent.Name + testconst.TestConfigMapSuffix))
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts[2].Name).To(Equal(ocmagenthandler.TrustedCaBundleConfigMapName))

			// make sure LivenessProbe is part of deployment config and has defned path, port, url and scheme
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port.IntVal).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Scheme).NotTo(BeEmpty())

			// make sure ReadinessProbe is part of deployment config and has defned path, port url and scheme
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntVal).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Scheme).NotTo(BeEmpty())

			// make sure Resources Limits is part of deployment config and has defined limits for memory and cpu
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().Value()).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().Value()).To(BeNumerically(">", 0))

			// make sure Resources Requests is part of deployment config and has defined limits for memory and cpu
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().Value()).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().Value()).To(BeNumerically(">", 0))

		})
	})

	Context("When building an OCM Agent HS Deployment", func() {
		It("deploys with the expected configured values", func() {
			deployment := buildOCMAgentDeployment(testHSOcmAgent)
			Expect(deployment.Name).To(Equal(testHSOcmAgent.Name))
			Expect(deployment.Namespace).To(Equal(ocmagenthandler.OCMAgentNamespace))
			Expect(*deployment.Spec.Replicas).To(Equal(testHSOcmAgent.Spec.Replicas))
			Expect(deployment.Spec.Template.Spec.Containers).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(testHSOcmAgent.Spec.OcmAgentImage))
			Expect(deployment.Spec.Template.Spec.Volumes).NotTo(BeEmpty())
			// This is a little brittle based on the naming conventions used in the testOcmAgent
			Expect(deployment.Spec.Template.Spec.Volumes[0].Name).To(Equal(testHSOcmAgent.Spec.TokenSecret))
			Expect(deployment.Spec.Template.Spec.Volumes[1].Name).To(Equal(testHSOcmAgent.Name + testconst.TestConfigMapSuffix))
			Expect(deployment.Spec.Template.Spec.Volumes[2].Name).To(Equal(ocmagenthandler.TrustedCaBundleConfigMapName))
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name).To(Equal(testHSOcmAgent.Spec.TokenSecret))
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name).To(Equal(testHSOcmAgent.Name + testconst.TestConfigMapSuffix))
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts[2].Name).To(Equal(ocmagenthandler.TrustedCaBundleConfigMapName))

			// make sure LivenessProbe is part of deployment config and has defned path, port, url and scheme
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port.IntVal).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Scheme).NotTo(BeEmpty())

			// make sure ReadinessProbe is part of deployment config and has defned path, port url and scheme
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntVal).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Scheme).NotTo(BeEmpty())

			// make sure Resources Limits is part of deployment config and has defined limits for memory and cpu
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().Value()).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().Value()).To(BeNumerically(">", 0))

			// make sure Resources Requests is part of deployment config and has defined limits for memory and cpu
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests).NotTo(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().Value()).To(BeNumerically(">", 0))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().Value()).To(BeNumerically(">", 0))
			Expect(func(arr []string, val string) bool {
				for _, item := range arr {
					if item == val {
						return true
					}
				}
				return false
			}(deployment.Spec.Template.Spec.Containers[0].Command, "fleet-mode"))
		})
	})

	Context("Managing the OCM Agent deployment", func() {
		var testDeployment appsv1.Deployment
		var testNamespacedName types.NamespacedName
		var testProxy, testNoProxy oconfigv1.Proxy
		BeforeEach(func() {
			testNamespacedName = ocmagenthandler.BuildNamespacedName(testOcmAgent.Name)
			testDeployment = buildOCMAgentDeployment(testOcmAgent)
			testProxy = oconfigv1.Proxy{
				Status: oconfigv1.ProxyStatus{
					HTTPProxy: "proxy.test:8080",
					NoProxy:   "www.example.com",
				},
			}
			testNoProxy = oconfigv1.Proxy{}
		})

		When("the OCM Agent deployment already exists", func() {
			When("the deployment differs from what is expected", func() {
				BeforeEach(func() {
					replicas := int32(50)
					testDeployment.Spec.Replicas = &replicas
					testDeployment.Spec.Template.Spec.Containers[0].Image = "nope"
				})
				It("updates the deployment", func() {
					goldenDeployment := buildOCMAgentDeployment(testOcmAgent)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testProxy),
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testDeployment),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
							func(ctx context.Context, d *appsv1.Deployment, opts ...client.UpdateOptions) error {
								Expect(d.Spec.Replicas).To(Equal(goldenDeployment.Spec.Replicas))
								Expect(d.Spec.Template.Spec.Containers[0].Image).To(Equal(goldenDeployment.Spec.Template.Spec.Containers[0].Image))
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureDeployment(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the deployment matches what is expected", func() {
				It("does not update the deployment", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testNoProxy),
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testDeployment),
					)
					err := testOcmAgentHandler.ensureDeployment(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})

		When("the OCM Agent deployment does not already exist", func() {
			BeforeEach(func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testProxy),
				)
				envVars, _ := testOcmAgentHandler.buildEnvVars(testOcmAgent)
				testDeployment.Spec.Template.Spec.Containers[0].Env = envVars
			})
			It("creates the deployment", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testDeployment.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testProxy),
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, d *appsv1.Deployment, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Spec, testDeployment.Spec)).To(BeTrue())
							Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
							Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
							Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
							return nil
						}),
				)
				err := testOcmAgentHandler.ensureDeployment(testOcmAgent)
				Expect(err).To(BeNil())
			})
		})

		When("the OCM Agent deployment should be removed", func() {
			When("the deployment is already removed", func() {
				It("does nothing", func() {
					notFound := k8serrs.NewNotFound(schema.GroupResource{}, testDeployment.Name)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(notFound),
					)
					err := testOcmAgentHandler.ensureDeploymentDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the deployment exists on the cluster", func() {
				It("removes the deployment", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testDeployment),
						mockClient.EXPECT().Delete(gomock.Any(), &testDeployment),
					)
					err := testOcmAgentHandler.ensureDeploymentDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})

		When("checking if the OCM Agent deployment has been changed", func() {
			var goldenDeployment appsv1.Deployment
			BeforeEach(func() {
				goldenDeployment = buildOCMAgentDeployment(testOcmAgent)
			})
			It("should detect a label change", func() {
				testDeployment.Labels = map[string]string{"dummy": "value"}
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should detect an image change", func() {
				testDeployment.Spec.Template.Spec.Containers[0].Image = "something else"
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should handle missing readiness probe", func() {
				testDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should handle missing liveness probe", func() {
				testDeployment.Spec.Template.Spec.Containers[0].LivenessProbe = nil
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should detect a readiness probe change", func() {
				testDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet = nil
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should detect a liveness probe change", func() {
				testDeployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet = nil
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should detect a replica change", func() {
				replicas := int32(5000)
				testDeployment.Spec.Replicas = &replicas
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("should detect a affinity change", func() {
				testDeployment.Spec.Template.Spec.Affinity = nil
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeTrue())
			})
			It("not detect a change if there are no differences", func() {
				changed := deploymentConfigChanged(&testDeployment, &goldenDeployment, testOcmAgent, testconst.Logger)
				Expect(changed).To(BeFalse())
			})
		})
	})
})
