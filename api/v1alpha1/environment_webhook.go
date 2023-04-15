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
var environmentlog = logf.Log.WithName("environment-resource")

func (r *Environment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-environment,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=environments,verbs=create;update;delete,versions=v1alpha1,name=venvironment.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Environment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Environment) ValidateCreate() error {
	environmentlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Environment) ValidateUpdate(old runtime.Object) error {
	environmentlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Environment) ValidateDelete() error {
	environmentlog.Info("validate delete", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}
	return r.IsDeletable(k8sClient)
}

func (r *Environment) IsDeletable(k8sClient client.Client) error {
	deploymentRuntimes := &DeploymentRuntimeList{}
	listOpts := client.ListOptions{
		Namespace: r.Namespace,
	}
	err := k8sClient.List(context.Background(), deploymentRuntimes, &listOpts)
	if err != nil {
		return err
	}

	for _, deploymentRuntime := range deploymentRuntimes.Items {
		if deploymentRuntime.Spec.Destination == r.Name {
			return fmt.Errorf("environment is referenced by runtime")
		}
	}

	return nil
}
