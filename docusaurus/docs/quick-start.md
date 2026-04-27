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

## 2. Create Cloudflare API Token

Create a custom token in Cloudflare:
`My Profile -> API Tokens -> Create Token -> Create Custom Token`.

Required permissions:

- One Account permission below (names vary by Cloudflare UI version):
  - `Cloudflare One Connectors Write`
  - `Cloudflare One Connector: cloudflared Write`
  - `Cloudflare Tunnel Write`
- Legacy name you may still see in old docs/UI: `Cloudflare Tunnel: Edit`
- `Zone -> DNS Write` (legacy label: `DNS: Edit`, required when using `ingress + hostname + zoneID`)

Recommended resource scope:

- `Account Resources -> Include -> Specific account -> <your-account>`
- `Zone Resources -> Include -> Specific zone -> <your-zone>` (or all managed zones)

Coverage checklist:

- Tunnel create/get/delete, token fetch, and tunnel config upsert use the tunnel connector write permission above.
- DNS CNAME create/update/delete uses `DNS Write`.

Common confusion (not the required DNS record permission here):

- `Cloudflare Zero Trust Secure DNS Locations Write`
- `Account DNS Settings`
- `DNS Firewall`
- `DNS View` (read-focused)

## 3. Create credentials and CloudflareTunnel

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

## 4. Verify reconciliation

```bash
kubectl get cloudflaretunnel my-tunnel -n default -o yaml
kubectl get secret my-tunnel-token -n default -o yaml
```

## 5. Expected behavior

- If the tunnel does not exist, the operator creates it.
- The tunnel token is synced to `spec.tokenSecretRef.name`.
- If `tokenSecretRef` is omitted, the default Secret name is `<metadata.name>-token`.
- On resource deletion, the operator removes the Cloudflare tunnel and then clears finalizer.
