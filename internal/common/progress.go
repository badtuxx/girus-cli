package common

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var _, homeDir = os.UserHomeDir()
var filePath = fmt.Sprintf("%s/%s/%s", homeDir, ".girus", "progresso.yaml")

type Progress struct {
	Labs []Lab `yaml:"labs,omitempty"`
}

type Lab struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
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

// SaveProgressToFile salva o progresso atual em um arquivo YAML na mesma pasta do girus-cli
func (p *Progress) SaveProgressToFile() error {
	// Converte o struct Progress para YAML
	data, err := yaml.Marshal(p.Labs)
	if err != nil {
		return fmt.Errorf("erro ao converter progresso.yaml para YAML: %w", err)
	}

	// Verifica se o diretorio do girus-cli já existe
	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erro ao criar o diretorio do girus-cli: %w", err)
	}

	// Verifica se o arquivo progresso.yaml já existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Se não existir, cria o arquivo com os dados do struct Progress
		err = os.WriteFile(filePath, data, 0644)
		if err != nil {
			return fmt.Errorf("erro ao criar o arquivo progresso.yaml: %w", err)
		}
	}

	// Se existir, atualiza o arquivo com os dados do struct Progress
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("erro ao atualizar o arquivo progresso.yaml: %w", err)
	}

	return nil
}

func (p *Progress) LoadProgressFromFile() (*Progress, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Se o arquivo progresso.yaml não existir, retorna um struct Progress vazio
		return &Progress{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o arquivo progresso.yaml: %w", err)
	}

	var progress Progress
	return &progress, yaml.Unmarshal(data, &progress)
}
