package templates

import "embed"

// FS contains embedded template files used by the template engine.
//
//go:embed manifest/*.tmpl shared/*.tmpl platform/*/*.tmpl platform/*/*/*.tmpl pipelines/*/*.tmpl landing-zones/*/*.tmpl
var FS embed.FS
