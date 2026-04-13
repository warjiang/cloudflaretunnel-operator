# Cloudflare Tunnel Operator

English | [中文](#中文说明)

The Cloudflare Tunnel Operator manages Cloudflare Tunnels declaratively through a `CloudflareTunnel` CRD.

![Cloudflare Tunnel Operator Advantages](docusaurus/static/img/project-advantages.svg)

## Overview

This operator watches `CloudflareTunnel` resources and ensures:

- The target Cloudflare tunnel exists.
- A Kubernetes Secret containing the tunnel token is created/updated.
- Status (`tunnelID`, `observedGeneration`, and conditions) reflects current reconciliation state.

## Why This Project

- Secret-first credential handling (`credentialsRef`) avoids plaintext tokens in CR manifests.
- Automated tunnel token secret sync reduces manual operations and drift risk.
- Finalizer-based lifecycle management keeps remote Cloudflare resources and Kubernetes state consistent.
- Condition-driven status model improves observability and troubleshooting.
- Built with Kubebuilder/controller-runtime patterns for production-friendly reconciliation and testing.

## Credentials Model

Cloudflare credentials are read from a Secret referenced by `spec.credentialsRef`.

Required keys in the credentials Secret:

- `api-token`
- `account-id`

Credentials are no longer accepted directly in CRD spec fields.

## CRD Example

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cloudflare-credentials
  namespace: default
type: Opaque
stringData:
  api-token: <YOUR_CLOUDFLARE_API_TOKEN>
  account-id: <YOUR_CLOUDFLARE_ACCOUNT_ID>
---
apiVersion: cloudflaretunnel.spotty.com.cn/v1alpha1
kind: CloudflareTunnel
metadata:
  name: my-tunnel
  namespace: default
spec:
  name: my-first-tunnel
  credentialsRef:
    name: cloudflare-credentials
  tokenSecretRef:
    name: my-tunnel-token
```

## Reconcile Behavior

- If the tunnel does not exist, the operator creates it.
- The operator fetches tunnel token and syncs it to `spec.tokenSecretRef.name`.
- If `tokenSecretRef` is omitted, default Secret name is `<CloudflareTunnel.metadata.name>-token`.
- On delete, the operator removes the Cloudflare tunnel, then clears finalizer.

## Deploy

With Kustomize:

```bash
make docker-build IMG=<image:tag>
make docker-push IMG=<image:tag>
make deploy IMG=<image:tag>
```

For cross-platform image builds (multi-arch manifest), use:

```bash
make docker-buildx IMG=<image:tag>
```

Default image platforms are `linux/amd64,linux/arm64,linux/s390x,linux/ppc64le`.
You can override with `PLATFORMS=<platforms>`.

Apply your manifests:

```bash
kubectl apply -f demo.yaml
```

With Helm (CRD auto-install via `crds/`):

```bash
helm install cloudflaretunnel-operator \
  oci://ghcr.io/warjiang/charts/cloudflaretunnel-operator \
  --version <x.y.z> \
  --namespace cloudflaretunnel-operator-system \
  --create-namespace
```

Upgrade:

```bash
helm upgrade cloudflaretunnel-operator \
  oci://ghcr.io/warjiang/charts/cloudflaretunnel-operator \
  --version <x.y.z> \
  --namespace cloudflaretunnel-operator-system
```

Uninstall:

```bash
helm uninstall cloudflaretunnel-operator --namespace cloudflaretunnel-operator-system
```

Note: CRDs installed from Helm `crds/` are not deleted by `helm uninstall`.

## Testing

- Unit/controller tests: `make test`
- E2E tests: `make test-e2e`

## Cross-Platform Binaries

Build manager binaries for multiple platforms:

```bash
make build-cross
```

Artifacts are generated under `dist/<goos>-<goarch>/manager`.
Default binary platforms are `linux/amd64,linux/arm64,linux/s390x,linux/ppc64le,darwin/arm64`.
You can override with `BINARY_PLATFORMS=<platforms>`.

For local controller tests, install envtest assets first:

```bash
make setup-envtest
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 中文说明

Cloudflare Tunnel Operator 通过 `CloudflareTunnel` CRD 以声明式方式管理 Cloudflare Tunnel。

![Cloudflare Tunnel Operator 项目优势图](docusaurus/static/img/project-advantages.svg)

### 功能概览

该 Operator 会监听 `CloudflareTunnel`，并保证：

- 目标 Cloudflare Tunnel 存在。
- 自动创建/更新存放 Tunnel Token 的 Kubernetes Secret。
- 在状态中写入 `tunnelID`、`observedGeneration` 与条件（conditions）。

### 项目优势

- 凭据采用 `credentialsRef`（Secret 引用）模式，避免在 CR 清单中明文暴露 Token。
- Tunnel Token 自动同步到 Kubernetes Secret，减少手工操作和配置漂移。
- 基于 finalizer 的资源生命周期管理，确保云端资源与集群状态一致。
- 通过 conditions + `observedGeneration` 提供可观测、可排障的状态反馈。
- 基于 Kubebuilder/controller-runtime 标准模式，便于扩展、测试和工程化落地。

### 凭据模型

Cloudflare 凭据通过 `spec.credentialsRef` 引用的 Secret 提供。

凭据 Secret 必须包含以下 key：

- `api-token`
- `account-id`

CRD `spec` 中已不再支持明文凭据字段。

### CRD 示例

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cloudflare-credentials
  namespace: default
type: Opaque
stringData:
  api-token: <你的 Cloudflare API Token>
  account-id: <你的 Cloudflare Account ID>
---
apiVersion: cloudflaretunnel.spotty.com.cn/v1alpha1
kind: CloudflareTunnel
metadata:
  name: my-tunnel
  namespace: default
spec:
  name: my-first-tunnel
  credentialsRef:
    name: cloudflare-credentials
  tokenSecretRef:
    name: my-tunnel-token
```

### 对账行为

- 若 Tunnel 不存在，Operator 会自动创建。
- Operator 会获取 Tunnel Token 并同步到 `spec.tokenSecretRef.name` 指定的 Secret。
- 若未配置 `tokenSecretRef`，默认 Secret 名称为 `<CloudflareTunnel.metadata.name>-token`。
- 删除 CR 时，Operator 会先删除 Cloudflare Tunnel，再移除 finalizer。

### 部署

使用 Kustomize：

```bash
make docker-build IMG=<image:tag>
make docker-push IMG=<image:tag>
make deploy IMG=<image:tag>
```

如需构建跨平台镜像（multi-arch manifest），可执行：

```bash
make docker-buildx IMG=<image:tag>
```

默认镜像平台为 `linux/amd64,linux/arm64,linux/s390x,linux/ppc64le`。
可通过 `PLATFORMS=<platforms>` 覆盖。

应用资源：

```bash
kubectl apply -f demo.yaml
```

使用 Helm（CRD 通过 `crds/` 自动安装）：

```bash
helm install cloudflaretunnel-operator \
  oci://ghcr.io/warjiang/charts/cloudflaretunnel-operator \
  --version <x.y.z> \
  --namespace cloudflaretunnel-operator-system \
  --create-namespace
```

升级：

```bash
helm upgrade cloudflaretunnel-operator \
  oci://ghcr.io/warjiang/charts/cloudflaretunnel-operator \
  --version <x.y.z> \
  --namespace cloudflaretunnel-operator-system
```

卸载：

```bash
helm uninstall cloudflaretunnel-operator --namespace cloudflaretunnel-operator-system
```

说明：通过 Helm `crds/` 安装的 CRD 不会在 `helm uninstall` 时自动删除。

### 测试

- 单元/控制器测试：`make test`
- 端到端测试：`make test-e2e`

### 跨平台二进制构建

构建多平台 manager 二进制：

```bash
make build-cross
```

产物输出在 `dist/<goos>-<goarch>/manager`。
默认二进制平台为 `linux/amd64,linux/arm64,linux/s390x,linux/ppc64le,darwin/arm64`。
可通过 `BINARY_PLATFORMS=<platforms>` 覆盖。

本地跑控制器测试前建议先准备 envtest 依赖：

```bash
make setup-envtest
```
