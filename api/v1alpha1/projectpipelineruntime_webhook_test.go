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
	"time"

	. "github.com/nautes-labs/pkg/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("cluster webhook", func() {
	var runtime *ProjectPipelineRuntime
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

		runtime = &ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: ProjectPipelineRuntimeSpec{
				Project:        projectName,
				PipelineSource: source.Name,
				Pipelines:      []Pipeline{},
				Destination:    env.Name,
				EventSources: []EventSource{
					{
						Name: fmt.Sprintf("evName-%s", randNum()),
						Gitlab: &Gitlab{
							RepoName: eventRepo.Name,
							Revision: "main",
							Events:   []string{},
						},
						Calendar: &Calendar{},
					},
				},
				Isolation:        "",
				PipelineTriggers: []PipelineTrigger{},
			},
		}

		logger.V(1).Info("=====Case start=====")
		logger.V(1).Info("product", "Name", productName)
		logger.V(1).Info("project", "Name", projectName)
		logger.V(1).Info("souce repo", "Name", source.Name, "Project", source.Spec.Project)
		logger.V(1).Info("runtime", "Name", runtime.Name, "Project", runtime.Spec.Project)
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

		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(ctx, env)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(ctx, ns)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(ctx, cluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
	})

	It("if source and runtime in the same project, create will success", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.EventSources = nil
		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if source and runtime not in the same project, it need coderepo binding", func() {
		source.Spec.Project = fmt.Sprintf("%s-2", source.Spec.Project)
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.EventSources = nil
		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())

		codeRepoBinding.Spec.CodeRepo = source.Name
		codeRepoBinding.Spec.Projects = []string{runtime.Spec.Project}
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, source.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if product is same, and coderepobinding's projects is nil, runtime permission check should pass", func() {
		source.Spec.Project = fmt.Sprintf("%s-2", source.Spec.Project)
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		codeRepoBinding.Spec.CodeRepo = source.Name
		codeRepoBinding.Spec.Projects = nil
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, source.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if event source repo and runtime not in the same project, it need coderepo binding", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		eventRepo.Name = fmt.Sprintf("%s-2", eventRepo.Name)
		eventRepo.Spec.Project = fmt.Sprintf("%s-2", eventRepo.Spec.Project)
		err = k8sClient.Create(ctx, eventRepo)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, eventRepo.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.EventSources[0].Gitlab.RepoName = eventRepo.Name
		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())

		codeRepoBinding.Spec.CodeRepo = eventRepo.Name
		codeRepoBinding.Spec.Projects = []string{runtime.Spec.Project}
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, eventRepo.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})
})

func waitForIndexFieldUpdateCodeRepo(targetNum int, repoName string) error {
	obj := &CodeRepoList{}
	opt := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(SelectFieldCodeRepoName, repoName),
	}
	for i := 0; i < 3; i++ {
		err := k8sClient.List(ctx, obj, opt)
		if err != nil {
			return err
		}

		if targetNum == len(obj.Items) {
			return nil
		}
		time.Sleep(time.Millisecond * 500)
	}
	return fmt.Errorf("wait for index update timeout")
}

func waitForIndexFieldUpdateBinding(targetNum int, productName, repoName string) error {
	obj := &CodeRepoBindingList{}
	listVar := fmt.Sprintf("%s/%s", productName, repoName)
	opt := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(SelectFieldCodeRepoBindingProductAndRepo, listVar),
	}
	for i := 0; i < 3; i++ {
		err := k8sClient.List(ctx, obj, opt)
		if err != nil {
			return err
		}

		if targetNum == len(obj.Items) {
			return nil
		}
		time.Sleep(time.Millisecond * 500)
	}
	return fmt.Errorf("wait for index update timeout")
}
