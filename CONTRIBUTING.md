# Contributing to the Cloudflare Tunnel Operator

We welcome contributions to the Cloudflare Tunnel Operator! This document provides guidelines for developing and contributing to the project.

## Development Environment

To develop the operator locally, you will need:

*   Go (v1.21+)
*   Docker
*   `make`

## Building the Operator

To build the operator binary, run the following command:

```bash
make build
```

## Running the Tests

To run the unit tests, use the following command:

```bash
make test
```

## Running the Operator Locally

To run the operator on your local machine, you will need to provide your Cloudflare API token and account ID as environment variables.

```bash
export CLOUDFLARE_API_TOKEN=<YOUR_CLOUDFLARE_API_TOKEN>
export CLOUDFLARE_ACCOUNT_ID=<YOUR_CLOUDFLARE_ACCOUNT_ID>
make run
```

This will run the operator outside of a Kubernetes cluster, using your local kubeconfig file to connect to the cluster.

## Submitting a Pull Request

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix.
3.  Make your changes and commit them with a clear and descriptive message.
4.  Push your changes to your fork.
5.  Open a pull request against the `main` branch of the original repository.


setup kind cluster for development
```shell
cat <<EOF | kind create cluster --name=cloudflaretunnel --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: "0.0.0.0"
nodes:
- role: control-plane
  image: fronted-cn-beijing.cr.volces.com/container/kindest/node:v1.31.4
EOF


kind get kubeconfig --name cloudflaretunnel > ~/.kube/cloudflaretunnel.config

scp lion@192.168.10.10:~/.kube/cloudflaretunnel ~/.kube/cloudflaretunnel

docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' cloudflaretunnel-control-plane


kustomize build config/on-premises | kubectl apply -f -
kustomize build config/on-premises | kubectl delete -f -
```


update `server` like `https://172.18.0.6:6443`