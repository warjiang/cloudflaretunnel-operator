# Cloudflare Tunnel Operator

[English](README.md) | [中文](README.zh-CN.md)

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
