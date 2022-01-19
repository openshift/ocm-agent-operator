package ocmagent

import (
	"context"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	ctrlconst "github.com/openshift/ocm-agent-operator/pkg/consts/controller"
	oahconst "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	"github.com/openshift/ocm-agent-operator/pkg/ocmagenthandler"
)

// ReconcileOCMAgent reconciles a OCMAgent object
type ReconcileOCMAgent struct {
	Client client.Client
	Scheme *runtime.Scheme
	Ctx    context.Context
	Log    logr.Logger

	OCMAgentHandler ocmagenthandler.OCMAgentHandler
}

var log = logf.Log.WithName("controller_ocmagent")

// Add creates a new OCMAgent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new ReconcileOCMAgent
func newReconciler(mgr manager.Manager) *ReconcileOCMAgent {
	client := mgr.GetClient()
	log := ctrl.Log.WithName("controllers").WithName("OCMAgent")
	ctx := context.Background()
	scheme := mgr.GetScheme()
	return &ReconcileOCMAgent{
		Client:          client,
		Scheme:          scheme,
		Ctx:             ctx,
		Log:             log,
		OCMAgentHandler: ocmagenthandler.New(client, scheme, log, ctx),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileOCMAgent) error {
	// Create a new controller
	c, err := controller.New("ocmagent-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OCMAgent
	err = c.Watch(&source.Kind{Type: &ocmagentv1alpha1.OcmAgent{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to Deployments
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ocmagentv1alpha1.OcmAgent{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Services
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ocmagentv1alpha1.OcmAgent{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to OAO ConfigMaps
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ocmagentv1alpha1.OcmAgent{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to OAO Secrets
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ocmagentv1alpha1.OcmAgent{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to pull secret
	psPredicate := predicate.Funcs{
		UpdateFunc:  func(e event.UpdateEvent) bool { return handlePullSecret(e.ObjectNew) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return handlePullSecret(e.Object) },
		CreateFunc:  func(e event.CreateEvent) bool { return handlePullSecret(e.Object) },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(pullSecretToOCMAgent(r.Client, r.Ctx, r.Log)), psPredicate)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileOCMAgent implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOCMAgent{}

// Reconcile reads that state of the cluster for a OCMAgent object and makes changes based on the state read
// and what is in the OCMAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOCMAgent) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.Ctx = ctx
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
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to retrieve OCMAgent. Will retry on next reconcile.")
		return reconcile.Result{}, err
	}

	// Is the OCMAgent being deleted?
	if !instance.DeletionTimestamp.IsZero() {
		log.V(2).Info("Entering EnsureOCMAgentResourcesAbsent")
		err := r.OCMAgentHandler.EnsureOCMAgentResourcesAbsent(instance)
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
		err := r.OCMAgentHandler.EnsureOCMAgentResourcesExist(instance)
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

// handlePullSecret returns true if meta indicates it is the cluster pull secret
func handlePullSecret(meta metav1.Object) bool {
	pullSecretNamespacedName := oahconst.BuildPullSecretNamespacedName()
	return meta.GetNamespace() == pullSecretNamespacedName.Namespace &&
		meta.GetName() == pullSecretNamespacedName.Name
}
