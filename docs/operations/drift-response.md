# Drift Response

Response procedure when infrastructure drift is detected.

## Detection

Drift is detected via:
- `lzctl drift` — on-demand local scan
- Scheduled CI/CD pipeline (weekly) — automatically creates an issue/work item

## Classification

| Type | Description | Action |
|------|-------------|--------|
| **Addition** | Resource created outside Terraform | Import or delete |
| **Modification** | Attribute manually changed | Fix in code or revert |
| **Deletion** | Resource manually deleted | Redeploy or update the code |

## Procedure

### 1. Identify the Drift

```bash
# Full scan
lzctl drift

# Scan a specific layer
lzctl drift --layer connectivity

# JSON output for analysis
lzctl drift --json
```

### 2. Analyze

- Check whether the change was intentional (maintenance, incident)
- Identify the affected layer and resources
- Assess the impact (blast radius)

### 3. Resolve

**Option A — Align code with reality:**
```bash
# Update the Terraform configuration
# Then validate
lzctl validate
lzctl plan --layer <layer>
```

**Option B — Revert to the declared state:**
```bash
# Re-apply the Terraform configuration
lzctl apply --layer <layer>
```

**Option C — Import the resource:**
```bash
# If a resource was added manually
lzctl import --resource-group <rg> --layer <layer>
```

### 4. Prevent

- Enable Azure Policy `Deny` to prevent manual modifications
- Restrict direct write permissions via RBAC
- Document the resolution in the issue/work item

## Escalation

| Severity | Criteria | SLA |
|----------|----------|-----|
| Critical | Drift on management-groups or identity | 4h |
| High | Drift on connectivity or governance | 24h |
| Medium | Drift on management | 72h |
| Low | Drift on landing-zones | Next sprint |
