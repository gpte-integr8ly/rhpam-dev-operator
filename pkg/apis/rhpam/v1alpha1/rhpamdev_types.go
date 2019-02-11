package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RhpamDevSpec defines the desired state of RhpamDev
type RhpamDevSpec struct {
	Config RhpamConfig `json:"config,omitempty"`
}

// RhpamDevStatus defines the observed state of RhpamDev
type RhpamDevStatus struct {
	Phase   StatusPhase `json:"phase"`
	Version string      `json:"version"`
}

type RhpamConfig struct {
	DatabaseConfig        RhpamDatabaseConfig        `json:"database,omitempty"`
	BusinessCentralConfig RhpamBusinessCentralConfig `json:"businessCentral,omitempty"`
	KieServerConfig       RhpamKieServerConfig       `json:"kieServer,omitempty"`
}

type RhpamDatabaseConfig struct {
	PersistentVolumeCapacity string `json:"persistentVolumeCapacity,omitempty"`
	MaxConnections           string `json:"maxConnections,omitempty"`
	SharedBuffers            string `json:"sharedBuffers,omitempty"`
	MaxPreparedTransactions  string `json:"maxPreparedTransactions,omitempty"`
	MemoryLimit              string `json:"memoryLimit,omitempty"`
}

type RhpamBusinessCentralConfig struct {
	PersistentVolumeCapacity string `json:"persistentVolumeCapacity,omitempty"`
	CpuRequest               string `json:"cpuRequest,omitempty"`
	CpuLimit                 string `json:"cpuLimit,omitempty"`
	MemoryRequest            string `json:"memoryRequest,omitempty"`
	MemoryLimit              string `json:"memoryLimit,omitempty"`
	GcMaxMetaSize            string `json:"gcMaxMetaSize,omitempty"`
	KieMBeans                string `json:"kieMbeans,omitempty"`
	JavaOptsAppend           string `json:"javaOptsAppend,omitempty"`
}

type RhpamKieServerConfig struct {
	CpuRequest          string `json:"cpuRequest,omitempty"`
	CpuLimit            string `json:"cpuLimit,omitempty"`
	MemoryRequest       string `json:"memoryRequest,omitempty"`
	MemoryLimit         string `json:"memoryLimit,omitempty"`
	GcMaxMetaSize       string `json:"gcMaxMetaSize,omitempty"`
	KieMBeans           string `json:"kieMbeans,omitempty"`
	KieServerId         string `json:"kieServerId,omitempty"`
	DroolsFilterClasses string `json:"filterClasses,omitempty"`
	BypassAuthUser      string `json:"bypassAuthUser,omitEmpty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RhpamDev is the Schema for the rhpamdevs API
// +k8s:openapi-gen=true
type RhpamDev struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RhpamDevSpec   `json:"spec,omitempty"`
	Status RhpamDevStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RhpamDevList contains a list of RhpamDev
type RhpamDevList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RhpamDev `json:"items"`
}

type StatusPhase string

var (
	NoPhase                       StatusPhase = ""
	PhaseInitialized              StatusPhase = "initialized"
	PhasePrepared                 StatusPhase = "prepared"
	PhaseDatabaseInstalled        StatusPhase = "database installed"
	PhaseBusinessCentralInstalled StatusPhase = "business central installed"
	PhaseDatabaseReady            StatusPhase = "database ready"
	PhaseKieServerInstalled       StatusPhase = "kie server installed"
	PhaseComplete                 StatusPhase = "complete"
)

func init() {
	SchemeBuilder.Register(&RhpamDev{}, &RhpamDevList{})
}

func (r *RhpamDev) Defaults() {
}

func (r *RhpamDev) Validate() error {
	return nil
}
