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
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterType string
type ClusterKind string
type ClusterUsage string
type ClusterSyncStatus string
type ClusterWorkType string

const (
	CLUSTER_KIND_KUBERNETES ClusterKind = "kubernetes"
)

const (
	CLUSTER_TYPE_PHYSICAL ClusterType = "physical"
	CLUSTER_TYPE_VIRTUAL  ClusterType = "virtual"
)

const (
	CLUSTER_USAGE_HOST   ClusterUsage = "host"
	CLUSTER_USAGE_WORKER ClusterUsage = "worker"
)

const (
	DEPLOYMENT_TYPE ClusterWorkType = "deployment"
	PIPELINE_TYPE   ClusterWorkType = "pipeline"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// +kubebuilder:validation:Pattern=`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&\/\/=]*)`
	ApiServer string `json:"apiserver" yaml:"apiserver"`
	// +kubebuilder:validation:Enum=physical;virtual
	ClusterType ClusterType `json:"clustertype" yaml:"clustertype"`
	// +optional
	// +kubebuilder:default:=kubernetes
	// +kubebuilder:validation:Enum=kubernetes
	ClusterKind ClusterKind `json:"clusterKind" yaml:"clusterKind"`
	// the usage of cluster, for user use it directry or deploy vcluster on it
	// +kubebuilder:validation:Enum=host;worker
	Usage ClusterUsage `json:"usage" yaml:"usage"`
	// +optional
	HostCluster string `json:"hostcluster" yaml:"hostcluster"`
	// +optional
	// +nullable
	// PrimaryDomain is used to build the domain of components within the cluster
	PrimaryDomain string `json:"primaryDomain" yaml:"primaryDomain"`
	// +optional
	// +nullable
	// +kubebuilder:validation:Enum="";pipeline;deployment
	// pipeline or deployment, when the cluster usage is 'worker', the WorkType is required.
	WorkerType ClusterWorkType `json:"workerType" yaml:"workerType"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
	// +optional
	MgtAuthStatus *MgtClusterAuthStatus `json:"mgtAuthStatus,omitempty" yaml:"mgtAuthStatus"`
	// +optional
	Sync2ArgoStatus *SyncCluster2ArgoStatus `json:"sync2ArgoStatus,omitempty" yaml:"sync2ArgoStatus"`
	// +optional
	// +nullable
	EntryPoints map[string]ClusterEntryPoint `json:"entryPoints,omitempty" yaml:"entryPoints"`
}

type ServiceType string

const (
	ServiceTypeNodePort     ServiceType = "NodePort"
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
	ServiceTypeExternalName ServiceType = "ExternalName"
)

type ClusterEntryPoint struct {
	// The port of entry point service
	HTTPPort  int32       `json:"httpPort,omitempty" yaml:"httpPort"`
	HTTPSPort int32       `json:"httpsPort,omitempty" yaml:"httpsPort"`
	Type      ServiceType `json:"type,omitempty" yaml:"type"`
}

type MgtClusterAuthStatus struct {
	LastSuccessSpec string      `json:"lastSuccessSpec" yaml:"lastSuccessSpec"`
	LastSuccessTime metav1.Time `json:"lastSuccessTime" yaml:"lastSuccessTime"`
	SecretID        string      `json:"secretID" yaml:"secretID"`
}

type SyncCluster2ArgoStatus struct {
	LastSuccessSpec string      `json:"lastSuccessSpec" yaml:"lastSuccessSpec"`
	LastSuccessTime metav1.Time `json:"lastSuccessTime" yaml:"lastSuccessTime"`
	SecretID        string      `json:"secretID" yaml:"secretID"`
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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="API SERVER",type=string,JSONPath=`.spec.apiserver`
//+kubebuilder:printcolumn:name="USAGE",type=string,JSONPath=".spec.usage"
//+kubebuilder:printcolumn:name="HOST",type=string,JSONPath=".spec.hostcluster"
//+kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (c *Cluster) Compare(obj client.Object) (bool, error) {
	return false, nil
}
