# ADR-001: Go as CLI Language

- **Status:** Accepted
- **Date:** 2026-02-16

## Context

lzctl est un outil CLI qui doit être distribué comme un binaire unique, performant, et cross-platform.

## Decision

Go (1.24+) est le langage de développement pour lzctl.

## Rationale

- **Single binary** — compilation statique, pas de runtime dependency
- **Cross-platform** — Linux, macOS, Windows sans effort
- **Écosystème CLI** — Cobra (CLI framework), Viper (config), survey (wizard)
- **Performance** — startup instantané, exécution rapide
- **Communauté IaC** — Terraform, Packer, et la majorité de l'écosystème HashiCorp sont en Go
- **Testabilité** — `go test` intégré, race detector, benchmarks

## Alternatives considérées

| Option | Rejetée car |
|--------|-------------|
| Python | Runtime dependency, startup lent, distribution complexe |
| Rust | Courbe d'apprentissage, écosystème CLI moins mature |
| TypeScript/Node | Runtime dependency, binaire volumineux (pkg/nexe) |

## Consequences

- Le projet suit les conventions Go (packages, interfaces, error handling)
- Les templates sont en Go `text/template`
- Les fichiers sont embarqués via `embed.FS`
