# Cloudflare Tunnel Operator

The Cloudflare Tunnel Operator is a Kubernetes operator that manages Cloudflare Tunnels declaratively using Custom Resource Definitions (CRDs).

## Overview

This operator introduces a `CloudflareTunnel` custom resource that allows you to create, update, and delete Cloudflare Tunnels by simply applying a manifest to your Kubernetes cluster. The operator will automatically handle the interaction with the Cloudflare API to ensure your tunnels are in the desired state.

## Prerequisites

*   A Kubernetes cluster (v1.20+)
*   `kubectl` installed and configured
*   A Cloudflare account with an API token and account ID

## Deployment

To deploy the operator to your Kubernetes cluster, you will need to have your Cloudflare API token and account ID available.

1.  **Create a secret for your Cloudflare credentials:**

    ```bash
    kubectl create secret generic cloudflare-credentials \
      --from-literal=api-token=<YOUR_CLOUDFLARE_API_TOKEN> \
      --from-literal=account-id=<YOUR_CLOUDFLARE_ACCOUNT_ID>
    ```

2.  **Deploy the operator:**

    The operator can be deployed using the manifests in the `config` directory. You will need to update the `config/manager/manager.yaml` to use the secret you created.

    ```bash
    # TODO: Add deployment instructions once the manager.yaml is updated to use the secret
    ```

## Usage

Once the operator is deployed, you can create a Cloudflare Tunnel by creating a `CloudflareTunnel` custom resource.

**Example:**

```yaml
apiVersion: cloudflaretunnel.spotty.com.cn/v1alpha1
kind: CloudflareTunnel
metadata:
  name: my-tunnel
spec:
  name: my-first-tunnel
  cloudflareApiToken: <YOUR_CLOUDFLARE_API_TOKEN>
  cloudflareAccountId: <YOUR_CLOUDFLARE_ACCOUNT_ID>
```

When you apply this manifest, the operator will create a new Cloudflare Tunnel named `my-first-tunnel` in your Cloudflare account. The operator will also create a Kubernetes secret named `my-tunnel-token` containing the tunnel's token.

## Configuration

The operator is configured through the `CloudflareTunnel` custom resource.

## Contributing

Contributions are welcome! Please see the [contributing guide](CONTRIBUTING.md) for more information on how to get started.

## CI/CD

This project uses GitHub Actions to automatically build and push container images to Docker Hub and GitHub Container Registry. The workflow is defined in `.github/workflows/ci.yaml`.