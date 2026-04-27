# Cloudflare Tunnel Operator

[English](README.md) | [中文](README.zh-CN.md)

Cloudflare Tunnel Operator manages Cloudflare Tunnels declaratively through the `CloudflareTunnel` CRD.

![Cloudflare Tunnel Operator Advantages](docusaurus/static/img/project-advantages.svg)

## Quick Start

Prerequisites:

- Kubernetes cluster access (`kubectl` configured)
- A Cloudflare API Token with tunnel permissions
- Cloudflare Account ID

### Cloudflare API Token Setup (Zero Trust Tunnel)

Create a custom API Token in Cloudflare:
`My Profile -> API Tokens -> Create Token -> Create Custom Token`.

Recommended permissions and scope:

- Required for tunnel lifecycle:
  - One of the following Account permissions (names vary by Cloudflare UI version):
    - `Cloudflare One Connectors Write`
    - `Cloudflare One Connector: cloudflared Write`
    - `Cloudflare Tunnel Write`
  - Legacy name you may still see in older docs/UI: `Cloudflare Tunnel: Edit`
  - `Account Resources -> Include -> Specific account -> <your-account>`
- Required when using public hostname (`spec.ingress` + `spec.hostname` + `spec.zoneID`):
  - `Zone -> DNS Write` (legacy label: `DNS: Edit`)
  - `Zone Resources -> Include -> Specific zone -> <your-zone>` (or all managed zones)

Coverage map to operator behavior:

- Tunnel create/get/delete, token fetch, and remote ingress config upsert are covered by the tunnel connector write permission above.
- DNS CNAME create/update/delete for `<tunnel-id>.cfargotunnel.com` is covered by `DNS Write`.

Permission verification checklist:

- `CloudflareTunnel` reaches `TunnelReady=True`.
- If ingress is configured, `RoutingReady=True` and `DNSReady=True`.
- Deleting `CloudflareTunnel` removes remote tunnel and managed DNS record.

Common confusion (not the required DNS record permission for this operator):

- `Cloudflare Zero Trust Secure DNS Locations Write`
- `Account DNS Settings`
- `DNS Firewall`
- `DNS View` (read-focused)

For this operator's DNS record reconciliation, select `Zone -> DNS Write`.

Quick verification script:

```bash
./hack/verify-cloudflare-token-permissions.sh \
  --api-token "$API_TOKEN" \
  --account-id "$ACCOUNT_ID" \
  --zone-id "$ZONE_ID" \
  --hostname "permcheck.example.com" \
  --write-check \
  --dns-write-check
```

1. Create a credentials Secret:

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
```

2. Create a `CloudflareTunnel` resource:

```yaml
apiVersion: cloudflaretunnel.spotty.com.cn/v1alpha1
kind: CloudflareTunnel
metadata:
  name: my-tunnel
  namespace: default
spec:
  tunnelName: my-first-tunnel
  hostname: app.example.com
  zoneID: <YOUR_CLOUDFLARE_ZONE_ID>
  credentialsRef:
    name: cloudflare-credentials
  ingress:
    rules:
      - path: /api
        service:
          name: my-backend
          namespace: my-backend-ns
          port: 8080
  connector:
    image: cloudflare/cloudflared:2026.3.0
    replicas: 1
```

3. Apply your manifest:

```bash
kubectl apply -f demo.yaml
```

4. Verify tunnel status:

```bash
kubectl get cloudflaretunnel my-tunnel -n default -o yaml
```

If `spec.tokenSecretRef` is omitted, the operator stores token in `${metadata.name}-token` by default and reports the actual Secret name in `status.tokenSecretName`.

## What It Does

The operator watches `CloudflareTunnel` resources and ensures:

- The target Cloudflare Tunnel exists.
- Public hostname routing is configured in Cloudflare Tunnel and DNS CNAME points to `<tunnel-id>.cfargotunnel.com`.
- A Kubernetes Secret containing the tunnel token is created or updated.
- A dedicated cloudflared Deployment is created for each CloudflareTunnel.
- Optional ingress-style path forwarding rules are rendered into cloudflared config.
- Status (`tunnelID`, `tokenSecretName`, `observedGeneration`, conditions) reflects reconciliation state.
- Finalizers are used to clean up remote tunnel resources during deletion.

## Credentials Model

Cloudflare credentials are read from the Secret referenced by `spec.credentialsRef`.

Required Secret keys:

- `api-token`
- `account-id`

Credentials are not accepted directly in CRD spec fields.

## Installation

With Kustomize:

```bash
make docker-build IMG=<image:tag>
make docker-push IMG=<image:tag>
make deploy IMG=<image:tag>
```

For multi-arch image build and push:

```bash
make docker-buildx IMG=<image:tag>
```

Cross-platform binaries are built on the host first, then Docker only packages
`dist/linux-<arch>/manager` into runtime images.

Default image platforms:

`linux/amd64,linux/arm64,linux/s390x,linux/ppc64le`

Override with `PLATFORMS=<platforms>`.
For single-platform image builds, override with `DOCKER_PLATFORM=<platform>`.

With Helm (CRDs auto-installed from `crds/`):

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

## Development Commands

- Build manager binary: `make build`
- Build cross-platform binaries: `make build-cross`
- Run unit/controller tests: `make test`
- Run e2e tests: `make test-e2e`
- Run controller locally: `make run`
- Install envtest assets: `make setup-envtest`

Cross-platform binaries are written to `dist/<goos>-<goarch>/manager`.

Default binary platforms:

`linux/amd64,linux/arm64,linux/s390x,linux/ppc64le,darwin/arm64`

Override with `BINARY_PLATFORMS=<platforms>`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
