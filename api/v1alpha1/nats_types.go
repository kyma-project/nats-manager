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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string

type ConditionType string

const (
	StateReady      = "Ready"
	StateError      = "Error"
	StateProcessing = "Processing"
	StateDeleting   = "Deleting"
	// StateDeleted is used only in deleted condition. Not a modularization compliant state.
	StateDeleted = "Deleted"

	ConditionReasonDeploying     = ConditionReason("Deploying")
	ConditionReasonDeployed      = ConditionReason("Deployed")
	ConditionReasonDeletion      = ConditionReason("Deletion")
	ConditionReasonDeployError   = ConditionReason("DeployError")
	ConditionReasonDeletionError = ConditionReason("DeletionError")
)


type Cluster struct {
	// Size of a NATS cluster, i.e. number of NATS nodes
	Size int `json:"size"`
}

// NATSSpec defines the desired state of Nats.
type NATSSpec struct {
	Cluster Cluster `json:"cluster"`
}

// NATSStatus defines the observed state of Nats.
type NATSStatus struct {
	State      string             `json:"state"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (n *NATS) UpdateStateFromErr(c ConditionType, r ConditionReason, err error) {
	n.Status.State = StateError
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "False",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            err.Error(),
	}
	meta.SetStatusCondition(&n.Status.Conditions, condition)
}

func (n *NATS) UpdateStateReady(c ConditionType, r ConditionReason, msg string) {
	n.Status.State = StateReady
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "True",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&n.Status.Conditions, condition)
}

func (n *NATS) UpdateStateProcessing(c ConditionType, r ConditionReason, msg string) {
	n.Status.State = StateProcessing
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "Unknown",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&n.Status.Conditions, condition)
}

func (n *NATS) UpdateStateDeletion(c ConditionType, r ConditionReason, msg string) {
	n.Status.State = StateDeleting
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "Unknown",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&n.Status.Conditions, condition)
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

// NATSList contains a list of Nats.
type NATSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATS `json:"items"`
}

func init() { //nolint:gochecknoinits //called in external function
	SchemeBuilder.Register(&NATS{}, &NATSList{})
}
