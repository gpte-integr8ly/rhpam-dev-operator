package rhpamdev

import (
	"context"

	gptev1alpha1 "github.com/gpte-naps/rhpam-dev-operator/pkg/apis/gpte/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	RhpamVersion                    = "7.1.1.GA"
	ApplicationName                 = "rhpam"
	ServiceAccount                  = "rhpam"
	ServiceAccountRoleBinding       = "rhpam-view"
	DatabaseCredentialsSecret       = "rhpam-postgresql"
	DatabaseName                    = "rhpam"
	DatabasePvc                     = "rhpam-postgresql"
	DatabaseService                 = "rhpam-postgresql"
	DatabaseDeployment              = "rhpam-postgresql"
	DatabaseVolumeCapacity          = "1Gi"
	DatabaseImage                   = "registry.redhat.io/rhscl/postgresql-96-rhel7:latest"
	DatabaseMaxConnections          = "100"
	DatabaseMaxPreparedTransactions = "100"
	DatabaseSharedBuffers           = "32MB"
	DatabaseMemoryLimit             = "512Mi"
	DatabaseInitConfigmap           = "rhpam-postgresql-init"
)

var log = logf.Log.WithName("controller_rhpamdev")

// Add creates a new RhpamDev Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	scheme := mgr.GetScheme()
	return &ReconcileRhpamDev{
		client:       client,
		scheme:       scheme,
		phaseHandler: NewPhaseHandler(client, scheme),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rhpamdev-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RhpamDev
	err = c.Watch(&source.Kind{Type: &gptev1alpha1.RhpamDev{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RhpamDev
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &gptev1alpha1.RhpamDev{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRhpamDev{}

// ReconcileRhpamDev reconciles a RhpamDev object
type ReconcileRhpamDev struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *runtime.Scheme
	phaseHandler *phaseHandler
}

// Reconcile reads that state of the cluster for a RhpamDev object and makes changes based on the state read
// and what is in the RhpamDev.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRhpamDev) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RhpamDev")

	// Fetch the RhpamDev instance
	rhpam := &gptev1alpha1.RhpamDev{}
	err := r.client.Get(context.TODO(), request.NamespacedName, rhpam)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error when getting rhpamdev object")
		return reconcile.Result{}, err
	}

	rhpamCopy := rhpam.DeepCopy()
	switch rhpamCopy.Status.Phase {
	case gptev1alpha1.NoPhase:
		rhpamState, err := r.phaseHandler.Initialize(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case gptev1alpha1.PhaseInitialized:
		rhpamState, err := r.phaseHandler.Prepare(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case gptev1alpha1.PhasePrepared:
		rhpamState, err := r.phaseHandler.InstallDatabase(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case gptev1alpha1.PhaseComplete:
		reqLogger.Info("RHSSO installation complete")
	}

	return reconcile.Result{}, nil

}

func (r *ReconcileRhpamDev) handleResult(rhpam *gptev1alpha1.RhpamDev, err error) (reconcile.Result, error) {
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, r.client.Update(context.TODO(), rhpam)
}
