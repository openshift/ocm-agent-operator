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
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
)

// MFN test constants - only values used multiple times
const (
	mfnAPIVersion    = "ocmagent.managed.openshift.io/v1alpha1"
	mfnKind          = "ManagedFleetNotification"
	mfnTestName      = "test-mfn-no-controller"
	noControllerWait = 10 * time.Second

	mfnrKind     = "ManagedFleetNotificationRecord"
	mfnrTestName = "test-mfnr-stale-deletion"
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
	ginkgo.It("validates ManagedFleetNotification has no controller behavior", func(ctx context.Context) {
		ginkgo.By("ensuring test prerequisites")
		Expect(client.Get(ctx, namespace, "", &corev1.Namespace{})).Should(BeNil(), "namespace %s must exist", namespace)

		ginkgo.By("cleaning up existing MFN if present for idempotency")
		existingMFN := &unstructured.Unstructured{}
		existingMFN.SetAPIVersion(mfnAPIVersion)
		existingMFN.SetKind(mfnKind)
		err := client.Get(ctx, mfnTestName, namespace, existingMFN)
		if err == nil {
			ginkgo.By("deleting existing MFN resource")
			Expect(client.Delete(ctx, existingMFN)).Should(BeNil(), "failed to delete existing MFN")
			Eventually(func() bool {
				checkMFN := &unstructured.Unstructured{}
				checkMFN.SetAPIVersion(mfnAPIVersion)
				checkMFN.SetKind(mfnKind)
				err := client.Get(ctx, mfnTestName, namespace, checkMFN)
				return err != nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue(), "existing MFN should be deleted")
		}

		ginkgo.By("creating new ManagedFleetNotification CR")
		mfn := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": mfnAPIVersion,
				"kind":       mfnKind,
				"metadata": map[string]interface{}{
					"name":      mfnTestName,
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"fleetNotification": map[string]interface{}{
						"name":                "test-notification-e2e",
						"summary":             "E2E Test MFN No Controller",
						"notificationMessage": "Testing MFN has no controller behavior",
						"severity":            "Info",
						"resendWait":          1,
					},
				},
			},
		}
		Expect(client.Create(ctx, mfn)).Should(BeNil(), "failed to create ManagedFleetNotification")

		ginkgo.DeferCleanup(func() {
			ginkgo.By("cleaning up MFN test resource")
			_ = client.Delete(ctx, mfn)
		})

		ginkgo.By("capturing baseline resource version for no-controller verification")
		baseline := &unstructured.Unstructured{}
		baseline.SetAPIVersion(mfnAPIVersion)
		baseline.SetKind(mfnKind)
		Expect(client.Get(ctx, mfnTestName, namespace, baseline)).Should(BeNil(), "failed to get created MFN")
		originalVersion := baseline.GetResourceVersion()
		originalGeneration := baseline.GetGeneration()

		ginkgo.By("verifying MFN spec matches expected values")
		spec := baseline.Object["spec"].(map[string]interface{})
		fleetNotif := spec["fleetNotification"].(map[string]interface{})
		Expect(fleetNotif["name"]).To(Equal("test-notification-e2e"))
		Expect(fleetNotif["severity"]).To(Equal("Info"))
		Expect(fleetNotif["resendWait"]).To(Equal(int64(1)))

		ginkgo.By("monitoring MFN for no controller activity over time")
		Consistently(func() []interface{} {
			current := &unstructured.Unstructured{}
			current.SetAPIVersion(mfnAPIVersion)
			current.SetKind(mfnKind)
			if err := client.Get(ctx, mfnTestName, namespace, current); err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to get MFN resource during monitoring: %v", err))
				return nil
			}
			return []interface{}{current.GetResourceVersion(), current.GetGeneration()}
		}, noControllerWait, 1*time.Second).Should(Equal([]interface{}{originalVersion, originalGeneration}))

		ginkgo.By("verifying final state shows no controller modifications")
		final := &unstructured.Unstructured{}
		final.SetAPIVersion(mfnAPIVersion)
		final.SetKind(mfnKind)
		Expect(client.Get(ctx, mfnTestName, namespace, final)).Should(BeNil())
		Expect(final.Object["status"]).To(BeNil())
	})

	ginkgo.It("validates ManagedFleetNotificationRecord deletes stale records and keeps recent ones", func(ctx context.Context) {
		ginkgo.By("Checking Namespace")
		Expect(client.Get(ctx, namespace, "", &corev1.Namespace{})).Should(BeNil(), "namespace %s must exist", namespace)

		ginkgo.By("cleaning up existing test MFNR")
		existingMFNR := &unstructured.Unstructured{}
		existingMFNR.SetAPIVersion(mfnAPIVersion)
		existingMFNR.SetKind(mfnrKind)
		err := client.Get(ctx, mfnrTestName, namespace, existingMFNR)
		if err == nil {
			ginkgo.By("deleting existing MFNR resource")
			Expect(client.Delete(ctx, existingMFNR)).Should(BeNil(), "failed to delete existing MFNR")
			Eventually(func() bool {
				checkMFNR := &unstructured.Unstructured{}
				checkMFNR.SetAPIVersion(mfnAPIVersion)
				checkMFNR.SetKind(mfnrKind)
				err := client.Get(ctx, mfnrTestName, namespace, checkMFNR)
				return err != nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue(), "existing MFNR should be deleted")
		}

		ginkgo.By("create test MNFR")
		mfnr := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": mfnAPIVersion,
				"kind":       mfnrKind,
				"metadata": map[string]interface{}{
					"name":      mfnrTestName,
					"namespace": namespace,
				},
			},
		}
		Expect(client.Create(ctx, mfnr)).Should(BeNil(), "failed to create ManagedFleetNotificationRecord")

		ginkgo.By("adding status with old and new notification records")

		// Create a controller-runtime client for status updates
		// If we tried to add status into the object and call client.Create or client.Update, the API server rejects it
		// so we use a controller-runtime client to update the status
		cfg, err := config.GetConfig()
		Expect(err).Should(BeNil(), "failed to get kubeconfig for controller-runtime client")
		scheme := runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(ocmagentv1alpha1.SchemeBuilder.AddToScheme(scheme))
		ctrlClient, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
		Expect(err).Should(BeNil(), "failed to create controller-runtime client")

		// Get the typed resource and update its status
		typedMFNR := &ocmagentv1alpha1.ManagedFleetNotificationRecord{}
		Expect(ctrlClient.Get(ctx, ctrlclient.ObjectKey{Name: mfnrTestName, Namespace: namespace}, typedMFNR)).Should(BeNil(), "failed to get MFNR for status update")

		// Update status with stale and recent records
		typedMFNR.Status = ocmagentv1alpha1.ManagedFleetNotificationRecordStatus{
			ManagementCluster: "test-mc-id",
			NotificationRecordByName: []ocmagentv1alpha1.NotificationRecordByName{
				{
					NotificationName: "test-notification",
					ResendWait:       24,
					NotificationRecordItems: []ocmagentv1alpha1.NotificationRecordItem{
						{
							HostedClusterID:               "test-cluster-1",
							FiringNotificationSentCount:   2,
							ResolvedNotificationSentCount: 1,
							LastTransitionTime:            nil,
						},
						{
							HostedClusterID:               "test-cluster-2",
							FiringNotificationSentCount:   1,
							ResolvedNotificationSentCount: 0,
							LastTransitionTime:            &metav1.Time{Time: time.Now()},
						},
						{
							HostedClusterID:               "test-cluster-3",
							FiringNotificationSentCount:   2,
							ResolvedNotificationSentCount: 1,
							LastTransitionTime:            &metav1.Time{Time: time.Now().Add(-30 * 24 * time.Hour)},
						},
					},
				},
			},
		}

		Expect(ctrlClient.Status().Update(ctx, typedMFNR)).Should(BeNil(), "failed to update MFNR status")

		ginkgo.By("waiting for controller to reconcile and delete stale record")
		Eventually(func() bool {
			current := &unstructured.Unstructured{}
			current.SetAPIVersion(mfnAPIVersion)
			current.SetKind(mfnrKind)
			if err := client.Get(ctx, mfnrTestName, namespace, current); err != nil {
				return false
			}

			status, ok := current.Object["status"].(map[string]interface{})
			if !ok {
				return false
			}

			notificationRecords, ok := status["notificationRecordByName"].([]interface{})
			if !ok || len(notificationRecords) == 0 {
				return false
			}

			firstRecord, ok := notificationRecords[0].(map[string]interface{})
			if !ok {
				return false
			}

			items, ok := firstRecord["notificationRecordItems"].([]interface{})
			if !ok {
				return false
			}

			// After reconciliation, only the new record should remain
			if len(items) != 1 {
				return false
			}

			item, ok := items[0].(map[string]interface{})
			if !ok {
				return false
			}

			// Verify it's the new cluster ID
			clusterID, ok := item["hostedClusterID"].(string)
			return ok && clusterID == "test-cluster-2"
		}, 60*time.Second, 2*time.Second).Should(BeTrue(), "stale record should be deleted and only new record should remain")

		ginkgo.By("verifying the old record was deleted and new record is kept")
		final := &unstructured.Unstructured{}
		final.SetAPIVersion(mfnAPIVersion)
		final.SetKind(mfnrKind)
		Expect(client.Get(ctx, mfnrTestName, namespace, final)).Should(BeNil(), "failed to get MFNR after reconciliation")

		status := final.Object["status"].(map[string]interface{})
		notificationRecords := status["notificationRecordByName"].([]interface{})
		Expect(len(notificationRecords)).To(Equal(1), "should have one notification record")

		firstRecord := notificationRecords[0].(map[string]interface{})
		items := firstRecord["notificationRecordItems"].([]interface{})
		Expect(len(items)).To(Equal(1), "should have only one notification record item after stale deletion")

		remainingItem := items[0].(map[string]interface{})
		clusterID := remainingItem["hostedClusterID"].(string)
		Expect(clusterID).To(Equal("test-cluster-2"), "remaining record should be the newest entry")

		// Verify the stale entries were removed
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			Expect(itemMap["hostedClusterID"]).NotTo(Equal("test-cluster-1"), "nil lastTransitionTime entry should be deleted")
			Expect(itemMap["hostedClusterID"]).NotTo(Equal("test-cluster-3"), "stale timestamp entry should be deleted")
		}

		ginkgo.By("cleaning up MFNR test resource")
		Expect(client.Delete(ctx, mfnr)).Should(BeNil(), "failed to delete MFNR test resource")
	})
})
