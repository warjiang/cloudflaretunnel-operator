---
id: intro
title: Introduction
slug: /
---

`cloudflaretunnel-operator` is a Kubernetes operator for managing Cloudflare Tunnel resources declaratively.

![Cloudflare Tunnel Operator Advantages](/img/project-advantages.svg)

## What it does

- Creates a Cloudflare tunnel when it does not exist.
- Syncs tunnel token to a Kubernetes Secret.
- Updates `status.tunnelID`, `status.observedGeneration`, and conditions.

## Why this project

- Secret-first credential handling via `credentialsRef`.
- Automated tunnel token secret sync for operational simplicity.
- Finalizer-based cleanup to keep Cloudflare and Kubernetes resources consistent.
- Conditions + `observedGeneration` for better observability and diagnostics.

## Core CRD fields

```yaml
spec:
  name: my-first-tunnel
  credentialsRef:
    name: cloudflare-credentials
  tokenSecretRef:
    name: my-tunnel-token
```

Credentials Secret requires:

- `api-token`
- `account-id`
