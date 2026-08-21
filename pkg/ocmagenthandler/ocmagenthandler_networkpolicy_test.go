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
			Scheme: testconst.Scheme,
		}
	})

	Context("When building an OCM Agent NetworkPolicy", func() {
		BeforeEach(func() {
			testNamespace = oah.NamespaceMonitorng
			var err error
			networkPolicy, err = buildNetworkPolicy(testOcmAgent, testNamespace)
			Expect(err).To(BeNil())
		})

		It("Should have the expected name, namespace and labels", func() {
			Expect(networkPolicy.Name).To(ContainSubstring(oah.OCMAgentDefaultNetworkPolicySuffix))
			Expect(networkPolicy.Namespace).To(Equal(oah.OCMAgentNamespace))
			Expect(networkPolicy.Labels["app"]).To(Equal(testOcmAgent.Name))
		})

		It("Should include an ingress rule to allow traffic from the specified namespace", func() {
			Expect(len(networkPolicy.Spec.Ingress)).To(Equal(1), "monitoring namespace ingress contract: expected exactly one ingress rule")
			Expect(networkPolicy.Spec.Ingress[0].From).To(HaveLen(1), "monitoring namespace ingress contract: expected exactly one From peer")

			nsSelector := networkPolicy.Spec.Ingress[0].From[0].NamespaceSelector
			Expect(nsSelector).NotTo(BeNil(), "monitoring namespace ingress contract: NamespaceSelector must be set")
			Expect(nsSelector.MatchLabels).To(HaveKeyWithValue("kubernetes.io/metadata.name", testNamespace), "monitoring namespace ingress contract: NamespaceSelector must match namespace %q", testNamespace)
		})

		It("Should restrict ingress to alertmanager pods in the monitoring namespace", func() {
			podSelector := networkPolicy.Spec.Ingress[0].From[0].PodSelector
			Expect(podSelector).NotTo(BeNil(), "monitoring namespace ingress contract: PodSelector must be set")
			Expect(podSelector.MatchLabels).To(HaveKeyWithValue(oah.AlertmanagerPodLabelKey, oah.AlertmanagerPodLabelValue), "monitoring namespace ingress contract: PodSelector must match alertmanager label %s=%s", oah.AlertmanagerPodLabelKey, oah.AlertmanagerPodLabelValue)
		})

		It("Should apply to pods with the correct app label", func() {
			Expect(networkPolicy.Spec.PodSelector.MatchLabels).To(HaveKeyWithValue("app", testOcmAgent.Name))
		})

		Context("for the MUO namespace", func() {
			BeforeEach(func() {
				testNamespace = oah.NamespaceMUO
				var err error
				networkPolicy, err = buildNetworkPolicy(testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})

			It("Should restrict ingress to MUO pods only", func() {
				podSelector := networkPolicy.Spec.Ingress[0].From[0].PodSelector
				Expect(podSelector).NotTo(BeNil())
				Expect(podSelector.MatchLabels).To(HaveKeyWithValue(oah.MUOPodLabelKey, oah.MUOPodLabelValue))
			})
		})

		Context("for the RHOBS namespace (fleet mode)", func() {
			BeforeEach(func() {
				testNamespace = oah.NamespaceRHOBS
				var err error
				networkPolicy, err = buildNetworkPolicy(testFleetOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})

			It("Should restrict ingress to RHOBS alertmanager pods only", func() {
				podSelector := networkPolicy.Spec.Ingress[0].From[0].PodSelector
				Expect(podSelector).NotTo(BeNil())
				Expect(podSelector.MatchLabels).To(HaveKeyWithValue(oah.RHOBSPodLabelKey, oah.RHOBSPodLabelValue))
			})

			It("Should scope ingress to the OBO namespace, since RHOBS has no namespace of its own", func() {
				nsSelector := networkPolicy.Spec.Ingress[0].From[0].NamespaceSelector
				Expect(nsSelector).NotTo(BeNil())
				Expect(nsSelector.MatchLabels).To(HaveKeyWithValue("kubernetes.io/metadata.name", oah.NamespaceOBO))
			})
		})

		Context("for the OBO namespace (fleet mode)", func() {
			BeforeEach(func() {
				testNamespace = oah.NamespaceOBO
				var err error
				networkPolicy, err = buildNetworkPolicy(testFleetOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})

			It("Should restrict ingress to OBO pods only", func() {
				podSelector := networkPolicy.Spec.Ingress[0].From[0].PodSelector
				Expect(podSelector).NotTo(BeNil())
				Expect(podSelector.MatchLabels).To(HaveKeyWithValue(oah.OBOPodLabelKey, oah.OBOPodLabelValue))
			})
		})

		Context("for an unrecognized namespace", func() {
			It("returns an error instead of silently falling back to a namespace-wide policy", func() {
				_, err := buildNetworkPolicy(testOcmAgent, "some-other-namespace")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Managing the OCM Agent NetworkPolicy", func() {
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespace = oah.NamespaceOBO
			testNamespacedName = buildNetworkPolicyName(testOcmAgent, testNamespace)
			var err error
			networkPolicy, err = buildNetworkPolicy(testOcmAgent, testNamespace)
			Expect(err).To(BeNil())
		})
		When("the network policy already exists", func() {
			When("the network policy differs from what is expected", func() {
				BeforeEach(func() {
					networkPolicy.Spec.PodSelector.MatchLabels = map[string]string{"fake": "fake"}
				})
				It("updates the networkpolicy", func() {
					goldenNetworkPolicy, err := buildNetworkPolicy(testOcmAgent, testNamespace)
					Expect(err).To(BeNil())
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, networkPolicy),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, d *netv1.NetworkPolicy, opts ...client.UpdateOptions) error {
								Expect(reflect.DeepEqual(d.Spec, goldenNetworkPolicy.Spec)).To(BeTrue())
								return nil
							}),
					)
					err = testOcmAgentHandler.ensureNetworkPolicy(testconst.Context, testOcmAgent, testNamespace)
					Expect(err).To(BeNil())
				})
			})
			When("the networkpolicy matches what is expected", func() {
				It("does not update the networkpolicy", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, networkPolicy),
					)
					err := testOcmAgentHandler.ensureNetworkPolicy(testconst.Context, testOcmAgent, testNamespace)
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
				err := testOcmAgentHandler.ensureNetworkPolicy(testconst.Context, testOcmAgent, testNamespace)
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
				var err error
				networkPolicy, err = buildNetworkPolicy(testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, networkPolicy)
				mockClient.EXPECT().Delete(gomock.Any(), gomock.Any())
				err = testOcmAgentHandler.ensureNetworkPolicyDeleted(testconst.Context, testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})
		})
		When("network policy does not exist", func() {
			It("should skip the deletion", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, networkPolicy.Name)
				mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound)
				err := testOcmAgentHandler.ensureNetworkPolicyDeleted(testconst.Context, testOcmAgent, testNamespace)
				Expect(err).To(BeNil())
			})
		})
	})

	Context("ensure all the required networkpolicies created", func() {
		When("creating a non-fleet ocm-agent", func() {
			It("should have the 2 networkpolicies created", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(2)
				err := testOcmAgentHandler.ensureAllNetworkPolicies(testconst.Context, testOcmAgent)
				Expect(err).To(BeNil())
			})
		})
		When("creating a fleet ocm-agent", func() {
			It("should have the 3 networkpolicies created", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
				err := testOcmAgentHandler.ensureAllNetworkPolicies(testconst.Context, testFleetOcmAgent)
				Expect(err).To(BeNil())
			})
		})
	})
})
