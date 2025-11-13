// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /osde2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

var _ = ginkgo.Describe("ocm-agent-operator", ginkgo.Ordered, func() {
	var (
		client             *resources.Resources
		configMapName      = "ocm-agent-cm"
		clusterRolePrefix  = "ocm-agent-operator"
		deploymentName     = "ocm-agent"
		namespace          = "openshift-ocm-agent-operator"
		networkPolicyName  = "ocm-agent-allow-only-alertmanager"
		secretName         = "ocm-access-token"
		serviceMonitorName = "ocm-agent-metrics"
		serviceName        = "ocm-agent"
		operatorName       = "ocm-agent-operator"
		operatorNamespace  = "openshift-ocm-agent-operator"

		deployments = []string{
			deploymentName,
			deploymentName + "-operator",
		}
	)

	ginkgo.BeforeAll(func() {
		// setup the k8s client
		cfg, err := config.GetConfig()
		Expect(err).Should(BeNil(), "failed to get kubeconfig")
		client, err = resources.New(cfg)
		Expect(err).Should(BeNil(), "resources.New error")
		Expect(monitoringv1.AddToScheme(client.GetScheme())).Should(BeNil(), "unable to register monitoringv1 api scheme")
	})

	ginkgo.It("is installed", func(ctx context.Context) {
		ginkgo.By("checking the namespace exists")
		err := client.Get(ctx, namespace, "", &corev1.Namespace{})
		Expect(err).Should(BeNil(), "namespace %s not found", namespace)

		ginkgo.By("checking the secret exists")
		err = client.Get(ctx, secretName, namespace, &corev1.Secret{})
		Expect(err).Should(BeNil(), "secret %s/%s not found", namespace, secretName)

		ginkgo.By("checking the service exists")
		err = client.Get(ctx, serviceName, namespace, &corev1.Service{})
		Expect(err).Should(BeNil(), "service %s/%s not found", namespace, serviceName)

		ginkgo.By("checking the service monitor exists")
		err = client.Get(ctx, serviceMonitorName, namespace, &monitoringv1.ServiceMonitor{})
		Expect(err).Should(BeNil(), "service monitor %s/%s not found", namespace, serviceMonitorName)

		ginkgo.By("checking the networkpolicy exists")
		err = client.Get(ctx, networkPolicyName, namespace, &networkingv1.NetworkPolicy{})
		Expect(err).Should(BeNil(), "networkpolicy %s/%s not found", namespace, networkPolicyName)

		ginkgo.By("checking the clusterroles exists")
		var clusterRoles rbacv1.ClusterRoleList
		err = client.List(ctx, &clusterRoles)
		Expect(err).Should(BeNil(), "failed to list clusterroles")
		found := false
		for _, clusterRole := range clusterRoles.Items {
			if strings.HasPrefix(clusterRole.Name, clusterRolePrefix) {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "unable to find cluster role")

		ginkgo.By("cluster role bindings exist")
		var clusterRoleBindings rbacv1.ClusterRoleBindingList
		err = client.List(ctx, &clusterRoleBindings)
		Expect(err).Should(BeNil(), "unable to list clusterrolebindings")
		found = false
		for _, clusterRoleBinding := range clusterRoleBindings.Items {
			if strings.HasPrefix(clusterRoleBinding.Name, clusterRolePrefix) {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "unable to find clusterrolebinding")

		ginkgo.By("checking the deployment exists")
		for _, deploymentName := range deployments {
			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: namespace}}
			err = wait.For(conditions.New(client).DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, corev1.ConditionTrue))
			Expect(err).Should(BeNil(), "deployment %s not available", deploymentName)
		}
	})

	ginkgo.It("reconciles required resources", func(ctx context.Context) {
		resources := &metav1.List{
			Items: []runtime.RawExtension{
				{Object: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: namespace}}},
				{Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: namespace}}},
				{Object: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace}}},
				{Object: &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: namespace}}},
				{Object: &monitoringv1.ServiceMonitor{ObjectMeta: metav1.ObjectMeta{Name: serviceMonitorName, Namespace: namespace}}},
				{Object: &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: networkPolicyName, Namespace: namespace}}},
			},
		}

		for _, resource := range resources.Items {
			obj := resource.Object.(k8s.Object)
			Expect(client.Delete(ctx, obj)).Should(BeNil(), "failed to delete resources")
		}

		Expect(wait.For(conditions.New(client).ResourcesFound(resources))).Should(BeNil(), "some resources were never found")
	})

	ginkgo.PIt("can be upgraded", func(ctx context.Context) {
		log.SetLogger(ginkgo.GinkgoLogr)
		k8sClient, err := openshift.New(ginkgo.GinkgoLogr)
		Expect(err).ShouldNot(HaveOccurred(), "unable to setup k8s client")

		ginkgo.By("forcing operator upgrade")
		err = k8sClient.UpgradeOperator(ctx, operatorName, operatorNamespace)
		Expect(err).NotTo(HaveOccurred(), "operator upgrade failed")
	})

	ginkgo.It("respects replicas parameter and creates PDB when replicas > 1", func(ctx context.Context) {
		testOCMAgentName := "test-ocm-agent-replicas"
		testOCMAgentPDB := testOCMAgentName + "-pdb"
		testReplicas := int64(2)

		baseURL := os.Getenv("OCM_BASE_URL")
		if baseURL == "" {
			cm := &corev1.ConfigMap{}
			if err := client.Get(ctx, configMapName, namespace, cm); err == nil {
				baseURL = cm.Data["ocmBaseURL"]
			}
		}
		if baseURL == "" {
			ginkgo.Skip("OCM_BASE_URL not set")
		}

		gvk := schema.GroupVersionKind{
			Group:   "ocmagent.managed.openshift.io",
			Version: "v1alpha1",
			Kind:    "OcmAgent",
		}

		ocmAgentCR := &unstructured.Unstructured{}
		ocmAgentCR.SetGroupVersionKind(gvk)
		if err := client.Get(ctx, testOCMAgentName, namespace, ocmAgentCR); err == nil {
			Expect(client.Delete(ctx, ocmAgentCR)).To(BeNil())
		}

		newOcmAgent := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "ocmagent.managed.openshift.io/v1alpha1",
				"kind":       "OcmAgent",
				"metadata":   map[string]interface{}{"name": testOCMAgentName, "namespace": namespace},
				"spec": map[string]interface{}{
					"agentConfig": map[string]interface{}{
						"ocmBaseUrl": baseURL,
						"services":   []interface{}{"service_logs", "clusters_mgmt"},
					},
					"ocmAgentImage": "quay.io/app-sre/ocm-agent:latest",
					"replicas":      testReplicas,
					"tokenSecret":   "ocm-access-token",
				},
			},
		}
		newOcmAgent.SetGroupVersionKind(gvk)
		Expect(client.Create(ctx, newOcmAgent)).To(BeNil())

		var deployment *appsv1.Deployment
		Eventually(func() bool {
			deploy := &appsv1.Deployment{}
			if err := client.Get(ctx, testOCMAgentName, namespace, deploy); err != nil {
				return false
			}
			deployment = deploy
			return deploy.Spec.Replicas != nil &&
				*deploy.Spec.Replicas == int32(testReplicas) &&
				deploy.Status.ReadyReplicas == int32(testReplicas)
		}, 120*time.Second, 5*time.Second).Should(BeTrue())

		Expect(*deployment.Spec.Replicas).To(Equal(int32(testReplicas)))

		pdb := &policyv1.PodDisruptionBudget{}
		Eventually(func() bool {
			return client.Get(ctx, testOCMAgentPDB, namespace, pdb) == nil
		}, 120*time.Second, 5*time.Second).Should(BeTrue())

		Expect(pdb.Spec.MinAvailable.IntVal).To(Equal(int32(1)))
		Expect(pdb.Spec.Selector.MatchLabels["app"]).To(Equal(testOCMAgentName))

		Expect(client.Delete(ctx, newOcmAgent)).To(BeNil())
	})

	ginkgo.It("respects ocmAgentImage parameter", func(ctx context.Context) {
		testOCMAgentName := "test-ocm-agent-image"
		customImage := "quay.io/app-sre/ocm-agent:test-image"

		baseURL := os.Getenv("OCM_BASE_URL")
		if baseURL == "" {
			cm := &corev1.ConfigMap{}
			if err := client.Get(ctx, configMapName, namespace, cm); err == nil {
				baseURL = cm.Data["ocmBaseURL"]
			}
		}
		if baseURL == "" {
			ginkgo.Skip("OCM_BASE_URL not set")
		}

		gvk := schema.GroupVersionKind{
			Group:   "ocmagent.managed.openshift.io",
			Version: "v1alpha1",
			Kind:    "OcmAgent",
		}

		ocmAgentCR := &unstructured.Unstructured{}
		ocmAgentCR.SetGroupVersionKind(gvk)
		if err := client.Get(ctx, testOCMAgentName, namespace, ocmAgentCR); err == nil {
			Expect(client.Delete(ctx, ocmAgentCR)).To(BeNil())
		}

		newOcmAgent := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "ocmagent.managed.openshift.io/v1alpha1",
				"kind":       "OcmAgent",
				"metadata":   map[string]interface{}{"name": testOCMAgentName, "namespace": namespace},
				"spec": map[string]interface{}{
					"agentConfig": map[string]interface{}{
						"ocmBaseUrl": baseURL,
						"services":   []interface{}{"service_logs", "clusters_mgmt"},
					},
					"ocmAgentImage": customImage,
					"replicas":      int64(1),
					"tokenSecret":   "ocm-access-token",
				},
			},
		}
		newOcmAgent.SetGroupVersionKind(gvk)
		Expect(client.Create(ctx, newOcmAgent)).To(BeNil())

		var deployment *appsv1.Deployment
		Eventually(func() bool {
			deploy := &appsv1.Deployment{}
			if err := client.Get(ctx, testOCMAgentName, namespace, deploy); err != nil {
				return false
			}
			deployment = deploy
			return deploy.Spec.Replicas != nil &&
				deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
				deploy.Status.ReadyReplicas > 0
		}, 120*time.Second, 5*time.Second).Should(BeTrue())

		Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(customImage))

		Expect(client.Delete(ctx, newOcmAgent)).To(BeNil())
	})
})
