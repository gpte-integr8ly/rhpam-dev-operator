package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Role struct {
	Name string `json:"name"`
}

type User struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

// RhpamUserSpec defines the desired state of RhpamUser
type RhpamUserSpec struct {
	Roles []*Role `json:"roles"`
	Users []*User `json:"users"`
}

// RhpamUserStatus defines the observed state of RhpamUser
type RhpamUserStatus struct {
	Phase StatusPhase `json:"phase"`
	Realm string      `json:"realm"`
}

// RhpamUser is the Schema for the rhpamusers API
// +k8s:openapi-gen=true
type RhpamUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RhpamUserSpec   `json:"spec,omitempty"`
	Status RhpamUserStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RhpamUserList contains a list of RhpamUser
type RhpamUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RhpamUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RhpamUser{}, &RhpamUserList{})
}
