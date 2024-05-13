package ocmagenthandler

import (
	"context"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	testconst "github.com/openshift/ocm-agent-operator/pkg/consts/test/init"
	clientmocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks/client"
	v1 "k8s.io/api/policy/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("OCM Agent Pod Disruption Budget Handler", func() {
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

	Context("Managing the OCM Agent Pod Disruption Budget", func() {
		var testPDB v1.PodDisruptionBudget
		var testNamespacedName types.NamespacedName

		BeforeEach(func() {
			testNamespacedName = oah.BuildNamespacedName(testOcmAgent.Name + "-pdb")
			testPDB = *buildOCMAgentPodDisruptionBudget(testOcmAgent)
		})

		It("creates the PDB if it does not exist", func() {
			notFound := k8serrs.NewNotFound(schema.GroupResource{}, "")
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound)
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, pdb *v1.PodDisruptionBudget, opts ...client.CreateOptions) error {
					Expect(pdb.Name).To(Equal(testPDB.Name))
					Expect(pdb.Namespace).To(Equal(testPDB.Namespace))
					Expect(pdb.Spec).To(Equal(testPDB.Spec))
					return nil
				},
			)
			err := testOcmAgentHandler.ensurePodDisruptionBudget(testOcmAgent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("updates the PDB if MinAvailable differs from the expected", func() {
			differentPDB := testPDB
			differentPDB.Spec.MinAvailable = &intstr.IntOrString{IntVal: 2}

			// Mock the Get call to return this modified PDB
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, differentPDB)

			// Expect the Update to be called with the correct spec
			mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, pdb *v1.PodDisruptionBudget, opts ...client.UpdateOptions) error {
					expectedPDB := buildOCMAgentPodDisruptionBudget(testOcmAgent)
					Expect(pdb.Spec.MinAvailable).To(Equal(expectedPDB.Spec.MinAvailable))
					return nil
				},
			)
			err := testOcmAgentHandler.ensurePodDisruptionBudget(testOcmAgent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("updates the PDB if Selector.MatchLabels differs from the expected", func() {
			differentPDB := testPDB
			differentPDB.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"key": "oldValue"},
			}

			// Mock the Get call to return this modified PDB
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, differentPDB)

			// Expect the Update to be called with the correct spec
			mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, pdb *v1.PodDisruptionBudget, opts ...client.UpdateOptions) error {
					expectedPDB := buildOCMAgentPodDisruptionBudget(testOcmAgent)
					Expect(pdb.Spec.Selector.MatchLabels).To(Equal(expectedPDB.Spec.Selector.MatchLabels))
					return nil
				},
			)
			err := testOcmAgentHandler.ensurePodDisruptionBudget(testOcmAgent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("updates the PDB if MaxUnavailable differs from the expected", func() {
			differentPDB := testPDB
			differentPDB.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: 1}

			// Mock the Get call to return this modified PDB
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, differentPDB)

			// Expect the Update to be called with the correct spec
			mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, pdb *v1.PodDisruptionBudget, opts ...client.UpdateOptions) error {
					expectedPDB := buildOCMAgentPodDisruptionBudget(testOcmAgent)
					Expect(pdb.Spec.MaxUnavailable).To(Equal(expectedPDB.Spec.MaxUnavailable))
					return nil
				},
			)
			err := testOcmAgentHandler.ensurePodDisruptionBudget(testOcmAgent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the PDB if it exists", func() {
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).SetArg(2, testPDB)
			mockClient.EXPECT().Delete(gomock.Any(), &testPDB).Return(nil)
			err := testOcmAgentHandler.ensurePodDisruptionBudgetDeleted(testOcmAgent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("does nothing if the PDB is already removed", func() {
			notFound := k8serrs.NewNotFound(schema.GroupResource{}, testPDB.Name)
			mockClient.EXPECT().Get(gomock.Any(), testNamespacedName, gomock.Any()).Return(notFound)
			err := testOcmAgentHandler.ensurePodDisruptionBudgetDeleted(testOcmAgent)
			Expect(err).To(BeNil())
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})
})
