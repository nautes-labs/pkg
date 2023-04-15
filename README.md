# Package
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![golang](https://img.shields.io/badge/golang-v1.17.13-brightgreen)](https://go.dev/doc/install)
[![version](https://img.shields.io/badge/version-v0.2.0-green)]()

PKG 项目主要是用于存放 Nautes 中可重用的 CRD(Custom Resource Definition) 和 CRD Webhook 的源文件，这些 CRD 会被 Nautes 中多个组件依赖，除此之外，PKG 还包含一些可以被其他组件使用的公共库：

- 全局配置

- Logger

- Kubeconfig 序列化

## 功能简介

PKG 中的 CRD Webhook 主要是提供校验资源有效性的功能，包括：

| 资源类型 | 校验逻辑 |
| --- | ---|
| Cluster | 有虚拟集群的宿主集群无法被删除<br>有运行时环境的运行时集群无法被删除<br>有运行时环境的运行时集群无法变更用途<br>有虚拟集群的宿主集群无法变更用途<br>宿主集群必须是物理集群<br>虚拟集群的 hostCluster 为必填项<br>物理集群的 hostCluster 必须为空 |
| Deployment Runtime | 已经部署过的运行时不可以切换目标环境<br>不允许在同一个环境中创建两个指向相同源的部署运行时 |
| Environment | 被运行时引用的环境无法被删除 |
| Product Provider | 被产品引用的产品提供者无法被删除 |
| Code Repo | 代码仓库地址是否存在且有效 |

## 快速开始

### 准备

安装以下工具，并配置 GOBIN 环境变量：

- [go](https://golang.org/dl/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

准备一个 kubernetes 实例，复制 kubeconfig 文件到 {$HOME}/.kube/config

### 构建

```shell
go mod tidy -go=1.16 && go mod tidy -go=1.17
go build -o manager main.go
```

### 运行

```shell
./manager
```

### 单元测试

安装 Envtest

```shell
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
setup-envtest use 1.21.x
export KUBEBUILDER_ASSETS=$HOME/.local/share/kubebuilder-envtest/k8s/1.21.4-linux-amd64
```

安装 Ginkgo

```shell
go install github.com/onsi/ginkgo/v2/ginkgo@v2.3.1
```

执行单元测试

```shell
ginkgo -r
```

