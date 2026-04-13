---
id: intro
title: 介绍
slug: /
---

`cloudflaretunnel-operator` 是一个用于声明式管理 Cloudflare Tunnel 的 Kubernetes Operator。

![Cloudflare Tunnel Operator 项目优势图](/img/project-advantages.svg)

## 前置条件

- 可访问的 Kubernetes 集群（已配置 `kubectl`）
- 具备 Tunnel 权限的 Cloudflare API Token
- Cloudflare Account ID

## 功能

- 当 Tunnel 不存在时自动创建。
- 将 Tunnel Token 同步到 Kubernetes Secret。
- 在 `status.tunnelID`、`status.observedGeneration` 和 conditions 中回写状态。

## 项目优势

- 通过 `credentialsRef` 采用 Secret 优先的凭据模型。
- 自动同步 Tunnel Token，减少人工操作与配置漂移。
- 通过 finalizer 保障云端与集群资源生命周期一致。
- 通过 conditions 和 `observedGeneration` 提升可观测性和排障效率。

## 核心字段

```yaml
spec:
  name: my-first-tunnel
  credentialsRef:
    name: cloudflare-credentials
  tokenSecretRef:
    name: my-tunnel-token
```

凭据 Secret 必须包含：

- `api-token`
- `account-id`

CRD `spec` 不支持直接填写明文凭据。
