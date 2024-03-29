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
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c ComponentsList) GetNamespaces() []string {
	namespaces := []string{}
	componentType := reflect.TypeOf(&Component{})

	vars := reflect.ValueOf(c)
	for i := 0; i < vars.NumField(); i++ {
		f := vars.Field(i)
		if f.Type() == componentType && !f.IsNil() {
			component := f.Elem().Interface().(Component)
			namespaces = append(namespaces, component.Namespace)
		}

	}

	return namespaces
}

func (c ComponentsList) GetNamespacesMap() map[string]bool {
	reservedNamespace := c.GetNamespaces()
	return convertArrayToBoolMap(reservedNamespace)
}

func (status *ClusterStatus) SetConditions(conditions []metav1.Condition, evaluatedTypes map[string]bool) {
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

func (c *Cluster) SpecToJsonString() string {
	specStr, err := json.Marshal(c.Spec)
	if err != nil {
		return ""
	}
	return string(specStr)
}

func (status *ClusterStatus) GetConditions(conditionTypes map[string]bool) []metav1.Condition {
	result := make([]metav1.Condition, 0)
	for i := range status.Conditions {
		condition := status.Conditions[i]
		if ok := conditionTypes[condition.Type]; ok {
			result = append(result, condition)
		}

	}
	return result
}

func GetClusterFromString(name, namespace string, specStr string) (*Cluster, error) {
	var spec ClusterSpec
	err := json.Unmarshal([]byte(specStr), &spec)
	if err != nil {
		return nil, err
	}
	return &Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}, nil
}
