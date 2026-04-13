---
id: quick-start
title: Quick Start
---

## 1. Deploy the operator

```bash
make docker-build IMG=<image:tag>
make docker-push IMG=<image:tag>
make deploy IMG=<image:tag>
```

## 2. Create credentials and CloudflareTunnel

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

## 3. Verify reconciliation

```bash
kubectl get cloudflaretunnel my-tunnel -n default -o yaml
kubectl get secret my-tunnel-token -n default -o yaml
```

## 4. Expected behavior

- If the tunnel does not exist, the operator creates it.
- The tunnel token is synced to `spec.tokenSecretRef.name`.
- If `tokenSecretRef` is omitted, the default Secret name is `<metadata.name>-token`.
- On resource deletion, the operator removes the Cloudflare tunnel and then clears finalizer.
