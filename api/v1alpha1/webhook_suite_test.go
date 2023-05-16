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
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	logr "github.com/go-logr/logr"
	. "github.com/nautes-labs/pkg/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var nautesNamespaceName = "nautes"

var mgr manager.Manager
var ctx context.Context
var cancel context.CancelFunc
var logger logr.Logger

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Webhookd Suite")

}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme
	err = AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	err = k8sClient.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nautesNamespaceName,
		},
	})
	Expect(err).NotTo(HaveOccurred())

	ctx, cancel = context.WithCancel(context.TODO())
	logger = logf.FromContext(ctx)
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Port:   9443,
	})
	Expect(err).NotTo(HaveOccurred())
	mgr.GetFieldIndexer().IndexField(context.Background(), &CodeRepo{}, SelectFieldCodeRepoName, func(obj client.Object) []string {
		logger.V(1).Info("add code repo index", "RepoName", obj.GetName(), "index", obj.GetName())
		return []string{obj.GetName()}
	})
	mgr.GetFieldIndexer().IndexField(context.Background(), &CodeRepoBinding{}, SelectFieldCodeRepoBindingProductAndRepo, func(obj client.Object) []string {
		binding := obj.(*CodeRepoBinding)
		logger.V(1).Info("add code repo binding index", "BindingName", binding.Name, "index", fmt.Sprintf("%s/%s", binding.Spec.Product, binding.Spec.CodeRepo))
		if binding.Spec.Product == "" || binding.Spec.CodeRepo == "" {
			return nil
		}

		return []string{fmt.Sprintf("%s/%s", binding.Spec.Product, binding.Spec.CodeRepo)}
	})

	KubernetesClient = mgr.GetClient()
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	k8sClientIsInit := false
	for i := 0; i < 10; i++ {
		podList := &corev1.PodList{}
		err := KubernetesClient.List(ctx, podList)
		if err == nil {
			k8sClientIsInit = true
			break
		}
		time.Sleep(time.Second)
	}
	Expect(k8sClientIsInit).Should(BeTrue())

	k8sClient = mgr.GetClient()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func randNum() string {
	return fmt.Sprintf("%04d", rand.Intn(9999))
}

func waitForDelete(obj client.Object) error {
	for i := 0; i < 10; i++ {
		err := k8sClient.Delete(context.Background(), obj)
		if apierrors.IsNotFound(err) {
			return nil
		}

		time.Sleep(time.Second)
	}
	return fmt.Errorf("wait for delete %s timeout", obj.GetName())
}
