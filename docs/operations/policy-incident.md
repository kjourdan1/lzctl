# Policy Incident Response

Response procedure when an Azure Policy blocks a deployment or generates non-compliance alerts.

## Incident Types

| Type | Description | Urgency |
|------|-------------|---------|
| **Blocking** | A `Deny` policy blocks a legitimate deployment | High |
| **Non-compliance** | Existing resources do not comply with a policy | Medium |
| **False positive** | A policy incorrectly reports non-compliance | Low |

## Procedure — Blocking Policy

### 1. Identify the Policy

```bash
# View policy status
lzctl policy status

# Compare local vs deployed
lzctl policy diff
```

### 2. Create a Temporary Exemption

```bash
# Scaffold an exemption
lzctl policy create --type exemption --name "temp-deploy-fix"
```

The exemption is created in `policies/exemptions/` with a mandatory expiration date.

### 3. Resolve

- **If the policy is correct**: modify the Terraform code to comply
- **If the policy is too restrictive**: adjust the policy definition
- **If it's a false positive**: create an upstream report

### 4. Remove the Exemption

After resolution, delete the exemption and redeploy:

```bash
lzctl policy deploy
lzctl policy verify
```

## Procedure — Non-Compliance

### 1. Generate the Compliance Report

```bash
lzctl policy verify
```

### 2. Create Remediation Tasks

```bash
lzctl policy remediate
```

### 3. Verify the Resolution

```bash
lzctl audit
```

## Prevention

- Always test policies in audit mode first: `lzctl policy test`
- Use `lzctl policy verify` before switching to enforcement
- Document exemptions with a justification and an expiration date
