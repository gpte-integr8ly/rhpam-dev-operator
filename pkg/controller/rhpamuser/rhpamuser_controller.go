package rhpamuser

import (
	"context"

	rhpamv1alpha1 "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/apis/rhpam/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var defaultRoles = [7]string{"admin", "process-admin", "manager", "analyst", "developer", "user", "kie-server"}

var defaultUsers = [4]rhpamv1alpha1.User{rhpamv1alpha1.User{Username: "adminuser", Password: "admin1!", Roles: []string{"admin", "kie-server"}},
	rhpamv1alpha1.User{Username: "controlleruser", Password: "controller1!", Roles: []string{"kie-server"}},
	rhpamv1alpha1.User{Username: "mavenuser", Password: "maven1!", Roles: []string{}},
	rhpamv1alpha1.User{Username: "executionuser", Password: "execution1!", Roles: []string{"kie-server"}}}

var log = logf.Log.WithName("controller_rhpamuser")

// Add creates a new RhpamUser Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	return &ReconcileRhpamUser{
		client:       client,
		phaseHandler: NewPhaseHandler(client),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rhpamuser-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rhpamv1alpha1.RhpamUser{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRhpamUser{}

type ReconcileRhpamUser struct {
	client       client.Client
	phaseHandler *phaseHandler
}

func (r *ReconcileRhpamUser) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RhpamUser")

	//only interested if rhpamuser object is created in same namespace as the operator
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		reqLogger.Error(err, "failed to get watch namespace")
		return reconcile.Result{}, nil
	}

	if namespace != request.Namespace {
		reqLogger.Info("Request originated from another namespace. Ignoring...")
		return reconcile.Result{}, nil
	}

	// Fetch the RhpamUser instance
	rhpam := &rhpamv1alpha1.RhpamUser{}
	err = r.client.Get(context.TODO(), request.NamespacedName, rhpam)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("rhpamuser not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error when getting rhpamuser object")
		return reconcile.Result{}, err
	}

	rhpamCopy := rhpam.DeepCopy()

	if rhpamCopy.GetDeletionTimestamp() != nil {
		switch rhpamCopy.Status.Phase {
		case rhpamv1alpha1.PhaseDeprovisioned:
			rhpamCopy.Finalizers = []string{}
			rhpamCopy.Status.Phase = rhpamv1alpha1.PhaseComplete
			return r.handleResult(rhpamCopy, nil, false)
		default:
			rhpamState, err := r.phaseHandler.Deprovision(rhpamCopy)
			if err != nil {
				rhpamState.Status.Phase = rhpamv1alpha1.PhaseDeprovisionFailed
				r.client.Update(context.TODO(), rhpamState)
				return r.handleResult(rhpamState, err, false)
			}
			return r.handleResult(rhpamState, nil, true)
		}
	}

	switch rhpamCopy.Status.Phase {
	case rhpamv1alpha1.NoPhase:
		rhpamState, err := r.phaseHandler.Initialize(rhpamCopy)
		return r.handleResult(rhpamState, err, true)
	case rhpamv1alpha1.PhaseAccepted:
		rhpamState, err := r.phaseHandler.Accepted(rhpamCopy)
		return r.handleResult(rhpamState, err, true)
	case rhpamv1alpha1.PhaseReconcile:
		rhpamState, err := r.phaseHandler.Reconcile(rhpamCopy)
		return r.handleResult(rhpamState, err, false)
	case rhpamv1alpha1.PhaseComplete:
		// no-op
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRhpamUser) handleResult(rhpam *rhpamv1alpha1.RhpamUser, err error, requeue bool) (reconcile.Result, error) {
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: requeue}, r.client.Update(context.TODO(), rhpam)
}
