// Package config fornece funcionalidades para gerenciar as configurações do Girus
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config guarda todas as configurações do Girus
type Config struct {
	ExternalRepositories []ExternalLabRepository `yaml:"externalRepositories"`
}

// ExternalLabRepository representa um repositório Git contendo templates de laboratório
type ExternalLabRepository struct {
	URL          string `yaml:"url"`          // URL do repositório Git
	Branch       string `yaml:"branch"`       // Branch a ser usada (padrão: main)
	Description  string `yaml:"description"`  // Descrição opcional
	ManifestPath string `yaml:"manifestPath"` // Caminho para o arquivo de manifesto (padrão: girus-labs.yaml)
}

// GetConfigPath retorna o diretório de configuração específico da plataforma
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".girus")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadConfig carrega a configuração do disco
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Verifica se o arquivo existe, se não, cria o padrão
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			ExternalRepositories: []ExternalLabRepository{},
		}, nil
	}

	// Lê e analisa a configuração
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig salva a configuração no disco
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// AddRepository adiciona um novo repositório de laboratório externo à configuração
func AddRepository(repo ExternalLabRepository) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Define valores padrão se não fornecidos
	if repo.Branch == "" {
		repo.Branch = "main"
	}

	if repo.ManifestPath == "" {
		repo.ManifestPath = "girus-labs.yaml"
	}

	// Verifica se o repositório com a mesma URL já existe
	for i, existingRepo := range config.ExternalRepositories {
		if existingRepo.URL == repo.URL {
			// Atualiza a entrada existente
			config.ExternalRepositories[i] = repo
			return SaveConfig(config)
		}
	}

	// Adiciona um novo repositório
	config.ExternalRepositories = append(config.ExternalRepositories, repo)
	return SaveConfig(config)
}

// GetExternalRepositories retorna todos os repositórios externos configurados
func GetExternalRepositories() ([]ExternalLabRepository, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return config.ExternalRepositories, nil
}

// RemoveRepository remove um repositório da configuração
func RemoveRepository(url string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	for i, repo := range config.ExternalRepositories {
		if repo.URL == url {
			// Remove o repositório da lista
			config.ExternalRepositories = append(
				config.ExternalRepositories[:i],
				config.ExternalRepositories[i+1:]...,
			)
			return SaveConfig(config)
		}
	}

	// Repositório não encontrado (não é um erro)
	return nil
}
