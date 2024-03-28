// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package v1alpha1

import (
	"github.com/korrel8r/korrel8r/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: Run "make" to regenerate code after modifying this file.
// JSON tags are required for fields to be serialized.

// Korrel8rSpec defines the desired state of Korrel8r
type Korrel8rSpec struct {
	// Config is the configuration for a korrel8r deployment.
	// If not provided there is a default configuration suitable for use in an openshift cluster.
	//
	// The "include" section can load additional configuration files provided at /etc/korrel8r
	Config *Config `json:"config,omitempty"`

	// Debug provides optional settings intended to help with debugging problems.
	Debug *DebugSpec `json:"debug,omitempty"`
}

type DebugSpec struct {
	// Verbose sets the numeric logging verbosity for the KORREL8R_VERBOSE environment variable.
	Verbose int `json:"verbose,omitempty"`
}

// Config wraps the korrel8r Config struct for API code generation.
type Config struct {
	config.Config `json:",inline"`
}

// Korrel8rStatus defines the observed state of Korrel8r
type Korrel8rStatus struct {
	// Conditions conditions.type is one of "Available", "Progressing", and "Degraded"
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Korrel8r is a service that correlates observabililty signals in the cluster.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Korrel8r struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Korrel8rSpec   `json:"spec,omitempty"`
	Status Korrel8rStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// Korrel8rList contains a list of Korrel8r
type Korrel8rList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Korrel8r `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Korrel8r{}, &Korrel8rList{})
}
