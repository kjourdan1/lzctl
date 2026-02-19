#!/usr/bin/env bash
# =============================================================================
# lzctl Demo Script
# Demonstrates a full landing zone lifecycle: init → validate → plan → apply
# =============================================================================

set -euo pipefail

BOLD='\033[1m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
RESET='\033[0m'

step() {
  echo ""
  echo -e "${BOLD}${CYAN}━━━ $1 ━━━${RESET}"
  echo ""
}

run() {
  echo -e "${GREEN}\$ $*${RESET}"
  "$@"
  echo ""
}

# ---------------------------------------------------------------------------
# Pre-requisites check
# ---------------------------------------------------------------------------
step "1/8 — Check prerequisites"
run lzctl doctor

# ---------------------------------------------------------------------------
# Scaffold a new landing zone repository
# ---------------------------------------------------------------------------
step "2/8 — Initialize a new landing zone"
run lzctl init \
  --tenant-id "00000000-0000-0000-0000-000000000000" \
  --subscription-id "11111111-1111-1111-1111-111111111111"

# ---------------------------------------------------------------------------
# Show the generated manifest
# ---------------------------------------------------------------------------
step "3/8 — Inspect generated manifest"
run cat lzctl.yaml

# ---------------------------------------------------------------------------
# Validate the configuration
# ---------------------------------------------------------------------------
step "4/8 — Validate configuration"
run lzctl validate --strict

# ---------------------------------------------------------------------------
# Plan all layers
# ---------------------------------------------------------------------------
step "5/8 — Plan all layers (dry run)"
run lzctl plan

# ---------------------------------------------------------------------------
# Apply (with auto-approve for demo)
# ---------------------------------------------------------------------------
step "6/8 — Apply all layers"
run lzctl apply --auto-approve

# ---------------------------------------------------------------------------
# Check status
# ---------------------------------------------------------------------------
step "7/8 — Check deployment status"
run lzctl status --live

# ---------------------------------------------------------------------------
# Add a workload landing zone
# ---------------------------------------------------------------------------
step "8/8 — Add a workload landing zone"
run lzctl workload add \
  --name "app-team-alpha" \
  --subscription-id "22222222-2222-2222-2222-222222222222"

echo ""
echo -e "${BOLD}${GREEN}✓ Demo complete!${RESET}"
echo ""
echo "Next steps:"
echo "  lzctl audit         — Score your environment against the CAF"
echo "  lzctl drift         — Detect manual changes"
echo "  lzctl upgrade       — Update AVM module versions"
echo "  lzctl state health  — Verify state backend security"
