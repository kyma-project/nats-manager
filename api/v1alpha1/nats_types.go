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

// +kubebuilder:validation:Optional // This sets 'required' as the default behaviour.
//
//nolint:lll //this is annotation
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string

type ConditionType string

const (
	StateReady      string = "Ready"
	StateError      string = "Error"
	StateProcessing string = "Processing"
	StateDeleting   string = "Deleting"

	ConditionAvailable   ConditionType = "Available"
	ConditionStatefulSet ConditionType = "StatefulSet"

	ConditionReasonProcessing           ConditionReason = "Processing"
	ConditionReasonDeploying            ConditionReason = "Deploying"
	ConditionReasonDeployed             ConditionReason = "Deployed"
	ConditionReasonProcessingError      ConditionReason = "FailedProcessing"
	ConditionReasonStatefulSetAvailable ConditionReason = "Available"
	ConditionReasonStatefulSetPending   ConditionReason = "Pending"
	ConditionReasonSyncFailError        ConditionReason = "FailedToSyncResources"
	ConditionReasonManifestError        ConditionReason = "InvalidManifests"
	ConditionReasonDeletionError        ConditionReason = "DeletionError"
)

/*
NATS uses kubebuilder decorators for validation and defaulting instead of webhooks. Here is an overview to this topic:
https://book.kubebuilder.io/reference/markers/crd-validation.html

Default values are defined at multiple levels to ensure that values are set independent of what level of NATS is defined
or gets deleted. E.g. spec.cluster.size will get set to a default value whether spec.cluster.size, spec.cluster or
spec gets deleted.

Validation utilizes the Common Expression Language (CEL, https://github.com/google/cel-spec). Here is an introduction
doc to this topic: https://kubernetes.io/docs/reference/using-api/cel/.

Testing for validation and defaulting is done via envtest at
nats-manager/internal/controller/nats/integrationtests/validation/integration_test.go.

For testing of defaulting it is advised to send an Unstructured object
(https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured) to the API-Server, because if
not nil-able properties like ResourceRequirements (https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements) stay
undefined they will be interpreted as set to "" and will result into 0, instead of getting replaced by the default
value.
*/

// NATS is the Schema for the nats API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of NATS deployment"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the resource"
type NATS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	// +kubebuilder:default:={jetStream:{fileStorage:{storageClassName:"default", size:"1Gi"},memStorage:{size:"20Mi",enabled:false}}, cluster:{size:3},logging:{trace:false,debug:false}, resources:{limits:{cpu:"20m",memory:"64Mi"}, requests:{cpu:"5m",memory:"16Mi"}}}
	Spec   NATSSpec   `json:"spec,omitempty"`
	Status NATSStatus `json:"status,omitempty"`
}

// NATSStatus defines the observed state of NATS.
type NATSStatus struct {
	State      string             `json:"state"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// NATSSpec defines the desired state of NATS.
type NATSSpec struct {
	// Cluster defines configurations that are specific to NATS clusters.
	// +optional
	// +kubebuilder:default:={size:3}
	Cluster `json:"cluster,omitempty"`

	// JetStream defines configurations that are specific to NATS JetStream.
	// +optional
	// +kubebuilder:default:={fileStorage:{storageClassName:"default", size:"1Gi"},memStorage:{size:"20Mi",enabled:false}}
	JetStream `json:"jetStream,omitempty"`

	// JetStream defines configurations that are specific to NATS logging in NATS.
	// +optional
	// +kubebuilder:default:={trace:false,debug:false}
	Logging `json:"logging,omitempty"`

	// Resources defines resources for NATS.
	// +optional
	// +kubebuilder:default:={limits:{cpu:"20m",memory:"64Mi"}, requests:{cpu:"5m",memory:"16Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Annotations allows to add annotations to NATS.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels allows to add Labels to NATS.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// Cluster defines configurations that are specific to NATS clusters.
type Cluster struct {
	// Size of a NATS cluster, i.e. number of NATS nodes.
	// +optional
	// +kubebuilder:default:=3
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:XValidation:rule="(self%2) != 0", message="size only accepts odd numbers"
	Size int `json:"size,omitempty"`
}

// JetStream defines configurations that are specific to NATS JetStream.
type JetStream struct {
	// MemStorage defines configurations to memory storage in NATS JetStream.
	// +optional
	// +kubebuilder:default:={size:"20Mi",enabled:false}
	MemStorage `json:"memStorage,omitempty"`

	// FileStorage defines configurations to file storage in NATS JetStream.
	// +optional
	// +kubebuilder:default:={storageClassName:"default",size:"1Gi"}
	FileStorage `json:"fileStorage,omitempty"`
}

// MemStorage defines configurations to memory storage in NATS JetStream.
type MemStorage struct {
	// Enabled allows the enablement of memory storage.
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled,omitempty"`

	// Size defines the mem.
	// +kubebuilder:default:="20Mi"
	Size resource.Quantity `json:"size,omitempty"`
}

// FileStorage defines configurations to file storage in NATS JetStream.
type FileStorage struct {
	// StorageClassName defines the file storage class name.
	// +kubebuilder:default:="default"
	StorageClassName string `json:"storageClassName,omitempty"`

	// Size defines the file storage size.
	// +kubebuilder:default:="1Gi"
	Size resource.Quantity `json:"size,omitempty"`
}

// Logging defines logging options.
type Logging struct {
	// Debug allows debug logging.
	// +kubebuilder:default:=false
	Debug bool `json:"debug,omitempty"`

	// Trace allows trace logging.
	// +kubebuilder:default:=false
	Trace bool `json:"trace,omitempty"`
}

// +kubebuilder:object:root=true

// NATSList contains a list of NATS.
type NATSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATS `json:"items"`
}

func (n *NATS) IsInDeletion() bool {
	return !n.DeletionTimestamp.IsZero()
}

func init() { //nolint:gochecknoinits //called in external function
	SchemeBuilder.Register(&NATS{}, &NATSList{})
}
