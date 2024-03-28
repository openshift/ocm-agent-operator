package ocmagenthandler

import (
	"context"
	"reflect"

	"go.uber.org/mock/gomock"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"
)

var _ = Describe("OCM Agent NetworkPolicy Handler", func() {
	var (
		mockClient *clientmocks.MockClient
		mockCtrl   *gomock.Controller

		testOcmAgent        ocmagentv1alpha1.OcmAgent
		testFleetOcmAgent   ocmagentv1alpha1.OcmAgent
		testOcmAgentHandler ocmAgentHandler
		testNamespace       string
		networkPolicy       netv1.NetworkPolicy
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		testOcmAgent = testconst.TestOCMAgent
		testFleetOcmAgent = testconst.TestHSOCMAgent
		testOcmAgentHandler = ocmAgentHandler{
			Client: mockClient,
			Log:    testconst.Logger,
			Ctx:    testconst.Context,
			Scheme: testconst.Scheme,
		}
	})

	Context("When building an OCM Agent NetworkPolicy", func() {
		BeforeEach(func() {
			testNamespace = oah.NamespaceMonitorng
			networkPolicy = buildNetworkPolicy(testOcmAgent, testNamespace)
		})

		It("Should have the expected name, namespace and labels", func() {
			Expect(networkPolicy.Name).To(ContainSubstring(oah.OCMAgentDefaultNetworkPolicySuffix))
			Expect(networkPolicy.Namespace).To(Equal(oah.OCMAgentNamespace))
			Expect(networkPolicy.Labels["app"]).To(Equal(testOcmAgent.Name))
		})

		It("Should include an ingress rule to allow traffic from the specified namespace", func() {
			Expect(len(networkPolicy.Spec.Ingress)).To(Equal(1))
			Expect(networkPolicy.Spec.Ingress[0].From).To(HaveLen(1))

			nsSelector := networkPolicy.Spec.Ingress[0].From[0].NamespaceSelector
			Expect(nsSelector).NotTo(BeNil())
			Expect(nsSelector.MatchLabels).To(HaveKeyWithValue("kubernetes.io/metadata.name", testNamespace))
		})

		It("Should apply to pods with the correct app label", func() {
			Expect(networkPolicy.Spec.PodSelector.MatchLabels).To(HaveKeyWithValue("app", testOcmAgent.Name))
		})
	})

	Context("Managing the OCM Agent NetworkPolicy", func() {
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespace = oah.NamespaceOBO
			testNamespacedName = buildNetworkPolicyName(testOcmAgent, testNamespace)
			networkPolicy = buildNetworkPolicy(testOcmAgent, testNamespace)
		})
		When("the network policy already exists", func() {
			When("the network policy differs from what is expected", func() {
				BeforeEach(func() {
					networkPolicy.Spec.PodSelector.MatchLabels = map[string]string{"fake": "fake"}
				})
				It("updates the networkpolicy", func() {
					goldenNetworkPolicy := buildNetworkPolicy(testOcmAgent, testNamespace)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, networkPolicy),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, d *netv1.NetworkPolicy, opts ...client.UpdateOptions) error {
								Expect(reflect.DeepEqual(d.Spec, goldenNetworkPolicy.Spec)).To(BeTrue())
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureNetworkPolicy(testOcmAgent, testNamespace)
					Expect(err).To(BeNil())
				})
			})
			When("the networkpolicy matches what is expected", func() {
				It("does not update the networkpolicy", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, networkPolicy),
					)
					err := testOcmAgentHandler.ensureNetworkPolicy(testOcmAgent, testNamespace)
					Expect(err).To(BeNil())
				})
			})
		})

		When("the OCM Agent networkpolicy does not already exist", func() {
			It("creates the networkpolicy", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, networkPolicy.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, d *netv1.NetworkPolicy, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Spec, networkPolicy.Spec)).To(BeTrue())
							Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
							Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
							Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
							return nil
						}),
				)
				err := testOcmAgentHandler.ensureNetworkPolicy(testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})
		})
	})

	Context("Deleting the ocm agent networkpolicies", func() {
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespace = oah.NamespaceMUO
			testNamespacedName = buildNetworkPolicyName(testOcmAgent, testNamespace)
		})
		When("network policy exists", func() {
			It("should be able to delete the networkpolicy", func() {
				networkPolicy = buildNetworkPolicy(testOcmAgent, testNamespace)
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, networkPolicy)
				mockClient.EXPECT().Delete(gomock.Any(), gomock.Any())
				err := testOcmAgentHandler.ensureNetworkPolicyDeleted(testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})
		})
		When("network policy does not exist", func() {
			It("should skip the deletion", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, networkPolicy.Name)
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound)
				err := testOcmAgentHandler.ensureNetworkPolicyDeleted(testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})
		})
	})

	Context("ensure all the required networkpolicies created", func() {
		When("creating a non-fleet ocm-agent", func() {
			It("should have the 2 networkpolicies created", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(2)
				err := testOcmAgentHandler.ensureAllNetworkPolicies(testOcmAgent)
				Expect(err).To(BeNil())
			})
		})
		When("creating a fleet ocm-agent", func() {
			It("should have the 3 networkpolicies created", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
				err := testOcmAgentHandler.ensureAllNetworkPolicies(testFleetOcmAgent)
				Expect(err).To(BeNil())
			})
		})
	})
})
