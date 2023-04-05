/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string
type ConditionType string

const (
	StateReady      = "Ready"
	StateError      = "Error"
	StateProcessing = "Processing"
	StateDeleting   = "Deleting"

	ConditionAvailable   ConditionType = "Available"
	ConditionStatefulSet ConditionType = "StatefulSet"

	ConditionReasonProcessing           = ConditionReason("Processing")
	ConditionReasonDeploying            = ConditionReason("Deploying")
	ConditionReasonDeployed             = ConditionReason("Deployed")
	ConditionReasonDeployError          = ConditionReason("FailedDeploy")
	ConditionReasonStatefulSetAvailable = ConditionReason("Available")
	ConditionReasonStatefulSetPending   = ConditionReason("Pending")
	ConditionReasonSyncFailError        = ConditionReason("FailedToSyncResources")
	ConditionReasonManifestError        = ConditionReason("InvalidManifests")
	ConditionReasonDeletionError        = ConditionReason("DeletionError")
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Cluster struct {
	// Size of a NATS cluster, i.e. number of NATS nodes
	Size int `json:"size"`
}

// NATSSpec defines the desired state of NATS.
type NATSSpec struct {
	Cluster Cluster `json:"cluster"`
}

// NATSStatus defines the observed state of NATS.
type NATSStatus struct {
	State      string             `json:"state"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (n *NATS) IsInDeletion() bool {
	return !n.DeletionTimestamp.IsZero()
}

//nolint:lll //this is annotation
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of NATS deployment"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the resource"

// NATS is the Schema for the nats API.
type NATS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NATSSpec   `json:"spec,omitempty"`
	Status NATSStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NATSList contains a list of NATS.
type NATSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATS `json:"items"`
}

func init() { //nolint:gochecknoinits //called in external function
	SchemeBuilder.Register(&NATS{}, &NATSList{})
}
