# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go-based Kubernetes operator (Kubebuilder layout).
- `cmd/main.go`: application entrypoint.
- `api/v1alpha1`: CRD API types and generated deepcopy code.
- `internal/controller`: reconciliation logic and controller-level tests.
- `pkg/cloudflare`: Cloudflare client integration and unit tests.
- `config/`: Kubernetes manifests (`crd`, `rbac`, `manager`, `default`, `samples`, `on-premises`).
- `test/e2e`: end-to-end suite; `test/utils`: e2e helpers.
- `examples/`: small Cloudflare API usage examples.

## Build, Test, and Development Commands
Use `make help` to list all targets.
- `make build`: generate manifests/code, format, vet, and build `bin/manager`.
- `make run`: run the controller locally against your kubeconfig.
- `make test`: run non-e2e tests with envtest and write `cover.out`.
- `make test-e2e`: create Kind cluster, run e2e suite, then clean up.
- `make lint` / `make lint-fix`: run `golangci-lint` checks (or apply safe fixes).
- `make docker-build IMG=<image:tag>`: build operator image.
- `make deploy IMG=<image:tag>`: deploy to the current Kubernetes context.

## Coding Style & Naming Conventions
- Follow standard Go formatting: run `gofmt`/`goimports` (enforced by lint config).
- Keep package names short, lowercase, and domain-focused (`controller`, `cloudflare`).
- Use descriptive exported identifiers in `CamelCase`; unexported in `camelCase`.
- Keep API changes in `api/v1alpha1` and regenerate via `make generate manifests`.

## Testing Guidelines
- Frameworks: Go `testing`, Ginkgo v2, and Gomega.
- Test files must end with `_test.go`; prefer table-driven tests for pure logic.
- Controller/integration tests live in `internal/controller`; e2e tests in `test/e2e`.
- Run `make test` before each PR; run `make test-e2e` for cluster-impacting changes.

## Commit & Pull Request Guidelines
- Follow Conventional Commit-style prefixes seen in history: `feat:`, `fix:`, `refactor:`, `doc:`, `chore:`, `ci:`, `lint:`.
- Keep commits focused and atomic; include regenerated artifacts when API/manifests change.
- PRs should include: clear summary, scope/risk, linked issue (if any), and test evidence (`make test`, lint, e2e when relevant).

## Security & Configuration Tips
- Do not commit credentials. Use environment variables or Kubernetes Secrets for Cloudflare tokens/account IDs.
- Start from `.env.example` for local variables and `config/samples/` for CR examples.
- Write contributor communication (issues/PR descriptions/reviews) in English.
