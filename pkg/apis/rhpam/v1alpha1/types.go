package v1alpha1

type StatusPhase string

const (
	RhpamFinalizer = "finalizer.rhpam"
)

var (
	NoPhase                       StatusPhase
	PhaseInitialized              StatusPhase = "initialized"
	PhaseAccepted                 StatusPhase = "accepted"
	PhaseRealmProvisioned         StatusPhase = "realm provisioned"
	PhasePrepare                  StatusPhase = "prepare provisioning"
	PhasePrepared                 StatusPhase = "prepared"
	PhaseDatabaseInstalled        StatusPhase = "database installed"
	PhaseBusinessCentralInstalled StatusPhase = "business central installed"
	PhaseDatabaseReady            StatusPhase = "database ready"
	PhaseKieServerInstalled       StatusPhase = "kie server installed"
	PhaseReconcile                StatusPhase = "reconcile"
	PhaseComplete                 StatusPhase = "complete"
	PhaseDeprovisioned            StatusPhase = "deprovisioned"
	PhaseDeprovisionFailed        StatusPhase = "deprovision failed"
)
