# Contributing to terraform-provider-graviteeam

Thank you for your interest in contributing!

## Development Setup

### Requirements

- [Go](https://golang.org/doc/install) >= 1.25
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- `golangci-lint` v2
- `tfplugindocs` v0.24.0
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose (for acceptance tests)

### Building

```bash
git clone https://github.com/maxvanp/terraform-provider-graviteeam.git
cd terraform-provider-graviteeam
make build
```

### Local Testing with dev_overrides

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "maxvanp/graviteeam" = "/path/to/terraform-provider-graviteeam"
  }
  direct {}
}
```

## Testing

### Unit Tests

```bash
make test
```

### Lint and Format

```bash
make lint
make fmt
```

### Acceptance Tests

Acceptance tests run against a real Gravitee AM instance:

```bash
# Start Gravitee AM
docker compose -f docker-compose.test.yml up -d

# Wait for it to be ready, then run tests
TF_ACC=1 make testacc
```

### Refresh the bundled OpenAPI reference

```bash
./scripts/update-openapi.sh
```

## Pull Request Process

1. Fork the repository and create a feature branch
2. Make your changes
3. Ensure all tests pass (`make test` and `make testacc`)
4. Run `make fmt` and `make lint`
5. Run `make docs` if you changed any schema
6. Open a pull request with a clear description of the changes

### PR Checklist

- [ ] Tests added/updated for new functionality
- [ ] `make fmt` and `make lint` pass
- [ ] `make docs` regenerated if schema changed
- [ ] CHANGELOG.md updated under `## [Unreleased]`

## Adding a New Resource

1. Create `internal/resources/<name>/` with resource and model files
2. Register in `internal/provider/provider.go`
3. Add client methods in `internal/client/`
4. Add acceptance test
5. Add example in `examples/resources/graviteeam_<name>/resource.tf`
6. Run `make docs` to generate documentation

## Code Style

- Follow standard Go conventions (`gofmt`, `goimports`)
- Use the Terraform Plugin Framework (not SDKv2)
- Keep resource files self-contained in their own package
