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

## 2. 创建凭据与 CloudflareTunnel

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

## 3. 验证对账结果

```bash
kubectl get cloudflaretunnel my-tunnel -n default -o yaml
kubectl get secret my-tunnel-token -n default -o yaml
```

## 4. 预期行为

- 若 Tunnel 不存在，Operator 会自动创建。
- Tunnel Token 会同步到 `spec.tokenSecretRef.name` 指定的 Secret。
- 若未配置 `tokenSecretRef`，默认 Secret 名称为 `<metadata.name>-token`。
- 删除资源时，Operator 会先删除 Cloudflare Tunnel，再移除 finalizer。
