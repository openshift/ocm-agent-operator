/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocmagent

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	ctrlconst "github.com/openshift/ocm-agent-operator/pkg/consts/controller"
	"github.com/openshift/ocm-agent-operator/pkg/localmetrics"
	"github.com/openshift/ocm-agent-operator/pkg/ocmagenthandler"
	monitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OcmAgentReconciler reconciles a OcmAgent object
type OcmAgentReconciler struct {
	Client                 client.Client
	Scheme                 *runtime.Scheme
	OCMAgentHandlerBuilder ocmagenthandler.OcmAgentHandlerBuilder
}

var log = logf.Log.WithName("controller_ocmagent")

var _ reconcile.Reconciler = &OcmAgentReconciler{}

//+kubebuilder:rbac:groups=ocmagent.managed.openshift.io,resources=ocmagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocmagent.managed.openshift.io,resources=ocmagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocmagent.managed.openshift.io,resources=ocmagents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OcmAgent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *OcmAgentReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OCMAgent")

	// Fetch the OCMAgent instance
	instance := ocmagentv1alpha1.OcmAgent{}
	err := r.Client.Get(ctx, request.NamespacedName, &instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			localmetrics.UpdateMetricOcmAgentResourceAbsent()
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to retrieve OCMAgent. Will retry on next reconcile.")
		return reconcile.Result{}, err
	}
	localmetrics.ResetMetricOcmAgentResourceAbsent()
	oaohandler, err := r.OCMAgentHandlerBuilder.New()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Is the OCMAgent being deleted?
	if !instance.DeletionTimestamp.IsZero() {
		log.V(2).Info("Entering EnsureOCMAgentResourcesAbsent")
		err := oaohandler.EnsureOCMAgentResourcesAbsent(instance)
		if err != nil {
			log.Error(err, "Failed to remove OCMAgent. Will retry on next reconcile.")
			return reconcile.Result{}, err
		}
		// The finalizer can now be removed
		if controllerutil.ContainsFinalizer(&instance, ctrlconst.ReconcileOCMAgentFinalizer) {
			controllerutil.RemoveFinalizer(&instance, ctrlconst.ReconcileOCMAgentFinalizer)
			if err := r.Client.Update(ctx, &instance); err != nil {
				log.Error(err, "Failed to remove finalizer from OCMAgent resource. Will retry on next reconcile.")
				return reconcile.Result{}, err
			}
		}
		log.Info("Successfully removed OCMAgent resources.")
	} else {
		// There needs to be an OCM Agent
		log.V(2).Info("Entering EnsureOCMAgentResourcesExist")
		err := oaohandler.EnsureOCMAgentResourcesExist(instance)
		if err != nil {
			log.Error(err, "Failed to create OCMAgent. Will retry on next reconcile.")
			return reconcile.Result{}, err
		}

		// The OCM Agent is deployed, so set a finalizer on the resource
		if !controllerutil.ContainsFinalizer(&instance, ctrlconst.ReconcileOCMAgentFinalizer) {
			controllerutil.AddFinalizer(&instance, ctrlconst.ReconcileOCMAgentFinalizer)
			if err := r.Client.Update(ctx, &instance); err != nil {
				log.Error(err, "Failed to apply finalizer to OCMAgent resource. Will retry on next reconcile.")
				return reconcile.Result{}, err
			}
		}
		log.Info("Successfully setup OCMAgent resources.")
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OcmAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&ocmagentv1alpha1.OcmAgent{}).
		Owns(&netv1.NetworkPolicy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&monitorv1.ServiceMonitor{}).
		Complete(r)
}
