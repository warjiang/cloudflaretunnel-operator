---
id: quick-start
title: 快速开始
---

## 1. 部署 Operator

```bash
make docker-build IMG=<image:tag>
make docker-push IMG=<image:tag>
make deploy IMG=<image:tag>
```

## 2. 申请 Cloudflare API Token

在 Cloudflare 控制台创建自定义 Token：
`My Profile -> API Tokens -> Create Token -> Create Custom Token`。

必需权限：

- 以下 Account 权限三选一（Cloudflare 不同版本 UI 名称不同）：
  - `Cloudflare One Connectors Write`
  - `Cloudflare One Connector: cloudflared Write`
  - `Cloudflare Tunnel Write`
- 旧版文档/UI 可能显示为：`Cloudflare Tunnel: Edit`
- `Zone -> DNS Write`（旧标签可能显示为 `DNS: Edit`，仅在使用 `ingress + hostname + zoneID` 时必需）

推荐资源范围：

- `Account Resources -> Include -> Specific account -> <your-account>`
- `Zone Resources -> Include -> Specific zone -> <your-zone>`（或纳管的全部 zones）

覆盖能力检查：

- Tunnel 创建/查询/删除、Token 获取、Tunnel 配置更新使用上述 Tunnel/Connector 写权限。
- DNS CNAME 创建/更新/删除使用 `DNS Write`。

## 3. 创建凭据与 CloudflareTunnel

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

应用：

```bash
kubectl apply -f demo.yaml
```

## 4. 验证对账结果

```bash
kubectl get cloudflaretunnel my-tunnel -n default -o yaml
kubectl get secret my-tunnel-token -n default -o yaml
```

## 5. 预期行为

- 若 Tunnel 不存在，Operator 会自动创建。
- Tunnel Token 会同步到 `spec.tokenSecretRef.name` 指定的 Secret。
- 若未配置 `tokenSecretRef`，默认 Secret 名称为 `<metadata.name>-token`。
- 删除资源时，Operator 会先删除 Cloudflare Tunnel，再移除 finalizer。
