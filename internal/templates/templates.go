package templates

import (
	"embed"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/badtuxx/girus-cli/internal/common"
)

//go:embed manifests/*.yaml manifests_es/*.yaml
var ManifestFS embed.FS

func GetManifest(name string) ([]byte, error) {
	dir := "manifests"
	if common.Lang() == "es" {
		dir = "manifests_es"
	}
	// embed.FS sempre usa "/" como separador, independente do SO.
	// Usar path.Join (não filepath.Join) garante o caminho correto no Windows.
	return fs.ReadFile(ManifestFS, path.Join(dir, name))
}

// ListManifests retorna uma lista de nomes de todos os arquivos YAML no diretório de manifests
func ListManifests() ([]string, error) {
	dir := "manifests"
	if common.Lang() == "es" {
		dir = "manifests_es"
	}
	entries, err := fs.ReadDir(ManifestFS, dir)
	if err != nil {
		return nil, err
	}

	var manifests []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			manifests = append(manifests, entry.Name())
		}
	}

	return manifests, nil
}
