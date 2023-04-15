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

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters,verbs=get;list
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=get;list
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=environments,verbs=get;list

//+kubebuilder:webhook:path=/mutate-nautes-resource-nautes-io-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=clusters,verbs=create,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Cluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Cluster) Default() {
	clusterlog.Info("default", "name", r.Name)

}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=clusters,verbs=create;update;delete,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Cluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateCreate() error {
	clusterlog.Info("validate create", "name", r.Name)

	return ValidateCluster(r, nil, false)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateUpdate(old runtime.Object) error {
	clusterlog.Info("validate update", "name", r.Name)

	return ValidateCluster(r, old.(*Cluster), false)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Cluster) ValidateDelete() error {
	clusterlog.Info("validate delete", "name", r.Name)

	return ValidateCluster(r, nil, true)
}

// ValidateCluster check cluster is changeable
func ValidateCluster(new, old *Cluster, isDelete bool) error {
	if isDelete {
		if new.Spec.Usage == CLUSTER_USAGE_HOST && new.hasVirtualCluster() {
			return fmt.Errorf("cluster still has virtual cluster on it, it can not be remove")
		}

		hasRuntime, err := new.hasRuntime()
		if err != nil {
			return fmt.Errorf("check has runtime failed: %w", err)
		}
		if new.Spec.Usage == CLUSTER_USAGE_WORKER && hasRuntime {
			return fmt.Errorf("cluster still has user resources on it, it can not be remove")
		}

		return nil
	}

	if err := new.validateBasic(); err != nil {
		return err
	}

	if old != nil {
		if new.Spec.Usage != old.Spec.Usage {
			hasRuntime, err := new.hasRuntime()
			if err != nil {
				return fmt.Errorf("check has runtime failed: %w", err)
			}
			if new.Spec.Usage == CLUSTER_USAGE_HOST && hasRuntime {
				return fmt.Errorf("cluster has runtime, change usage is not allow")
			}

			if new.Spec.Usage == CLUSTER_USAGE_WORKER && new.hasVirtualCluster() {
				return fmt.Errorf("cluster has virtual cluster on it, change usage is not allow")
			}
		}
	}

	return nil
}

func (r *Cluster) validateBasic() error {
	if r.Spec.Usage == CLUSTER_USAGE_HOST &&
		r.Spec.ClusterType != CLUSTER_TYPE_PHYSICAL {
		return errors.New("host cluster can not be a virautl cluster")
	}

	if r.Spec.ClusterType == CLUSTER_TYPE_PHYSICAL &&
		r.Spec.HostCluster != "" {
		return errors.New("host cluster can not belong to another host")
	}

	if r.Spec.ClusterType == CLUSTER_TYPE_VIRTUAL &&
		r.Spec.HostCluster == "" {
		return errors.New("virtual cluster must have a host")
	}

	return nil
}

func (r *Cluster) hasVirtualCluster() bool {
	clusters, err := getClusterList()
	if err != nil {
		return true
	}

	for i := 0; i < len(clusters.Items); i++ {
		if clusters.Items[i].Spec.HostCluster == r.Name {
			return true
		}
	}
	return false
}

func (r *Cluster) hasRuntime() (bool, error) {
	k8sClient, err := getClient()
	if err != nil {
		return false, err
	}

	runtimes := &DeploymentRuntimeList{}
	err = k8sClient.List(context.TODO(), runtimes)
	if err != nil {
		return false, err
	}

	for _, runtime := range runtimes.Items {
		env := &Environment{}
		key := types.NamespacedName{
			Namespace: runtime.Namespace,
			Name:      runtime.Spec.Destination,
		}
		err := k8sClient.Get(context.TODO(), key, env)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return false, err
		}

		if env.Spec.Cluster == r.Name {
			return true, nil
		}
	}

	return false, nil
}

func getClusterList() (*ClusterList, error) {
	k8sClient, err := getClient()
	if err != nil {
		return nil, err
	}
	clusterList := &ClusterList{}
	if err := k8sClient.List(context.TODO(), clusterList, &client.ListOptions{}); err != nil {
		clusterlog.Error(err, "get cluster list failed")
		return nil, err
	}
	return clusterList, nil
}
