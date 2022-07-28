package ocmagenthandler

import (
	"context"
	"reflect"

	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/golang/mock/gomock"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent NetworkPolicy Handler", func() {
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

	Context("When building an OCM Agent NetworkPolicy", func() {
		var np netv1.NetworkPolicy
		BeforeEach(func() {
			np = buildNetworkPolicy()
		})
		It("Has the expected name and namespace", func() {
			Expect(np.Name).To(Equal(oah.OCMAgentNetworkPolicyName))
			Expect(np.Namespace).To(Equal(oah.OCMAgentNamespace))
		})
	})

	Context("Managing the OCM Agent NetworkPolicy", func() {
		var testNetworkPolicy netv1.NetworkPolicy
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNetworkPolicy = buildNetworkPolicy()
			testNamespacedName = types.NamespacedName{
				Namespace: testNetworkPolicy.Namespace,
				Name:      testNetworkPolicy.Name,
			}
		})
		When("the network policy already exists", func() {
			When("the network policy differs from what is expected", func() {
				BeforeEach(func() {
					testNetworkPolicy.Spec.PodSelector.MatchLabels = map[string]string{"fake": "fake"}
				})
				It("updates the networkpolicy", func() {
					goldenNetworkPolicy := buildNetworkPolicy()
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, testNetworkPolicy),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, d *netv1.NetworkPolicy, opts ...client.UpdateOptions) error {
								Expect(reflect.DeepEqual(d.Spec, goldenNetworkPolicy.Spec)).To(BeTrue())
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureNetworkPolicy(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the networkpolicy matches what is expected", func() {
				It("does not update the networkpolicy", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, testNetworkPolicy),
					)
					err := testOcmAgentHandler.ensureNetworkPolicy(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})

		When("the OCM Agent networkpolicy does not already exist", func() {
			It("creates the networkpolicy", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testNetworkPolicy.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, d *netv1.NetworkPolicy, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Spec, testNetworkPolicy.Spec)).To(BeTrue())
							Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
							Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
							Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
							return nil
						}),
				)
				err := testOcmAgentHandler.ensureNetworkPolicy(testOcmAgent)
				Expect(err).To(BeNil())
			})
		})
	})
})
