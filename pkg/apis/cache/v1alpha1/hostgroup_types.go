package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HostGroupSpec defines the desired state of HostGroup
type HostGroupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	HostGroup Group `json:"hostgroup,omitempty"`

	Endpoints []string `json:"endpoints"`
}

// HostGroupStatus defines the observed state of HostGroup
type HostGroupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

type Group struct {
	Id   int       `json:"id,omitempty"`
	Name string    `json:"name,omitempty"`
	Vars Variables `json:"variables,omitempty"`
}

// Used when unmarshalling Group.Vars which contains a yaml string
type Variables struct {
	Type            string    `json:"mo_type,omitempty"`
	Endpoint        string    `json:"mo_endpoint,omitempty"`
	BearerTokenFile string    `json:"mo_bearerTokenFile,omitempty"`
	Port            string    `json:"mo_port,omitempty"`
	Scheme          string    `json:"mo_scheme,omitempty"`
	TargetPort      int       `json:"mo_targetPort,omitempty"`
	TLSConf         TLSConfig `json:"mo_tlsConfig,omitempty"`
}

// Used when unmarshalling Variables.TLSConf which contains a yaml string
type TLSConfig struct {
	CAFile             string `json:"mo_caFile,omitempty"`
	Hostname           string `json:"mo_hostname,omitempty"`
	InsecureSkipVerify bool   `json:"mo_insecureSkipVerify,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostGroup is the Schema for the hostgroups API
// +k8s:openapi-gen=true
type HostGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostGroupSpec   `json:"spec,omitempty"`
	Status HostGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostGroupList contains a list of HostGroup
type HostGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HostGroup{}, &HostGroupList{})
}
