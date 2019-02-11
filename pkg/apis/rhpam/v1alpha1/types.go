package v1alpha1

type StatusPhase string

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
	PhaseComplete                 StatusPhase = "complete"
)
