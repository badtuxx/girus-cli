// Package git fornece funcionalidades para operações com repositórios Git
package git

import (
	"fmt"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// CloneRepository clona um repositório Git para um diretório temporário
func CloneRepository(url, branch string) (string, *git.Repository, error) {
	// Cria diretório temporário
	tempDir, err := os.MkdirTemp("", "girus-repo-*")
	if err != nil {
		return "", nil, fmt.Errorf("erro ao criar diretório temporário: %w", err)
	}

	// Opções de clonagem
	options := &git.CloneOptions{
		URL:           url,
		Progress:      nil, // Clone silencioso
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}

	// Clona o repositório
	repo, err := git.PlainClone(tempDir, false, options)
	if err != nil {
		os.RemoveAll(tempDir) // Limpa em caso de erro
		return "", nil, fmt.Errorf("erro ao clonar repositório %s (branch %s): %w", url, branch, err)
	}

	return tempDir, repo, nil
}

// GetFile recupera um arquivo do repositório clonado
func GetFile(repoPath, filePath string) ([]byte, error) {
	fullPath := filepath.Join(repoPath, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo %s: %w", filePath, err)
	}
	return data, nil
}

// FileExists verifica se um arquivo existe no repositório clonado
func FileExists(repoPath, filePath string) bool {
	fullPath := filepath.Join(repoPath, filePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ListFiles lista arquivos em um diretório do repositório clonado
func ListFiles(repoPath, dirPath string) ([]string, error) {
	fullPath := filepath.Join(repoPath, dirPath)

	// Verifica se o diretório existe
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar diretório %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s não é um diretório", dirPath)
	}

	// Lista arquivos no diretório
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar diretório %s: %w", dirPath, err)
	}

	var files []string
	for _, entry := range entries {
		files = append(files, filepath.Join(dirPath, entry.Name()))
	}

	return files, nil
}

// CleanupRepo remove o diretório temporário do repositório
func CleanupRepo(repoPath string) error {
	if repoPath == "" {
		return nil
	}
	return os.RemoveAll(repoPath)
}
