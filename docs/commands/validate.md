# lzctl validate

Multi-layer validation of the manifest and Terraform configuration.

## Synopsis

```bash
lzctl validate [flags]
```

## Description

Runs three levels of validation:

1. **JSON Schema** — Validates `lzctl.yaml` against the embedded schema
2. **Cross-validation** — Checks cross-field rules:
   - UUID format (tenant, subscription, state backend)
   - CIDR overlaps (hub vs spokes)
   - Storage account name length (3-24 characters)
   - State versioning and soft delete enabled
3. **Terraform validate** — Runs `terraform validate` on each layer

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--strict` | `false` | Treat warnings as errors |

## Output

```
✅ Schema validation passed
✅ Cross-field validation: 0 errors, 0 warnings
✅ Terraform validate: management-groups — ok
✅ Terraform validate: identity — ok
✅ Terraform validate: connectivity — ok
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All valid |
| 1 | Validation errors |

In `--strict` mode, warnings also trigger a non-zero exit code.

## Examples

```bash
# Standard validation
lzctl validate

# Strict (CI)
lzctl validate --strict

# JSON output
lzctl validate --json
```

## See Also

- [schema](schema.md) — export the JSON schema
- [init](init.md) — validate after initialisation
