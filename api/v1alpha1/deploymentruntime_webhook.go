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
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var deploymentruntimelog = logf.Log.WithName("deploymentruntime-resource")

func (r *DeploymentRuntime) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-deploymentruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=create;update,versions=v1alpha1,name=vdeploymentruntime.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DeploymentRuntime{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DeploymentRuntime) ValidateCreate() error {
	deploymentruntimelog.Info("validate create", "name", r.Name)

	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	IllegalProjectRefs, err := r.Validate(context.Background(), &ValidateClientK8s{Client: k8sClient})
	if err != nil {
		return err
	}
	if len(IllegalProjectRefs) != 0 {
		failureReasons := []string{}
		for _, IllegalProjectRef := range IllegalProjectRefs {
			failureReasons = append(failureReasons, IllegalProjectRef.Reason)
		}
		return fmt.Errorf("no permission project reference found %v", failureReasons)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DeploymentRuntime) ValidateUpdate(old runtime.Object) error {
	deploymentruntimelog.Info("validate update", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	IllegalProjectRefs, err := r.Validate(context.Background(), &ValidateClientK8s{Client: k8sClient})
	if err != nil {
		return err
	}
	if len(IllegalProjectRefs) != 0 {
		failureReasons := []string{}
		for _, IllegalProjectRef := range IllegalProjectRefs {
			failureReasons = append(failureReasons, IllegalProjectRef.Reason)
		}
		return fmt.Errorf("no permission project reference found %v", failureReasons)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DeploymentRuntime) ValidateDelete() error {
	deploymentruntimelog.Info("validate delete", "name", r.Name)

	return nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=get;list

// Validate used to check deployment runtime is legal
func (r *DeploymentRuntime) Validate(ctx context.Context, validateClient ValidateClient) ([]IllegalProjectRef, error) {
	if r.Status.DeployHistory != nil {
		oldRuntime := &DeploymentRuntime{
			Spec: DeploymentRuntimeSpec{
				Product:        r.Spec.Product,
				ProjectsRef:    []string{},
				ManifestSource: r.Status.DeployHistory.ManifestSource,
				Destination:    r.Status.DeployHistory.Destination,
			},
		}

		manifest := oldRuntime.Spec.ManifestSource
		// Runtime destination cant not change after deployment.
		// If runtime has already deploy, it should not be block when other runtime use same config.
		if r.Spec.Destination != oldRuntime.Spec.Destination {
			return nil, fmt.Errorf("the deployed destination cannot be changed")
		} else if r.Spec.ManifestSource.CodeRepo == manifest.CodeRepo &&
			r.Spec.ManifestSource.Path == manifest.Path &&
			r.Spec.ManifestSource.TargetRevision == manifest.TargetRevision {
			return nil, nil
		}
	}

	runtimeList, err := validateClient.ListDeploymentRuntime(ctx, r.Namespace)
	if err != nil {
		return nil, fmt.Errorf("get deployment runtime list failed: %w", err)
	}

	for _, runtime := range runtimeList.Items {
		isDuplicate, err := r.Compare(&runtime)
		if err != nil {
			return nil, err
		}
		if isDuplicate && runtime.Name != r.Name {
			return nil, fmt.Errorf("can not deploy same repo to the same destination")
		}
	}

	return r.ValidateProjectRef(ctx, validateClient)
}

func (r *DeploymentRuntime) ValidateProjectRef(ctx context.Context, validateClient ValidateClient) ([]IllegalProjectRef, error) {
	illegalProjectRefs := []IllegalProjectRef{}
	codeRepo, err := validateClient.GetCodeRepo(ctx, r.Spec.ManifestSource.CodeRepo)
	if err != nil {
		return nil, err
	}

	for _, project := range r.Spec.ProjectsRef {
		err := hasCodeRepoPermission(ctx, validateClient, codeRepo.Spec.Product, project, codeRepo.Name)
		if err != nil {
			illegalProjectRefs = append(illegalProjectRefs, IllegalProjectRef{
				ProjectName: project,
				Reason:      fmt.Sprintf("project %s is illegal: %s", project, err.Error()),
			})
		}
	}
	return illegalProjectRefs, nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=get;list

func init() {
	GetClusterSubResourceFunctions = append(GetClusterSubResourceFunctions, GetDependentResourcesOfClusterFromDeploymentRuntime)
}

func GetDependentResourcesOfClusterFromDeploymentRuntime(ctx context.Context, k8sClient client.Client, clusterName string) ([]string, error) {
	runtimeList := &DeploymentRuntimeList{}

	if err := k8sClient.List(ctx, runtimeList); err != nil {
		return nil, err
	}

	dependencies := []string{}
	for _, runtime := range runtimeList.Items {
		if runtime.Status.Cluster == clusterName {
			dependencies = append(dependencies, fmt.Sprintf("deploymentRuntime/%s/%s", runtime.Namespace, runtime.Name))
		}
	}
	return dependencies, nil
}
