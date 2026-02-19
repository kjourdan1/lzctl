# CLI Reference

Full reference for all `lzctl` commands, flags, and environment variables.

## Global Flags

| Flag | Short | Default | Env Var | Description |
|------|-------|---------|---------|-------------|
| `--config` | | `lzctl.yaml` | `LZCTL_CONFIG` | Config file path |
| `--repo-root` | | `.` | `LZCTL_REPO_ROOT` | Repository root path |
| `--verbose` | `-v` | `0` | `LZCTL_VERBOSE` | Verbosity level (0-3) |
| `--dry-run` | | `false` | | Simulate without making changes |
| `--json` | | `false` | | Output in JSON format |

## Commands

### `lzctl init`

Bootstrap a new landing zone project. Runs an interactive wizard that generates `lzctl.yaml` and optionally bootstraps the state backend.

```bash
lzctl init [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--tenant-id` | interactive | Azure AD tenant ID |

### `lzctl plan`

Run `terraform plan` across platform layers in dependency order.

```bash
lzctl plan [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer (`management-groups`, `identity`, `management`, `governance`, `connectivity`) |
| `--out` | | Save plan output to file |

### `lzctl apply`

Run `terraform apply` across platform layers in dependency order.

```bash
lzctl apply [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer |
| `--auto-approve` | `false` | Skip approval prompt |

### `lzctl validate`

Validate `lzctl.yaml` against the JSON schema and run cross-field checks.

```bash
lzctl validate [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--strict` | `false` | Treat warnings as errors |

**Checks:** JSON schema validation, cross-field validation (UUID formats, CIDR overlaps, state backend config, versioning/soft-delete enforcement), `terraform validate` per layer.

### `lzctl drift`

Detect infrastructure drift by running `terraform plan` per layer.

```bash
lzctl drift [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer |
| `--json` | `false` | JSON output with per-layer add/change/destroy counts |

### `lzctl status`

Show project status: metadata, platform layers, git info.

```bash
lzctl status [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--live` | `false` | Show live state from Azure |

### `lzctl rollback`

Rollback platform layers in reverse CAF dependency order.

```bash
lzctl rollback [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer to rollback |
| `--to` | | Timestamp to rollback to (ISO 8601) |
| `--auto-approve` | `false` | Skip confirmation prompt |

### `lzctl assess`

Run a readiness assessment for your landing zone project.

```bash
lzctl assess [flags]
```

Shows project metadata, platform layer readiness, and landing zone details from `lzctl.yaml`.

### `lzctl select`

Browse the CAF platform layer catalog and see which layers are active.

```bash
lzctl select [flags]
```

### `lzctl audit`

Run compliance checks against a live Azure tenant.

```bash
lzctl audit [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | tenant root | Audit scope |
| `--json` | `false` | JSON output |
| `--output` | | Write report to file |

### `lzctl schema`

Export or validate the `lzctl.yaml` JSON schema.

```bash
lzctl schema export    # Print the embedded JSON schema
lzctl schema validate  # Validate lzctl.yaml against the schema
```

### `lzctl docs`

Generate a README.md from the project configuration.

```bash
lzctl docs [flags]
```

### `lzctl doctor`

Check prerequisites and environment readiness.

```bash
lzctl doctor
```

**Checks:** terraform, az CLI, git, gh CLI (optional), Azure session, management group access, resource providers, state backend accessibility.

### `lzctl upgrade`

Check and apply AVM module version updates.

```bash
lzctl upgrade [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--apply` | `false` | Apply version updates to HCL files |
| `--allow-major` | `false` | Allow major version bumps |

---

### `lzctl state`

Terraform state lifecycle management. Treats state as a critical asset with dedicated operations for visibility, protection, and recovery.

> See [State Life Management Guide](operations/state-management.md) for full documentation.

#### `lzctl state list`

List all state files in the backend with lock status.

```bash
lzctl state list [--json]
```

**Output:**

```
LAYER               KEY                                SIZE       LAST MODIFIED             LOCK
management-groups   platform-management-groups.tfstate  2048 B     2026-02-19T10:30:00Z      🔓
identity            platform-identity.tfstate           1234 B     2026-02-19T10:31:00Z      🔓
connectivity        platform-connectivity.tfstate       8192 B     2026-02-19T10:32:00Z      🔒
```

#### `lzctl state snapshot`

Create point-in-time backups of state files before mutations.

```bash
lzctl state snapshot --all --tag "pre-sprint-5"
lzctl state snapshot --layer connectivity --tag "before-firewall"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Snapshot all state files |
| `--layer` | | Snapshot a specific layer |
| `--tag` | auto-timestamp | Label for the snapshot |

#### `lzctl state health`

Validate state backend security posture (versioning, encryption, soft delete, TLS).

```bash
lzctl state health [--json]
```

**Checks:** HTTPS-only, TLS 1.2, blob versioning, soft delete, container soft delete, infrastructure encryption.

#### `lzctl state unlock`

Force-release a stuck blob lease (after a failed pipeline).

```bash
lzctl state unlock --key platform-connectivity.tfstate
```

| Flag | Default | Description |
|------|---------|-------------|
| `--key` | required | State file key to unlock |

---

### `lzctl policy`

Policy-as-Code lifecycle management.

```bash
lzctl policy <subcommand> [flags]
```

| Subcommand | Description |
|------------|-------------|
| `create` | Scaffold policy definitions/initiatives/assignments/exemptions |
| `test` | Deploy in DoNotEnforce mode |
| `verify` | Generate compliance report |
| `remediate` | Create remediation tasks |
| `deploy` | Switch to Default enforcement |
| `status` | Show policy workflow state |
| `diff` | Compare desired vs actual state |

### `lzctl workload`

Landing zone (subscription) management.

#### `lzctl workload add`

Add a new landing zone to the project.

```bash
lzctl workload add <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--archetype` | `corp` | Landing zone archetype (`corp`, `online`, `sandbox`) |
| `--connected` | `true` | Connect to hub network |
| `--address-space` | | CIDR block for the landing zone |

#### `lzctl workload adopt`

Adopt an existing Azure subscription as a landing zone.

```bash
lzctl workload adopt <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--subscription` | required | Existing subscription ID |
| `--archetype` | `corp` | Landing zone archetype |
| `--connected` | `true` | Connect to hub network |

#### `lzctl workload list`

List all landing zones in the project.

```bash
lzctl workload list [--output json]
```

#### `lzctl workload remove`

Remove a landing zone from the project configuration.

```bash
lzctl workload remove <name>
```

### `lzctl version`

Show version information.

```bash
lzctl version
```

## Environment Variables

All flags can be set via environment variables with the `LZCTL_` prefix:

```bash
export LZCTL_REPO_ROOT=/path/to/repo
export LZCTL_VERBOSE=2
export LZCTL_CONFIG=/path/to/lzctl.yaml
```
