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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var projectpipelineruntimelog = logf.Log.WithName("projectpipelineruntime-resource")

func (r *ProjectPipelineRuntime) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-projectpipelineruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=create;update,versions=v1alpha1,name=vprojectpipelineruntime.kb.io,admissionReviewVersions=v1
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepobindings,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes,verbs=get;list;watch

var _ webhook.Validator = &ProjectPipelineRuntime{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateCreate() error {
	projectpipelineruntimelog.Info("validate create", "name", r.Name)
	client, err := getClient()
	if err != nil {
		return err
	}

	illegalEventSources, err := r.Validate(context.TODO(), &ValidateClientK8s{Client: client})
	if err != nil {
		return err
	}

	if len(illegalEventSources) != 0 {
		failureReasons := []string{}
		for _, illegalEventSource := range illegalEventSources {
			failureReasons = append(failureReasons, illegalEventSource.Reason)
		}
		return fmt.Errorf("no permission code repo found %v", failureReasons)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateUpdate(old runtime.Object) error {
	projectpipelineruntimelog.Info("validate update", "name", r.Name)
	client, err := getClient()
	if err != nil {
		return err
	}

	illegalEventSources, err := r.Validate(context.TODO(), &ValidateClientK8s{Client: client})
	if err != nil {
		return err
	}

	if len(illegalEventSources) != 0 {
		failureReasons := []string{}
		for _, illegalEventSource := range illegalEventSources {
			failureReasons = append(failureReasons, illegalEventSource.Reason)
		}
		return fmt.Errorf("no permission code repo found in eventsource %v", failureReasons)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateDelete() error {
	projectpipelineruntimelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

const (
	SelectFieldCodeRepoBindingProductAndRepo = "productAndRepo"
	SelectFieldCodeRepoName                  = "metadata.name"
)

// Validate use to verify pipeline runtime is legal, it will check the following things
// - runtime has permission to use repo in eventsources
// if runtime has no permission to use code repo, it return them in the first var.
func (r *ProjectPipelineRuntime) Validate(ctx context.Context, validateClient ValidateClient) ([]IllegalEventSource, error) {
	productName := r.Namespace
	projectName := r.Spec.Project

	if err := r.StaticCheck(); err != nil {
		return nil, err
	}

	if err := hasCodeRepoPermission(ctx, validateClient, productName, projectName, r.Spec.PipelineSource); err != nil {
		return nil, err
	}

	illegalEvnentSources := []IllegalEventSource{}
	for _, eventSource := range r.Spec.EventSources {

		if eventSource.Gitlab != nil &&
			eventSource.Gitlab.RepoName != "" {
			err := hasCodeRepoPermission(ctx, validateClient, productName, projectName, eventSource.Gitlab.RepoName)
			if err != nil {
				illegalEvnentSources = append(illegalEvnentSources, IllegalEventSource{
					EventSource: eventSource,
					Reason:      err.Error(),
				})
			}

		}
	}

	return illegalEvnentSources, nil
}

func (r *ProjectPipelineRuntime) StaticCheck() error {
	eventSourceNames := make(map[string]bool, 0)
	for _, es := range r.Spec.EventSources {
		if eventSourceNames[es.Name] {
			return fmt.Errorf("event source %s is duplicate", es.Name)
		}
		eventSourceNames[es.Name] = true
	}

	pipelineNames := make(map[string]bool, 0)
	for _, pipeline := range r.Spec.Pipelines {
		if pipelineNames[pipeline.Name] {
			return fmt.Errorf("pipeline %s is duplicate", pipeline.Name)
		}
		pipelineNames[pipeline.Name] = true
	}

	triggerTags := make(map[string]bool, 0)
	for _, trigger := range r.Spec.PipelineTriggers {
		if !eventSourceNames[trigger.EventSource] {
			return fmt.Errorf("found non-existent event source %s in trigger", trigger.EventSource)
		}

		if !pipelineNames[trigger.Pipeline] {
			return fmt.Errorf("found non-existent pipeline %s in trigger", trigger.Pipeline)
		}

		tag := fmt.Sprintf("%s|%s|%s", trigger.EventSource, trigger.Pipeline, trigger.Revision)
		if triggerTags[tag] {
			return fmt.Errorf("trigger is duplicate, event source %s, pipeline %s, trigger %s", trigger.EventSource, trigger.Pipeline, trigger.Revision)
		}
		triggerTags[tag] = true
	}

	return nil
}
