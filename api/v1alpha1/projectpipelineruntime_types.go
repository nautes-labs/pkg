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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CalendarEventSource struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	// +optional
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	// +optional
	Interval string `json:"interval" protobuf:"bytes,2,opt,name=interval"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.
	ExclusionDates []string `json:"exclusionDates,omitempty" protobuf:"bytes,3,rep,name=exclusionDates"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty" protobuf:"bytes,4,opt,name=timezone"`
}

type EventSource struct {
	// disabled or enabled
	Webhook  string              `json:"webhook,omitempty"`
	Calendar CalendarEventSource `json:"calendar,omitempty"`
}

type Pipeline struct {
	Name string `json:"name,omitempty"`
	// Default is 'default'
	Label string `json:"label"`
	// Branch name, wildcard support.
	Branch string `json:"branch,omitempty"`
	// Pipeline manifest path, wildcard support.
	Path string `json:"path,omitempty"`
	// Definition of events that trigger pipeline
	EventSources []EventSource `json:"eventsource"`
}

// ProjectPipelineRuntimeSpec defines the desired state of ProjectPipelineRuntime
type ProjectPipelineRuntimeSpec struct {
	Project string `json:"project,omitempty"`
	// Code repo for pipeline manifests.
	PipelineSource string `json:"pipelinesource"`
	// Other code repos used in pipeline.
	// +optional
	CodeSources []string `json:"codesources,omitempty"`
	// Definition of pipeline.
	// +optional
	Pipelines []Pipeline `json:"pipelines,omitempty"`
	// Target environment for running the pipeline.
	Destination string `json:"destination"`
}

func (r *ProjectPipelineRuntime) GetProduct() string {
	return r.Namespace
}

// ProjectPipelineRuntimeStatus defines the observed state of ProjectPipelineRuntime
type ProjectPipelineRuntimeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectPipelineRuntime is the Schema for the projectpipelineruntimes API
type ProjectPipelineRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectPipelineRuntimeSpec   `json:"spec,omitempty"`
	Status ProjectPipelineRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectPipelineRuntimeList contains a list of ProjectPipelineRuntime
type ProjectPipelineRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectPipelineRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectPipelineRuntime{}, &ProjectPipelineRuntimeList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (p *ProjectPipelineRuntime) Compare(obj client.Object) (bool, error) {
	val, ok := obj.(*ProjectPipelineRuntime)
	if !ok {
		return false, fmt.Errorf("the resource %s type is inconsistent", obj.GetName())
	}

	for i := 0; i < len(p.Spec.Pipelines); i++ {
		for j := 0; j < len(val.Spec.Pipelines); j++ {
			if p.Spec.PipelineSource == val.Spec.PipelineSource &&
				p.Spec.Destination == val.Spec.Destination &&
				p.Spec.Pipelines[i].Path == val.Spec.Pipelines[j].Path {
				return true, nil
			}
		}
	}

	return false, nil
}
