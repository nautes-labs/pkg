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
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	LABEL_FROM_PRODUCT          = "resource.nautes.io/reference"
	LABEL_BELONG_TO_PRODUCT     = "resource.nautes.io/belongsto"
	LABEL_FROM_PRODUCT_PROVIDER = "resource.nautes.io/from"
)

const (
	EnvUseRealName = "USEREALNAME"
)

var (
	// KubernetesClient create from webhook main.go, it use to validate resource is legal
	KubernetesClient client.Client
)

func GetConditions(conditions []metav1.Condition, conditionTypes map[string]bool) []metav1.Condition {
	result := make([]metav1.Condition, 0)
	for i := range conditions {
		condition := conditions[i]
		if ok := conditionTypes[condition.Type]; ok {
			result = append(result, condition)
		}

	}
	return result
}

func GetNewConditions(src, conditions []metav1.Condition, evaluatedTypes map[string]bool) []metav1.Condition {
	appConditions := make([]metav1.Condition, 0)
	for i := 0; i < len(src); i++ {
		condition := src[i]
		if _, ok := evaluatedTypes[condition.Type]; !ok {
			appConditions = append(appConditions, condition)
		}
	}
	for i := range conditions {
		condition := conditions[i]
		eci := findConditionIndexByType(src, condition.Type)
		if eci >= 0 &&
			src[eci].Message == condition.Message &&
			src[eci].Status == condition.Status &&
			src[eci].Reason == condition.Reason {
			// If we already have a condition of this type, only update the timestamp if something
			// has changed.
			src[eci].LastTransitionTime = metav1.Now()
			appConditions = append(appConditions, src[eci])
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
	return appConditions
}

func findConditionIndexByType(conditions []metav1.Condition, t string) int {
	for i := range conditions {
		if conditions[i].Type == t {
			return i
		}
	}
	return -1
}

func getClient() (client.Client, error) {
	if KubernetesClient == nil {
		return nil, fmt.Errorf("kubernetes client is not initializated")
	}
	return KubernetesClient, nil
}

// +kubebuilder:object:generate=false
type ValidateClient interface {
	GetCodeRepo(ctx context.Context, name string) (*CodeRepo, error)
	GetEnvironment(ctx context.Context, productName, name string) (*Environment, error)
	GetCluster(ctx context.Context, name string) (*Cluster, error)
	ListCodeRepoBindings(ctx context.Context, productName, repoName string) ([]CodeRepoBinding, error)
	// ListDeploymentRuntime will return deployment runtimes in specified product. If product is empty, it will return all deployment runtimes.
	ListDeploymentRuntimes(ctx context.Context, productName string) ([]DeploymentRuntime, error)
	ListProjectPipelineRuntimes(ctx context.Context, productName string) ([]ProjectPipelineRuntime, error)
}

// ValidateClientFromK8s is the k8s implementation of interface ValidateClient.
// It's creation requires an implementation of client.Client.
// This implementation requires adding the following indexes:
//
//	metadata.name in resource Cluster.
//	metadata.name in resource CodeRepo.
//	productAndRepo in resource CodeRepoBinding. The format of the index value should be like "Product/CodeRepo".
//
// +kubebuilder:object:generate=false
type ValidateClientFromK8s struct {
	client.Client
}

func NewValidateClientFromK8s(client client.Client) ValidateClient {
	return &ValidateClientFromK8s{Client: client}
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes,verbs=get;list

func (c *ValidateClientFromK8s) GetCodeRepo(ctx context.Context, name string) (*CodeRepo, error) {
	codeRepoList := &CodeRepoList{}
	listOpt := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(SelectFieldMetaDataName, name),
	}
	if err := c.List(ctx, codeRepoList, listOpt); err != nil {
		projectpipelineruntimelog.V(1).Info("grep code repo", "MatchNum", len(codeRepoList.Items))
		return nil, err
	}

	count := len(codeRepoList.Items)
	if count != 1 {
		return nil, fmt.Errorf("returned %d results based on coderepo name %s", count, name)
	}
	return &codeRepoList.Items[0], nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=environments,verbs=get;list

func (c *ValidateClientFromK8s) GetEnvironment(ctx context.Context, productName, name string) (*Environment, error) {
	env := &Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: productName,
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(env), env); err != nil {
		return nil, err
	}

	return env, nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters,verbs=get;list

func (c *ValidateClientFromK8s) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	clusterList := &ClusterList{}
	if err := c.List(ctx, clusterList, client.MatchingFields{"metadata.name": name}); err != nil {
		return nil, err
	}

	count := len(clusterList.Items)
	if count != 1 {
		return nil, fmt.Errorf("returned %d results based on cluster name %s", count, name)
	}

	return &clusterList.Items[0], nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepobindings,verbs=get;list

func (c *ValidateClientFromK8s) ListCodeRepoBindings(ctx context.Context, productName, repoName string) ([]CodeRepoBinding, error) {
	logger := logf.FromContext(ctx)

	codeRepoBindingList := &CodeRepoBindingList{}
	listVar := fmt.Sprintf("%s/%s", productName, repoName)
	listOpt := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(SelectFieldCodeRepoBindingProductAndRepo, listVar),
	}
	if err := c.List(ctx, codeRepoBindingList, listOpt); err != nil {
		return nil, err
	}

	logger.V(1).Info("grep code repo binding", "ListVar", listVar, "MatchNum", len(codeRepoBindingList.Items))
	return codeRepoBindingList.Items, nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=get;list

func (c *ValidateClientFromK8s) ListDeploymentRuntimes(ctx context.Context, productName string) ([]DeploymentRuntime, error) {
	runtimeList := &DeploymentRuntimeList{}
	listOpts := []client.ListOption{}
	if productName != "" {
		listOpts = append(listOpts, client.InNamespace(productName))
	}

	if err := c.List(ctx, runtimeList, listOpts...); err != nil {
		return nil, err
	}
	return runtimeList.Items, nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=get;list

func (c *ValidateClientFromK8s) ListProjectPipelineRuntimes(ctx context.Context, productName string) ([]ProjectPipelineRuntime, error) {
	runtimeList := &ProjectPipelineRuntimeList{}
	listOpts := []client.ListOption{}
	if productName != "" {
		listOpts = append(listOpts, client.InNamespace(productName))
	}
	if err := c.List(ctx, runtimeList, listOpts...); err != nil {
		return nil, err
	}
	return runtimeList.Items, nil
}

func hasCodeRepoPermission(ctx context.Context, validateClient ValidateClient, productName, projectName, repoName string) error {
	codeRepo, err := validateClient.GetCodeRepo(ctx, repoName)
	if err != nil {
		return err
	}
	if codeRepo.DeletionTimestamp.IsZero() &&
		codeRepo.Spec.Product == productName &&
		codeRepo.Spec.Project == projectName {
		return nil
	}

	codeRepoBindings, err := validateClient.ListCodeRepoBindings(ctx, productName, repoName)
	if err != nil {
		return err
	}
	for _, binding := range codeRepoBindings {
		if !binding.DeletionTimestamp.IsZero() {
			continue
		}

		// if projects is nil or empty, means the scope of permission is at the product level
		if binding.Spec.Projects == nil || len(binding.Spec.Projects) == 0 {
			return nil
		}

		for _, project := range binding.Spec.Projects {
			if project == projectName {
				return nil
			}
		}
	}

	return fmt.Errorf("not permitted to use code repo %s", getCodeRepoName(codeRepo))
}

// +kubebuilder:object:generate=false
type Runtime interface {
	GetProduct() string
	GetName() string
	GetDestination() string
	GetNamespaces() []string
}

func GetClusterByRuntime(ctx context.Context, client ValidateClient, runtime Runtime) (*Cluster, error) {
	env, err := client.GetEnvironment(ctx, runtime.GetProduct(), runtime.GetDestination())
	if err != nil {
		return nil, err
	}
	return client.GetCluster(ctx, env.Spec.Cluster)
}

func convertArrayToBoolMap(in []string) map[string]bool {
	out := map[string]bool{}
	for _, str := range in {
		out[str] = true
	}
	return out
}
