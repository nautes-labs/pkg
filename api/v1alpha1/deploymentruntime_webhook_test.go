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

package v1alpha1_test

import (
	"context"
	"fmt"

	. "github.com/nautes-labs/pkg/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("cluster webhook", func() {
	var runtime *DeploymentRuntime
	var env *Environment
	var cluster *Cluster
	var ctx context.Context
	var productName string
	var projectName string
	var ns *corev1.Namespace
	var source *CodeRepo
	var eventRepo *CodeRepo
	var codeRepoBinding *CodeRepoBinding
	BeforeEach(func() {
		ctx = context.Background()
		productName = fmt.Sprintf("product-%s", randNum())
		projectName = fmt.Sprintf("project-%s", randNum())
		cluster = &Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", randNum()),
				Namespace: nautesNamespaceName,
			},
			Spec: ClusterSpec{
				ApiServer:   "",
				ClusterType: CLUSTER_TYPE_PHYSICAL,
				ClusterKind: CLUSTER_KIND_KUBERNETES,
				Usage:       CLUSTER_USAGE_WORKER,
				HostCluster: "",
			},
		}

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).Should(BeNil())

		env = &Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("env-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: EnvironmentSpec{
				Product: productName,
				Cluster: cluster.Name,
				EnvType: "test",
			},
		}

		source = &CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("repo-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: CodeRepoSpec{
				CodeRepoProvider:  "",
				Product:           productName,
				Project:           projectName,
				RepoName:          "",
				URL:               "",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook:           nil,
			},
		}

		eventRepo = source.DeepCopyObject().(*CodeRepo)

		codeRepoBinding = &CodeRepoBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("binding-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: CodeRepoBindingSpec{
				CodeRepo:    eventRepo.Name,
				Product:     productName,
				Projects:    []string{eventRepo.Spec.Project},
				Permissions: "",
			},
		}

		runtime = &DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: DeploymentRuntimeSpec{
				Product:     productName,
				ProjectsRef: []string{projectName},
				ManifestSource: ManifestSource{
					CodeRepo:       source.Name,
					TargetRevision: "HEAD",
					Path:           "/",
				},
				Destination: env.Name,
			},
		}
		logger.V(1).Info("=====Case start=====")
		logger.V(1).Info("product", "Name", productName)
		logger.V(1).Info("project", "Name", projectName)
		logger.V(1).Info("souce repo", "Name", source.Name, "Project", source.Spec.Project)
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, runtime)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(ctx, codeRepoBinding)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		err = k8sClient.Delete(ctx, source)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = waitForDelete(source)
		Expect(err).Should(BeNil())

		err = k8sClient.Delete(ctx, eventRepo)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = waitForDelete(eventRepo)
		Expect(err).Should(BeNil())

		err = k8sClient.Delete(ctx, env)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(ctx, ns)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(ctx, cluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
	})
	It("if project has permission to use coderepo, create will successed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if project has no permission to use coderepo , create will failed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.ProjectsRef = []string{"fake"}

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when an identical runtime has already been deployed, create will failed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime2 := runtime.DeepCopyObject().(*DeploymentRuntime)
		runtime2.Name = fmt.Sprintf("%s-2", runtime.Name)
		err = k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())

		err = runtime2.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when an identical runtime has already been deployed, the deployed runtime should pass validate", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime2 := runtime.DeepCopyObject().(*DeploymentRuntime)
		runtime2.Name = fmt.Sprintf("%s-2", runtime.Name)
		runtime2.Status.DeployHistory = &DeployHistory{
			ManifestSource: runtime2.Spec.ManifestSource,
			Destination:    runtime2.Spec.Destination,
		}
		err = k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())

		err = runtime2.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("when an identical runtime has already been deployed, object meta change should be ignore ", func() {
		runtime.Spec.ProjectsRef = []string{"fake"}

		runtime2 := runtime.DeepCopyObject().(*DeploymentRuntime)
		runtime2.Finalizers = []string{"one two"}

		err := runtime.ValidateUpdate(runtime2)
		Expect(err).Should(BeNil())
	})
})
