// Package schemas embeds the JSON Schema files and registers them with the
// config package on import. CLI entry points should import this package with
// a blank identifier: import _ "github.com/kjourdan1/lzctl/schemas"
package schemas

import (
	"embed"

	"github.com/kjourdan1/lzctl/internal/config"
)

//go:embed lzctl-v1.schema.json
var fs embed.FS

func init() {
	data, err := fs.ReadFile("lzctl-v1.schema.json")
	if err != nil {
		panic("schemas: failed to read embedded lzctl-v1.schema.json: " + err.Error())
	}
	config.SetSchema(data)
}
