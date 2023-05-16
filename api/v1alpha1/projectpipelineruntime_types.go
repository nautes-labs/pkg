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

type Gitlab struct {
	// Gitlab project name.
	RepoName string `json:"repoName,omitempty"`
	// Supports wildcards.
	Revision string `json:"revision,omitempty"`
	// Gitlab webhook events: PushEvents, TagPushEvents, etc.
	Events []string `json:"events,omitempty"`
}

type Calendar struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	// +optional
	Schedule string `json:"schedule,omitempty"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	// +optional
	Interval string `json:"interval,omitempty"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.
	ExclusionDates []string `json:"exclusionDates,omitempty"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

type EventSource struct {
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name     string    `json:"name,omitempty"`
	Gitlab   *Gitlab   `json:"gitlab,omitempty"`
	Calendar *Calendar `json:"calendar,omitempty"`
}

// The definition of event source triggered pipeline mode.
type PipelineTrigger struct {
	EventSource string `json:"eventSource,omitempty"`
	Pipeline    string `json:"pipeline,omitempty"`
	// Optional, does not support wildcards. If it is empty, the trigger will determine the revision of the pipeline based on the revision of the event source.
	Revision string `json:"revision,omitempty"`
}

// The definition of a multi-branch pipeline.One pipeline corresponds to one declaration file in the Git repository.
type Pipeline struct {
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name,omitempty"`
	// Default is 'default'
	Label string `json:"label,omitempty"`
	// Pipeline manifest path, wildcard support.
	Path string `json:"path,omitempty"`
}

// ProjectPipelineRuntimeSpec defines the desired state of ProjectPipelineRuntime
type ProjectPipelineRuntimeSpec struct {
	Project string `json:"project,omitempty"`
	// The code repo for pipeline manifests.
	PipelineSource string `json:"pipelinesource,omitempty"`
	// The definition of pipeline.
	Pipelines []Pipeline `json:"pipelines,omitempty"`
	// The target environment for running the pipeline.
	Destination string `json:"destination,omitempty"`
	// Events source that may trigger the pipeline.
	EventSources []EventSource `json:"eventsources,omitempty"`
	// Isolation definition of pipeline runtime related resources: shared(default) or exclusive
	Isolation        string            `json:"isolation,omitempty"`
	PipelineTriggers []PipelineTrigger `json:"pipeline_triggers,omitempty"`
}

func (r *ProjectPipelineRuntime) GetProduct() string {
	return r.Namespace
}

func (r *ProjectPipelineRuntime) GetDestination() string {
	return r.Spec.Destination
}

func (r *ProjectPipelineRuntime) GetEventSource(name string) (*EventSource, error) {
	for _, eventSource := range r.Spec.EventSources {
		if eventSource.Name == name {
			return &eventSource, nil
		}
	}

	return nil, fmt.Errorf("can not find event source %s in runtime", name)
}

func (r *ProjectPipelineRuntime) GetPipeline(name string) (*Pipeline, error) {
	for _, pipeline := range r.Spec.Pipelines {
		if pipeline.Name == name {
			return &pipeline, nil
		}
	}

	return nil, fmt.Errorf("can not find pipeline %s in runtime", name)
}

// ProjectPipelineRuntimeStatus defines the observed state of ProjectPipelineRuntime
type ProjectPipelineRuntimeStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
	// +optional
	Cluster string `json:"cluster,omitempty"`
	// +optional
	// +nullable
	// IllegalEventSources records eventsources that will not be synchronized to the environment,
	// and why it will not be synchronized to the past
	IllegalEventSources []IllegalEventSource `json:"illegalEventSources"`
}

type IllegalEventSource struct {
	EventSource EventSource `json:"eventSource"`
	Reason      string      `json:"reason"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=ppr
//+kubebuilder:printcolumn:name="Project",type=string,JSONPath=".spec.project"
//+kubebuilder:printcolumn:name="Source",type=string,JSONPath=".spec.pipelinesource"

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
