package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Sentinel is a specification for a Sentinel resource
type Sentinel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SentinelSpec   `json:"spec"`
	Status SentinelStatus `json:"status"`
}

// SentinelSpec is the spec for a Sentinel resource
type SentinelSpec struct {
	Foo string `json:"foo"`
}

// SentinelStatus is the status for a Sentinel resource
type SentinelStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SentinelList is a list of Sentinel resources
type SentinelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Sentinel `json:"items"`
}
