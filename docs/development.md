# Development Guide

Technical guide for contributors and maintainers.

## Code Architecture

```
cmd/                    Cobra CLI commands
internal/               Business logic
  audit/                CAF compliance audit
  azauth/               Azure authentication
  azure/                az CLI wrapper
  bootstrap/            State backend bootstrap
  config/               Parsing, validation, cross-checks for lzctl.yaml
  doctor/               Prerequisite checks
  exitcode/             Standardised exit codes
  importer/             Terraform import block generation
  oidcsetup/            OIDC federated credential setup
  output/               Output formatting (Info, Success, Warning, Error)
  planverify/           Plan verification
  policy/               Policy-as-Code lifecycle
  state/                State file lifecycle management
  template/             Go template engine
  upgrade/              AVM module version checker and updater
  wizard/               Interactive wizard (survey)
schemas/                Embedded JSON schema (embed.FS)
templates/              Go templates (embed.FS)
  manifest/             lzctl.yaml templates
  shared/               Shared templates (backend, providers, gitignore)
  platform/             Terraform templates per layer
  pipelines/            CI/CD templates (GitHub Actions, Azure DevOps)
profiles/               CAF profile catalogue (catalog.yaml)
```

## Conventions

### Commands (cmd/)

Files in `cmd/` are **thin wrappers**:
1. Parse flags
2. Load configuration
3. Call logic in `internal/`
4. Format and display the result

```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Short description",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.Load(cfgFile, repoRoot)
        if err != nil {
            return err
        }
        return mypackage.Run(cfg)
    },
}
```

### Internal Packages

- No dependency on Cobra
- Interfaces for testability
- Wrapped errors: `fmt.Errorf("context: %w", err)`

### Output

Use `internal/output` for messages:

```go
output.Info("Processing layer: %s", layer)
output.Success("Deployment complete")
output.Warning("Versioning is disabled")
output.Error("Failed to apply: %v", err)
```

### Templates

Go templates live in `templates/` and are embedded via `embed.FS`:

```
templates/
├── manifest/
│   └── lzctl.yaml.tmpl
├── shared/
│   ├── backend.tf.tmpl
│   ├── backend.hcl.tmpl
│   ├── providers.tf.tmpl
│   └── gitignore.tmpl
├── platform/
│   ├── management-groups/
│   ├── identity/
│   ├── management/
│   ├── governance/
│   └── connectivity/
└── pipelines/
    ├── github/
    └── azuredevops/
```

## Debug

```bash
# Verbose output
lzctl plan -vvv

# Dry-run
lzctl apply --dry-run
```

## Azure Integration Tests (live)

Azure live tests are separate from standard PR tests:

- Standard PR/CI: `go test ./...` (no real Azure calls)
- Live (nightly or manual): `go test -tags=integration -v ./test/integration/...`

Required variables for live tests:

- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_SUBSCRIPTION_ID`

## Adding a New Command

1. Create `cmd/<command>.go` with a Cobra command
2. Create `internal/<package>/<package>.go` with the logic
3. Wire in `cmd/root.go`: `rootCmd.AddCommand(myCmd)`
4. Add tests in `cmd/cmd_test.go` and `internal/<package>/<package>_test.go`
5. Document in `docs/commands/<command>.md`
6. Update `docs/cli-reference.md`

## Adding an Audit Rule

1. Create the function in `internal/audit/compliance_engine.go`
2. Follow the existing `check<Rule>()` pattern
3. Test with fixtures in `audit_test.go`
4. Document the rule and its remediation

## Adding a Doctor Check

1. Add in `internal/doctor/checks.go`
2. Categorise: `tools`, `auth`, `azure`, `state`
3. Return a `CheckResult` with Name, Status, Details, Fix
4. Test in `checks_test.go`

## Adding a Field to lzctl.yaml

1. Add the field in `internal/config/schema.go`
2. Add the default value in `internal/config/defaults.go`
3. Update `schemas/lzctl-v1.schema.json`
4. Add cross-field validation if needed in `internal/config/crossvalidator.go`
5. Update the template in `templates/manifest/lzctl.yaml.tmpl`
6. Document in the README
