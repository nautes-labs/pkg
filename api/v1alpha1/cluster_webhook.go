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
	"strings"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clusterlog = logf.Log.WithName("cluster-resource")

func (r *Cluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-nautes-resource-nautes-io-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=clusters,verbs=create,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Cluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Cluster) Default() {
	clusterlog.Info("default", "name", r.Name)

}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=clusters,verbs=create;update;delete,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters,verbs=get;list

var _ webhook.Validator = &Cluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateCreate() error {
	clusterlog.Info("validate create", "name", r.Name)

	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	return r.ValidateCluster(context.TODO(), nil, k8sClient, false)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateUpdate(old runtime.Object) error {
	clusterlog.Info("validate update", "name", r.Name)

	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	return r.ValidateCluster(context.TODO(), old.(*Cluster), k8sClient, false)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateDelete() error {
	clusterlog.Info("validate delete", "name", r.Name)

	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	return r.ValidateCluster(context.TODO(), nil, k8sClient, true)
}

// ValidateCluster check cluster is changeable
func (r *Cluster) ValidateCluster(ctx context.Context, old *Cluster, k8sClient client.Client, isDelete bool) error {
	if isDelete {
		dependencies, err := r.GetDependencies(ctx, k8sClient)
		if err != nil {
			return err
		}
		if len(dependencies) != 0 {
			return fmt.Errorf("cluster referenced by [%s], not allowed to be deleted", strings.Join(dependencies, "|"))
		}
		return nil
	}

	if err := r.staticCheck(); err != nil {
		return err
	}

	if old != nil {
		dependencies, err := r.GetDependencies(ctx, k8sClient)
		if err != nil {
			return err
		}
		if len(dependencies) == 0 {
			return nil
		}

		if r.Spec.ClusterKind != old.Spec.ClusterKind ||
			r.Spec.ClusterType != old.Spec.ClusterType ||
			r.Spec.Usage != old.Spec.Usage ||
			r.Spec.HostCluster != old.Spec.HostCluster ||
			r.Spec.WorkerType != old.Spec.WorkerType {
			return fmt.Errorf("cluster referenced by [%s], modifying [cluster kind, cluster type, usage, host cluster, worker type] is not allowed", strings.Join(dependencies, "|"))
		}
	}

	return nil
}

func (r *Cluster) staticCheck() error {
	if r.Spec.Usage == CLUSTER_USAGE_HOST {
		if r.Spec.ClusterType != CLUSTER_TYPE_PHYSICAL {
			return errors.New("host cluster can not be a virautl cluster")
		}

		if r.Spec.WorkerType != "" {
			return fmt.Errorf("host cluster's work type should be empty")
		}
	}

	if r.Spec.ClusterType == CLUSTER_TYPE_PHYSICAL &&
		r.Spec.HostCluster != "" {
		return errors.New("host cluster can not belong to another host")
	}

	if r.Spec.ClusterType == CLUSTER_TYPE_VIRTUAL &&
		r.Spec.HostCluster == "" {
		return errors.New("virtual cluster must have a host")
	}

	reservedNamespace := r.Spec.ComponentsList.GetNamespacesMap()

	for namespace := range r.Spec.ReservedNamespacesAllowedProducts {
		if !reservedNamespace[namespace] {
			return fmt.Errorf("namespace %s is not in compnent list", namespace)
		}
	}

	return nil
}

// +kubebuilder:object:generate=false
type GetClusterSubResources func(ctx context.Context, k8sClient client.Client, clusterName string) ([]string, error)

// GetClusterSubResourceFunctions stores a set of methods for obtaining a list of cluster sub-resources.
// When the cluster checks whether it is being referenced, it will loop through the method list here.
var GetClusterSubResourceFunctions = []GetClusterSubResources{}

func (r *Cluster) GetDependencies(ctx context.Context, k8sClient client.Client) ([]string, error) {
	subResources := []string{}
	for _, fn := range GetClusterSubResourceFunctions {
		resources, err := fn(ctx, k8sClient, r.Name)
		if err != nil {
			return nil, fmt.Errorf("get dependent resources failed: %w", err)
		}
		subResources = append(subResources, resources...)
	}

	return subResources, nil
}

func init() {
	GetClusterSubResourceFunctions = append(GetClusterSubResourceFunctions, getDependentResourcesOfClusterFromCluster)
}

func getDependentResourcesOfClusterFromCluster(ctx context.Context, k8sClient client.Client, clusterName string) ([]string, error) {
	clusterList := &ClusterList{}
	if err := k8sClient.List(ctx, clusterList); err != nil {
		return nil, err
	}

	dependencies := []string{}
	for _, cluster := range clusterList.Items {
		if cluster.Spec.HostCluster == clusterName {
			dependencies = append(dependencies, fmt.Sprintf("cluster/%s", cluster.Name))
		}
	}
	return dependencies, nil
}
