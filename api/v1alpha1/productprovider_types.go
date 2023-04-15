// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LABEL_FROM_PRODUCT_PROVIDER = "resource.nautes.io/from"
)

// ProductProviderSpec defines the desired state of ProductProvider
type ProductProviderSpec struct {
	Type string `json:"type" yaml:"type"`
	Name string `json:"name" yaml:"name"`
}

// ProductProviderStatus defines the observed state of ProductProvider
type ProductProviderStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
}

func (status *ProductProviderStatus) GetConditions(conditionTypes map[string]bool) []metav1.Condition {
	result := make([]metav1.Condition, 0)
	for i := range status.Conditions {
		condition := status.Conditions[i]
		if ok := conditionTypes[condition.Type]; ok {
			result = append(result, condition)
		}

	}
	return result
}

func (status *ProductProviderStatus) SetConditions(conditions []metav1.Condition, evaluatedTypes map[string]bool) {
	appConditions := make([]metav1.Condition, 0)
	for i := 0; i < len(status.Conditions); i++ {
		condition := status.Conditions[i]
		if _, ok := evaluatedTypes[condition.Type]; !ok {
			appConditions = append(appConditions, condition)
		}
	}
	for i := range conditions {
		condition := conditions[i]
		eci := findConditionIndexByType(status.Conditions, condition.Type)
		if eci >= 0 &&
			status.Conditions[eci].Message == condition.Message &&
			status.Conditions[eci].Status == condition.Status &&
			status.Conditions[eci].Reason == condition.Reason {
			// If we already have a condition of this type, only update the timestamp if something
			// has changed.
			status.Conditions[eci].LastTransitionTime = metav1.Now()
			appConditions = append(appConditions, status.Conditions[eci])
		} else {
			// Otherwise we use the new incoming condition with an updated timestamp:
			condition.LastTransitionTime = metav1.Now()
			appConditions = append(appConditions, condition)
		}
	}
	sort.Slice(appConditions, func(i, j int) bool {
		left := appConditions[i]
		right := appConditions[j]
		return fmt.Sprintf("%s/%s/%v", left.Type, left.Message, left.LastTransitionTime) < fmt.Sprintf("%s/%s/%v", right.Type, right.Message, right.LastTransitionTime)
	})
	status.Conditions = appConditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProductProvider is the Schema for the productproviders API
type ProductProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProductProviderSpec   `json:"spec,omitempty"`
	Status ProductProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProductProviderList contains a list of ProductProvider
type ProductProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProductProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProductProvider{}, &ProductProviderList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (p *ProductProvider) Compare(obj client.Object) (bool, error) {
	return false, nil
}
