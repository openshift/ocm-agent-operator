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

package fleetnotification

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
)

const (
	NotificationRecordStaleTimeoutInHour int32 = 360
)

// ManagedFleetNotificationReconciler reconciles a ManagedFleetNotification object
type ManagedFleetNotificationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var log = logf.Log.WithName("controller_fleetnotification")

var _ reconcile.Reconciler = &ManagedFleetNotificationReconciler{}

//+kubebuilder:rbac:groups=ocmagent.managed.openshift.io,resources=managedfleetnotifications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocmagent.managed.openshift.io,resources=managedfleetnotifications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocmagent.managed.openshift.io,resources=managedfleetnotifications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedFleetNotification object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ManagedFleetNotificationReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling FleetNotification")

	nr := ocmagentv1alpha1.ManagedFleetNotificationRecord{}

	err := r.Client.Get(ctx, request.NamespacedName, &nr)
	if err != nil {
		return reconcile.Result{}, err
	}

	for n, rn := range nr.Status.NotificationRecordByName {
		resendWait := rn.ResendWait
		for i, ri := range rn.NotificationRecordItems {
			// Consider the record is stale if the lastSendTime is older than resendWait + 15 days
			eol := ri.LastTransitionTime.Time.Add(time.Duration(resendWait+NotificationRecordStaleTimeoutInHour) * time.Hour)
			if time.Now().After(eol) {
				log.Info(fmt.Sprintf("NotificationRecord for notification %s and hostedcluster %s has not been updated "+
					"for %d hours and considered as stale, cleaning up...", rn.NotificationName, ri.HostedClusterID,
					resendWait+NotificationRecordStaleTimeoutInHour))

				patch := []byte(fmt.Sprintf(`[{"op": "remove", "path": "/status/notificationRecordByName/%d/notificationRecordItems/%d"}]`, n, i))

				err = r.Client.Status().Patch(ctx, &nr, client.RawPatch(types.JSONPatchType, patch))
				if err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedFleetNotificationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&ocmagentv1alpha1.ManagedFleetNotificationRecord{}).
		Complete(r)
}
