package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CodeRepoBindingPermissionReadOnly  = "readonly"
	CodeRepoBindingPermissionReadWrite = "readwrite"
)

// CodeRepoBindingSpec defines the desired state of CodeRepoBinding
type CodeRepoBindingSpec struct {
	// Authorized Code Repository.
	CodeRepo string `json:"coderepo,omitempty"`
	// The Code repo is authorized to this product or projects under it.
	Product string `json:"product,omitempty"`
	// If the project list is empty, it means that the code repo is authorized to the product.
	// If the project list has values, it means that the code repo is authorized to the specified projects.
	// +optional
	Projects []string `json:"projects,omitempty"`
	// Authorization Permissions, readwrite or readonly.
	Permissions string `json:"permissions,omitempty"`
}

// CodeRepoBindingStatus defines the observed state of CodeRepoBinding
type CodeRepoBindingStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="PRODUCTRESOURCENAME",type=string,JSONPath=".spec.product"
//+kubebuilder:printcolumn:name="CODEREPORESOURCENAME",type=string,JSONPath=".spec.coderepo"

// CodeRepoBinding is the Schema for the coderepobindings API
type CodeRepoBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodeRepoBindingSpec   `json:"spec,omitempty"`
	Status CodeRepoBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CodeRepoBindingList contains a list of CodeRepoBinding
type CodeRepoBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeRepoBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodeRepoBinding{}, &CodeRepoBindingList{})
}
