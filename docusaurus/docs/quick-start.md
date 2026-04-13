---
id: quick-start
title: Quick Start
---

## 1. Deploy operator

```bash
make docker-build IMG=<image:tag>
make docker-push IMG=<image:tag>
make deploy IMG=<image:tag>
```

## 2. Create credentials and CR

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

Apply:

```bash
kubectl apply -f demo.yaml
```

## 3. Verify

```bash
kubectl get cloudflaretunnels
kubectl get secret my-tunnel-token -o yaml
```
