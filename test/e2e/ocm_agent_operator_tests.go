// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /osde2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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

		// Test-specific resource names
		testOCMAgentName         = "test-ocm-agent"
		testOCMAgentConfigMap    = testOCMAgentName + "-cm"
		testOCMAgentServiceMon   = testOCMAgentName + "-metrics"
		testOCMAgentNetPolicy    = testOCMAgentName + "-allow-only-alertmanager"
		testOCMAgentNetPolicyMUO = testOCMAgentName + "-allow-muo-communication"

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

	ginkgo.It("create a new OcmAgent CR and verify operator-created resources", func(ctx context.Context) {
		const (
			waitTimeout  = 60 * time.Second
			waitInterval = 5 * time.Second
		)

		ginkgo.By("Setting up GroupVersionKind for OcmAgent CR")
		ocmAgentCR := &unstructured.Unstructured{}
		ocmAgentCR.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "ocmagent.managed.openshift.io",
			Version: "v1alpha1",
			Kind:    "OcmAgent",
		})
		ginkgo.By("Cleaning up existing OcmAgent CR if present")
		err := client.Get(ctx, testOCMAgentName, namespace, ocmAgentCR)
		if err == nil {
			Expect(client.Delete(ctx, ocmAgentCR)).To(BeNil(), "Failed to delete existing OcmAgent CR")
			Eventually(func() bool {
				err := client.Get(ctx, testOCMAgentName, namespace, ocmAgentCR)
				return err != nil
			}, waitTimeout, waitInterval).Should(BeTrue(), "OcmAgent CR not deleted in time")
		}
		ginkgo.By("Creating a new OcmAgent CR")
		newOcmAgent := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "ocmagent.managed.openshift.io/v1alpha1",
				"kind":       "OcmAgent",
				"metadata": map[string]interface{}{
					"name":      testOCMAgentName,
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"agentConfig": map[string]interface{}{
						"ocmBaseUrl": "https://api.stage.openshift.com",
						"services":   []interface{}{"service_logs", "clusters_mgmt"},
					},
					"ocmAgentImage": "quay.io/slamba_openshift/ocm-agent-new:latest",
					"replicas":      int64(1),
					"tokenSecret":   "ocm-access-token",
				},
			},
		}
		newOcmAgent.SetGroupVersionKind(ocmAgentCR.GroupVersionKind())
		Expect(client.Create(ctx, newOcmAgent)).To(BeNil(), "Failed to create new OcmAgent CR")
		ginkgo.By("Waiting for Deployment to be ready")
		Eventually(func() bool {
			deploy := &appsv1.Deployment{}
			err := client.Get(ctx, testOCMAgentName, namespace, deploy)
			if err != nil || deploy.Spec.Replicas == nil {
				return false
			}
			return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas && deploy.Status.ReadyReplicas > 0
		}, waitTimeout, waitInterval).Should(BeTrue(), "Deployment not ready")
		ginkgo.By("Verifying operator-created resources exist")
		verifyExists := func(name string, obj k8s.Object) {
			Expect(client.Get(ctx, name, namespace, obj)).To(BeNil(), fmt.Sprintf("%T %s not found", obj, name))
		}
		verifyExists(testOCMAgentName, &appsv1.Deployment{})
		verifyExists(testOCMAgentName, &corev1.Service{})
		verifyExists(testOCMAgentConfigMap, &corev1.ConfigMap{})
		verifyExists(testOCMAgentServiceMon, &monitoringv1.ServiceMonitor{})
		verifyExists(testOCMAgentNetPolicy, &networkingv1.NetworkPolicy{})
		verifyExists(testOCMAgentNetPolicyMUO, &networkingv1.NetworkPolicy{})
	})

	ginkgo.It("deletes test OCMAgent CR and verifies cleanup", func(ctx context.Context) {

		ocmAgentCR := &unstructured.Unstructured{}
		ocmAgentCR.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "ocmagent.managed.openshift.io",
			Version: "v1alpha1",
			Kind:    "OcmAgent",
		})
		err := client.Get(ctx, testOCMAgentName, namespace, ocmAgentCR)
		if err != nil {
			ginkgo.Skip(fmt.Sprintf("OcmAgent CR '%s' not found in namespace '%s'; skipping test", testOCMAgentName, namespace))
		}
		resourceExists := func(name string, obj k8s.Object) {
			Expect(client.Get(ctx, name, namespace, obj)).To(BeNil(), fmt.Sprintf("Expected resource %T %s to exist", obj, name))
		}

		resourceExists(testOCMAgentName, &appsv1.Deployment{})
		resourceExists(testOCMAgentName, &corev1.Service{})
		resourceExists(testOCMAgentConfigMap, &corev1.ConfigMap{})
		resourceExists(testOCMAgentServiceMon, &monitoringv1.ServiceMonitor{})
		resourceExists(testOCMAgentNetPolicy, &networkingv1.NetworkPolicy{})
		resourceExists(testOCMAgentNetPolicyMUO, &networkingv1.NetworkPolicy{})

		Expect(client.Delete(ctx, ocmAgentCR)).To(BeNil(), "Failed to delete OcmAgent CR")
		Eventually(func() bool {
			err := client.Get(ctx, testOCMAgentName, namespace, ocmAgentCR)
			return err != nil
		}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "OcmAgent CR not deleted in expected time")
		checkDeleted := func(name string, obj k8s.Object) {
			Eventually(func() bool {
				err := client.Get(ctx, name, namespace, obj)
				return err != nil
			}, 1*time.Minute, 5*time.Second).Should(BeTrue(), fmt.Sprintf("%T %s was not deleted", obj, name))
		}
		checkDeleted(testOCMAgentName, &appsv1.Deployment{})
		checkDeleted(testOCMAgentName, &corev1.Service{})
		checkDeleted(testOCMAgentConfigMap, &corev1.ConfigMap{})
		checkDeleted(testOCMAgentServiceMon, &monitoringv1.ServiceMonitor{})
		checkDeleted(testOCMAgentNetPolicy, &networkingv1.NetworkPolicy{})
		checkDeleted(testOCMAgentNetPolicyMUO, &networkingv1.NetworkPolicy{})
	})
})
