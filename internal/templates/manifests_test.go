package templates_test

import (
	"embed"
	"path/filepath"
	"testing"

	"io/fs"

	"gopkg.in/yaml.v3"
)

//go:embed manifests/*.yaml
var manifests embed.FS

func TestYAMLFilesAreValid(t *testing.T) {
	err := fs.WalkDir(manifests, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".yaml" {
			data, err := manifests.ReadFile(path)
			if err != nil {
				t.Errorf("Erro ao ler %s: %v", path, err)
				return nil
			}

			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				t.Errorf("YAML inválido em %s: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Erro ao percorrer diretório de manifests: %v", err)
	}
}
