package ocmagenthandler

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oahconst "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	"github.com/openshift/ocm-agent-operator/pkg/localmetrics"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"

	"github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent Access Token Secret Handler", func() {

	const (
		testAccessTokenValue = "dGhpcyBpcyBhIHRlc3QgdmFsdWU=" //#nosec G101 -- This is a false positive (test value)
	)

	var (
		mockClient *clientmocks.MockClient
		mockCtrl   *gomock.Controller

		testOcmAgent                  ocmagentv1alpha1.OcmAgent
		testOcmAgentHandler           ocmAgentHandler
		testOcmAccessTokenSecretValue []byte
		testClusterPullSecretValue    []byte
		testPullSecret                corev1.Secret
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clientmocks.NewMockClient(mockCtrl)
		testOcmAgent = testconst.TestOCMAgent
		testOcmAgentHandler = ocmAgentHandler{
			Client: mockClient,
			Scheme: testconst.Scheme,
			Log:    testconst.Logger,
			Ctx:    testconst.Context,
		}
		testClusterPullSecretValue = []byte(fmt.Sprintf(`{
			"auths": {
				"cloud.openshift.com": {
					"auth": "%s",
					"email": "testuser@example.com"
				}
			}
		}`, testAccessTokenValue))
		testOcmAccessTokenSecretValue = []byte(testAccessTokenValue)
		pullSecretNamespacedName := oahconst.PullSecretNamespacedName
		testPullSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretNamespacedName.Name,
				Namespace: pullSecretNamespacedName.Namespace,
			},
			Data: map[string][]byte{
				oahconst.PullSecretKey: testClusterPullSecretValue,
			},
		}
	})

	Context("When building an OCM Agent Access Token Secret", func() {
		It("Sets a correct name", func() {
			cm := buildOCMAgentAccessTokenSecret(testOcmAccessTokenSecretValue, testOcmAgent)
			Expect(cm.Data).Should(HaveKey(oahconst.OCMAgentAccessTokenSecretKey))
			Expect(bytes.Compare(cm.Data[oahconst.OCMAgentAccessTokenSecretKey], testOcmAccessTokenSecretValue)).To(BeZero())
			Expect(cm.Name).To(Equal(testOcmAgent.Spec.TokenSecret))
		})
	})

	Context("Managing the OCM Agent Secret", func() {
		var testSecret corev1.Secret
		var testNamespacedName types.NamespacedName
		BeforeEach(func() {
			testNamespacedName = oahconst.BuildNamespacedName(testOcmAgent.Spec.TokenSecret)
			testSecret = buildOCMAgentAccessTokenSecret(testOcmAccessTokenSecretValue, testOcmAgent)
		})
		When("the OCM Agent secret already exists", func() {
			When("the secret differs from what is expected", func() {
				BeforeEach(func() {
					testSecret.Data = map[string][]byte{"fake": []byte("not the right value")}
				})
				It("updates the secret", func() {
					goldenSecret := buildOCMAgentAccessTokenSecret(testOcmAccessTokenSecretValue, testOcmAgent)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), oahconst.PullSecretNamespacedName, gomock.Any()).Times(1).SetArg(2, testPullSecret),
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testSecret),
						mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
							func(ctx context.Context, d *corev1.Secret, opts ...client.UpdateOptions) error {
								Expect(d.Data).Should(HaveKey(oahconst.OCMAgentAccessTokenSecretKey))
								Expect(bytes.Compare(d.Data[oahconst.OCMAgentAccessTokenSecretKey], goldenSecret.Data[oahconst.OCMAgentAccessTokenSecretKey])).To(BeZero())
								return nil
							}),
					)
					err := testOcmAgentHandler.ensureAccessTokenSecret(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the secret matches what is expected", func() {
				It("does not update the secret", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), oahconst.PullSecretNamespacedName, gomock.Any()).Times(1).SetArg(2, testPullSecret),
						mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testSecret),
					)
					err := testOcmAgentHandler.ensureAccessTokenSecret(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})
		When("the OCM Agent secret does not already exist", func() {
			It("creates the secret", func() {
				notFound := k8serrs.NewNotFound(schema.GroupResource{}, testSecret.Name)
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), oahconst.PullSecretNamespacedName, gomock.Any()).Times(1).SetArg(2, testPullSecret),
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).Return(notFound),
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, d *corev1.Secret, opts ...client.CreateOptions) error {
							Expect(reflect.DeepEqual(d.Data, testSecret.Data)).To(BeTrue())
							Expect(d.ObjectMeta.OwnerReferences[0].Kind).To(Equal("OcmAgent"))
							Expect(*d.ObjectMeta.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
							Expect(*d.ObjectMeta.OwnerReferences[0].Controller).To(BeTrue())
							return nil
						}),
				)
				err := testOcmAgentHandler.ensureAccessTokenSecret(testOcmAgent)
				Expect(err).To(BeNil())
			})
		})
		When("the access token secret should be removed", func() {
			When("the secret is already removed", func() {
				It("does nothing", func() {
					notFound := k8serrs.NewNotFound(schema.GroupResource{}, testSecret.Name)
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(notFound),
					)
					err := testOcmAgentHandler.ensureAccessTokenSecretDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
			When("the configmap exists on the cluster", func() {
				It("removes the configmap", func() {
					gomock.InOrder(
						mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testSecret),
						mockClient.EXPECT().Delete(gomock.Any(), &testSecret),
					)
					err := testOcmAgentHandler.ensureAccessTokenSecretDeleted(testOcmAgent)
					Expect(err).To(BeNil())
				})
			})
		})
		When("the pull secret can't be found", func() {
			BeforeEach(func() {
				delete(testPullSecret.Data, oahconst.PullSecretKey)
			})
			It("returns an error and sets the metric", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).SetArg(2, testSecret),
				)
				err := testOcmAgentHandler.ensureAccessTokenSecret(testOcmAgent)
				Expect(err).NotTo(BeNil())
				expectedMetric := `
# HELP ocm_agent_operator_pull_secret_invalid Failed to obtain a valid pull secret
# TYPE ocm_agent_operator_pull_secret_invalid gauge
ocm_agent_operator_pull_secret_invalid{ocmagent_name="test-ocm-agent"} 1
`
				err = testutil.CollectAndCompare(localmetrics.MetricPullSecretInvalid, strings.NewReader(expectedMetric))
				Expect(err).To(BeNil())
			})
		})
		When("the pull secret can be found", func() {
			It("sets the correct metric", func() {
				gomock.InOrder(
					mockClient.EXPECT().Get(gomock.Any(), oahconst.PullSecretNamespacedName, gomock.Any()).Times(1).SetArg(2, testPullSecret),
					mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Times(1).SetArg(2, testSecret),
				)
				err := testOcmAgentHandler.ensureAccessTokenSecret(testOcmAgent)
				Expect(err).To(BeNil())
				expectedMetric := `
# HELP ocm_agent_operator_pull_secret_invalid Failed to obtain a valid pull secret
# TYPE ocm_agent_operator_pull_secret_invalid gauge
ocm_agent_operator_pull_secret_invalid{ocmagent_name="test-ocm-agent"} 0
`
				err = testutil.CollectAndCompare(localmetrics.MetricPullSecretInvalid, strings.NewReader(expectedMetric))
				Expect(err).To(BeNil())
			})
		})
	})
})
