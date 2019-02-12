package rhpamdev

import (
	"context"

	rhpamv1alpha1 "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/apis/rhpam/v1alpha1"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/k8sutil"
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
	RhpamVersion                        = "7.2.0.GA"
	ApplicationName                     = "rhpam"
	ServiceAccount                      = "rhpam"
	DatabaseCredentialsSecret           = "rhpam-postgresql"
	DatabaseName                        = "rhpam"
	DatabasePvc                         = "rhpam-postgresql"
	DatabaseService                     = "rhpam-postgresql"
	DatabaseDeployment                  = "rhpam-postgresql"
	DatabaseVolumeCapacity              = "1Gi"
	DatabaseImage                       = "registry.redhat.io/rhscl/postgresql-96-rhel7:latest"
	DatabaseMaxConnections              = "100"
	DatabaseMaxPreparedTransactions     = "100"
	DatabaseSharedBuffers               = "32MB"
	DatabaseMemoryLimit                 = "512Mi"
	DatabaseInitConfigmap               = "rhpam-postgresql-init"
	BusinessCentralService              = "rhpam-bc"
	BusinessCentralPvc                  = "rhpam-bc"
	BusinessCentralRoute                = "rhpam-bc"
	BusinessCentralDeployment           = "rhpam-bc"
	BusinessCentralVolumeCapacity       = "1Gi"
	BusinessCentralImageStreamNamespace = "openshift"
	BusinessCentralImage                = "rhpam72-businesscentral-openshift"
	BusinessCentralImageTag             = "1.0"
	BusinessCentralCpuRequest           = "200m"
	BusinessCentralCpuLimit             = "2000m"
	BusinessCentralMemoryRequest        = "1Gi"
	BusinessCentralMemoryLimit          = "3Gi"
	KieAdminUser                        = "adminuser"
	KieAdminPassword                    = "admin1!"
	BusinessCentralGcMaxMetaSize        = "500"
	BusinessCentralKieMBeans            = "enabled"
	KieServerControllerUser             = "controllerUser"
	KieServerControllerPassword         = "controller1!"
	KieServerUser                       = "executionuser"
	KieServerPassword                   = "execution1!"
	KieMavenUser                        = "mavenuser"
	KieMavenPassword                    = "maven1!"
	EapAdminUserName                    = "eapadmin"
	EapAdminPassword                    = "eapadmin1!"
	BusinessCentralJavaOptsAppend       = "-Dorg.uberfire.nio.git.ssh.algorithm=RSA"
	BusinessCentralRealmSecret          = "rhpam-bc-sso"
	KieServerRoute                      = "rhpam-kieserver"
	KieServerService                    = "rhpam-kieserver"
	KieServerDeployment                 = "rhpam-kieserver"
	KieServerImageStreamNamespace       = "openshift"
	KieServerImage                      = "rhpam72-kieserver-openshift"
	KieServerImageTag                   = "1.0"
	KieServerCpuRequest                 = "200m"
	KieServerCpuLimit                   = "1000m"
	KieServerMemoryRequest              = "1Gi"
	KieServerMemoryLimit                = "3Gi"
	KieServerGcMaxMetaSize              = "500"
	KieServerDroolsFilterClasses        = "true"
	KieServerBypassAuthUser             = "false"
	KieServerControllerProtocol         = "ws"
	KieServerId                         = "kieserver-dev"
	KieServerKieMBeans                  = "enabled"
	KieServerRealmSecret                = "rhpam-ks-sso"
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
	err = c.Watch(&source.Kind{Type: &rhpamv1alpha1.RhpamDev{}}, &handler.EnqueueRequestForObject{})
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
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRhpamDev) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RhpamDev")

	//only interested if rhpamdev object is created in same namespace as the operator
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		reqLogger.Error(err, "failed to get watch namespace")
		return reconcile.Result{}, nil
	}

	if namespace != request.Namespace {
		reqLogger.Info("Request originated from another namespace. Ignoring...")
		return reconcile.Result{}, nil
	}

	// Fetch the RhpamDev instance
	rhpam := &rhpamv1alpha1.RhpamDev{}
	err1 := r.client.Get(context.TODO(), request.NamespacedName, rhpam)
	if err1 != nil {
		if errors.IsNotFound(err1) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("rhpamdev not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err1, "Error when getting rhpamdev object")
		return reconcile.Result{}, err1
	}

	rhpamCopy := rhpam.DeepCopy()

	if rhpamCopy.GetDeletionTimestamp() != nil {
		rhpamState, err := r.phaseHandler.Deprovision(rhpamCopy)
		rhpamState.Finalizers = []string{}
		return r.handleResult(rhpamState, err)
	}

	switch rhpamCopy.Status.Phase {
	case rhpamv1alpha1.NoPhase:
		rhpamState, err := r.phaseHandler.Initialize(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhaseAccepted:
		rhpamState, err := r.phaseHandler.ProvisionRealm(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhasePrepare:
		rhpamState, err := r.phaseHandler.Prepare(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhasePrepared:
		rhpamState, err := r.phaseHandler.InstallDatabase(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhaseDatabaseInstalled:
		rhpamState, err := r.phaseHandler.InstallBusinessCentral(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhaseBusinessCentralInstalled:
		ready, rhpamState, err := r.phaseHandler.WaitForDatabase(rhpamCopy)
		if !ready && err == nil {
			return reconcile.Result{Requeue: true}, nil
		}
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhaseDatabaseReady:
		rhpamState, err := r.phaseHandler.InstallKieServer(rhpamCopy)
		return r.handleResult(rhpamState, err)
	case rhpamv1alpha1.PhaseComplete:
		reqLogger.Info("RHPAM installation complete")
	}

	return reconcile.Result{}, nil

}

func (r *ReconcileRhpamDev) handleResult(rhpam *rhpamv1alpha1.RhpamDev, err error) (reconcile.Result, error) {
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, r.client.Update(context.TODO(), rhpam)
}
