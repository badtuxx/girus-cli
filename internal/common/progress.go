package common

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var homeDir, _ = os.UserHomeDir()
var DefaultFilePath = filepath.Join(homeDir, ".girus", "progresso.yaml")

type Progress struct {
	FilePath string `yaml:"-"`
	Labs     []Lab  `yaml:"labs,omitempty"`
}

type Lab struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
}

func NewProgress(filePath string) *Progress {
	if filePath == "" {
		filePath = DefaultFilePath
	}
	return &Progress{
		FilePath: filePath,
		Labs:     []Lab{},
	}
}

func (p *Progress) AddLab(labName string, status string) {
	p.Labs = append(p.Labs, Lab{
		Name:   labName,
		Status: status,
	})
}

func (p *Progress) GetLab(labName string) *Lab {
	for i := range p.Labs {
		if p.Labs[i].Name == labName {
			return &p.Labs[i]
		}
	}
	return nil
}

func (p *Progress) SetLabStatus(labName string, status string) error {
	lab := p.GetLab(labName)
	if lab == nil {
		return fmt.Errorf("😩%s não encontrado", labName)
	}
	lab.Status = status
	return nil
}

// SaveProgressToFile salva o progresso atual em um arquivo YAML
func (p *Progress) SaveProgressToFile() error {
	path := p.FilePath
	if path == "" {
		path = DefaultFilePath
	}

	// Converte o struct Progress para YAML
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("erro ao converter progresso.yaml para YAML: %w", err)
	}

	// Verifica se o diretorio do girus-cli já existe
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erro ao criar o diretorio do girus-cli: %w", err)
	}

	// Escreve o arquivo (sobrescreve se existir)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("erro ao escrever o arquivo progresso.yaml: %w", err)
	}

	return nil
}

func (p *Progress) LoadProgressFromFile() error {
	path := p.FilePath
	if path == "" {
		path = DefaultFilePath
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Se o arquivo não existir, apenas retorna nil (lista vazia)
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("erro ao ler o arquivo progresso.yaml: %w", err)
	}

	return yaml.Unmarshal(data, p)
}
