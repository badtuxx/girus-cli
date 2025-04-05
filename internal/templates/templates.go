package templates

import (
	"embed"
	"io/fs"
)

//go:embed manifests/*.yaml
var ManifestFS embed.FS

func GetManifest(name string) ([]byte, error) {
	return fs.ReadFile(ManifestFS, "manifests/"+name)
}
