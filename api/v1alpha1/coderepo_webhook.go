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

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes;products;coderepoproviders,verbs=get;list

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var coderepolog = logf.Log.WithName("coderepo-resource")
var manager ctrl.Manager

func (r *CodeRepo) SetupWebhookWithManager(mgr ctrl.Manager) error {
	manager = mgr
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &CodeRepo{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CodeRepo) Default() {

	coderepolog.Info("default", "name", r.Name, "url", r.Spec.URL)
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-coderepo,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=coderepoes,verbs=create;update;delete,versions=v1alpha1,name=vcoderepo.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CodeRepo{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CodeRepo) ValidateCreate() error {
	coderepolog.Info("validate create", "name", r.Name)

	err := r.Validate()
	if err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CodeRepo) ValidateUpdate(old runtime.Object) error {
	coderepolog.Info("validate update", "name", r.Name)

	err := r.Validate()
	if err != nil {
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CodeRepo) ValidateDelete() error {
	coderepolog.Info("validate delete", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}
	return r.IsDeletable(k8sClient)
}

func (r *CodeRepo) GetURL(spec CodeRepoSpec) (string, error) {
	if spec.URL != "" {
		return spec.URL, nil
	} else if spec.URL == "" && spec.Product != "" && spec.CodeRepoProvider != "" {
		client, err := getClient()
		if err != nil {
			return "", err
		}

		configs, err := nautesconfigs.NewConfigInstanceForK8s("nautes", "nautes-configs", "")
		if err != nil {
			return "", nil
		}

		product := &Product{}
		err = client.Get(context.Background(), types.NamespacedName{
			Namespace: configs.Nautes.Namespace,
			Name:      r.Spec.Product,
		}, product)
		if err != nil {
			return "", err
		}

		provider := &CodeRepoProvider{}
		err = client.Get(context.Background(), types.NamespacedName{
			Namespace: configs.Nautes.Namespace,
			Name:      r.Spec.CodeRepoProvider,
		}, provider)
		if err != nil {
			return "", err
		}

		url := fmt.Sprintf("%v/%v/%v.git", provider.Spec.SSHAddress, product.Spec.Name, spec.RepoName)
		return url, nil
	} else {
		return "", errors.New("the resource is not avaiable. if the url does not exist, it should contain coderepo provider and product resources")
	}
}

func (r *CodeRepo) Validate() error {
	url, err := r.GetURL(r.Spec)
	if err != nil {
		return err
	}

	matched, err := regexp.MatchString(`^(ssh:\/\/)?git@(?:((?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}|[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(?:\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+):(?:(\d{1,5})\/)?([\w-]+)\/([\w.-]+)\.git$`, url)
	if err != nil {
		return err
	}

	if !matched {
		return fmt.Errorf("the url %v is illegal", url)
	}

	return nil
}

func (r *CodeRepo) IsDeletable(k8sClient client.Client) error {
	deploymentRuntimes := &DeploymentRuntimeList{}
	listOpts := &client.ListOptions{
		Namespace: r.Namespace,
	}
	if err := k8sClient.List(context.Background(), deploymentRuntimes, listOpts); err != nil {
		return err
	}

	for _, deploymentRuntime := range deploymentRuntimes.Items {
		if deploymentRuntime.Spec.ManifestSource.CodeRepo == r.Name {
			return fmt.Errorf("code repo is referenced by runtime")
		}
	}
	return nil
}
