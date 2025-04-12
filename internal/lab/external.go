// Package lab implementa funcionalidades relacionadas aos laboratórios do Girus
package lab

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/badtuxx/girus-cli/internal/config"
	"github.com/badtuxx/girus-cli/internal/git"
	"github.com/badtuxx/girus-cli/internal/helpers"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/yaml.v3"
)

// BackendRestartStatus controla se o backend já foi reiniciado nesta execução
var BackendRestartNeeded = false

// LabManifest representa a estrutura do arquivo de manifesto de labs
type LabManifest struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Labs        []LabEntry `yaml:"labs"`
}

// LabEntry representa um laboratório individual no manifesto
type LabEntry struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ParseLabManifest analisa o conteúdo de um arquivo de manifesto de laboratórios
func ParseLabManifest(data []byte) (*LabManifest, error) {
	manifest := &LabManifest{}
	err := yaml.Unmarshal(data, manifest)
	if err != nil {
		return nil, fmt.Errorf("erro ao analisar o manifesto de labs: %w", err)
	}

	return manifest, nil
}

// ValidateManifest valida se o manifesto tem a estrutura correta
func ValidateManifest(manifest *LabManifest) error {
	// Verifica se o manifesto tem nome
	if manifest.Name == "" {
		return fmt.Errorf("o manifesto de labs não possui um nome")
	}

	// Verifica se há labs definidos
	if len(manifest.Labs) == 0 {
		return fmt.Errorf("o manifesto não contém definições de laboratórios")
	}

	// Verifica cada entrada de lab
	for i, lab := range manifest.Labs {
		if lab.Name == "" {
			return fmt.Errorf("lab #%d não possui nome", i+1)
		}
		if lab.Path == "" {
			return fmt.Errorf("lab '%s' não possui caminho para o arquivo", lab.Name)
		}
	}

	return nil
}

// ApplySingleLabFile aplica um único arquivo de lab sem reiniciar o backend
// Esta é uma versão modificada do AddLabFromFile que não reinicia o backend automaticamente
func ApplySingleLabFile(labFile string, verboseMode bool) error {
	// Verificar se o arquivo existe
	if _, err := os.Stat(labFile); os.IsNotExist(err) {
		return fmt.Errorf("arquivo '%s' não encontrado", labFile)
	}

	if verboseMode {
		fmt.Printf("📦 Processando laboratório: %s\n", labFile)
		fmt.Println("   Aplicando ConfigMap no cluster...")
	}

	// Aplicar o ConfigMap no cluster
	applyCmd := exec.Command("kubectl", "apply", "-f", labFile)
	var stderr bytes.Buffer
	applyCmd.Stderr = &stderr

	if verboseMode {
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
	}

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("erro ao aplicar o laboratório: %v - %s", err, stderr.String())
	}

	return nil
}

// RestartBackendAfterLabsApplied reinicia o backend para carregar os novos templates
func RestartBackendAfterLabsApplied(verboseMode bool) error {
	fmt.Println("\n🔄 Reiniciando backend para carregar novos templates...")

	// Reiniciar o deployment do backend
	restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
	var stderr bytes.Buffer
	restartCmd.Stderr = &stderr

	if verboseMode {
		restartCmd.Stdout = os.Stdout
		restartCmd.Stderr = os.Stderr
	}

	err := restartCmd.Run()
	if err != nil {
		return fmt.Errorf("erro ao reiniciar o backend: %v - %s", err, stderr.String())
	}

	// Usar barra de progresso para aguardar
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("   Aguardando reinício do backend"),
		progressbar.OptionSetWidth(80),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)

	// Aguardar o reinício completar
	waitCmd := exec.Command("kubectl", "rollout", "status", "deployment/girus-backend", "-n", "girus", "--timeout=60s")
	var waitOutput bytes.Buffer
	waitCmd.Stdout = &waitOutput
	waitCmd.Stderr = &waitOutput

	// Iniciar o comando
	err = waitCmd.Start()
	if err != nil {
		bar.Finish()
		return fmt.Errorf("erro ao verificar status do reinício: %v", err)
	}

	// Atualizar a barra de progresso enquanto o comando está em execução
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				bar.Add(1)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Aguardar o final do comando
	waitCmd.Wait()
	close(done)
	bar.Finish()

	// Aguardar mais alguns segundos para que o backend reinicie completamente
	fmt.Println("   Aguardando inicialização completa...")
	time.Sleep(3 * time.Second)

	// Verificar se precisamos reconfigurar os port-forwards
	portForwardStatus := helpers.CheckPortForwardNeeded()
	if portForwardStatus {
		fmt.Println("\n🔌 Reconfigurando port-forwards após reinício do backend...")
		if err := k8s.SetupPortForward("girus"); err != nil {
			fmt.Println("⚠️ Aviso:", err)
			fmt.Println("   Para configurar manualmente, execute:")
			fmt.Println("   kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
			fmt.Println("   kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
		} else {
			fmt.Println("✅ Port-forwards reconfigurados com sucesso!")
		}
	}

	return nil
}

// ProcessExternalRepo processa um repositório externo para extrair laboratórios
func ProcessExternalRepo(repo config.ExternalLabRepository, verboseMode bool) ([]string, error) {
	appliedLabs := []string{}

	// Clona o repositório
	fmt.Printf("📦 Clonando repositório %s...\n", repo.URL)
	repoPath, _, err := git.CloneRepository(repo.URL, repo.Branch)
	if err != nil {
		return nil, fmt.Errorf("erro ao clonar repositório: %w", err)
	}
	defer git.CleanupRepo(repoPath)

	// Lê o arquivo de manifesto
	fmt.Printf("🔍 Lendo arquivo de manifesto: %s\n", repo.ManifestPath)
	manifestData, err := git.GetFile(repoPath, repo.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de manifesto: %w", err)
	}

	// Analisa o manifesto
	manifest, err := ParseLabManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("erro ao analisar manifesto: %w", err)
	}

	// Valida o manifesto
	if err := ValidateManifest(manifest); err != nil {
		return nil, fmt.Errorf("manifesto inválido: %w", err)
	}

	fmt.Printf("✅ Encontrados %d laboratórios no repositório\n", len(manifest.Labs))

	// Processa cada lab definido no manifesto
	for _, labEntry := range manifest.Labs {
		fmt.Printf("   - Processando laboratório: %s\n", labEntry.Name)

		// Constrói o caminho completo para o arquivo do lab
		labFilePath := filepath.Join(repoPath, labEntry.Path)

		// Verifica se o arquivo existe
		if _, err := os.Stat(labFilePath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Arquivo de lab não encontrado: %s\n", labEntry.Path)
			continue
		}

		// Extrai o lab para um arquivo temporário
		tempFile, err := ExtractLabFile(repoPath, labEntry.Path)
		if err != nil {
			fmt.Printf("❌ Erro ao extrair laboratório %s: %v\n", labEntry.Name, err)
			continue
		}

		// Aplica o lab usando a versão modificada que não reinicia o backend
		fmt.Printf("   - Aplicando laboratório: %s\n", labEntry.Name)
		if err := ApplySingleLabFile(tempFile, verboseMode); err != nil {
			fmt.Printf("❌ Erro ao aplicar laboratório %s: %v\n", labEntry.Name, err)
			os.Remove(tempFile)
			continue
		}

		// Remove o arquivo temporário após o uso
		os.Remove(tempFile)

		// Registra que o lab foi aplicado
		appliedLabs = append(appliedLabs, labEntry.Name)
	}

	// Se aplicou algum laboratório, reinicia o backend uma única vez
	if len(appliedLabs) > 0 {
		if err := RestartBackendAfterLabsApplied(verboseMode); err != nil {
			fmt.Printf("⚠️ Aviso ao reiniciar backend: %v\n", err)
			fmt.Println("   Os laboratórios foram aplicados, mas pode ser necessário reiniciar o backend manualmente:")
			fmt.Println("   kubectl rollout restart deployment/girus-backend -n girus")
		}
	}

	return appliedLabs, nil
}

// ExtractLabFile extrai um arquivo de lab do repositório para um arquivo temporário
func ExtractLabFile(repoPath, labPath string) (string, error) {
	// Constrói o caminho completo para o arquivo do lab
	labFilePath := filepath.Join(repoPath, labPath)

	// Verifica se o arquivo existe
	if _, err := os.Stat(labFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("arquivo de lab não encontrado: %s", labPath)
	}

	// Lê o conteúdo do arquivo
	content, err := os.ReadFile(labFilePath)
	if err != nil {
		return "", fmt.Errorf("erro ao ler arquivo de lab: %w", err)
	}

	// Cria um arquivo temporário para o lab
	tempFile, err := os.CreateTemp("", "girus-lab-*.yaml")
	if err != nil {
		return "", fmt.Errorf("erro ao criar arquivo temporário: %w", err)
	}

	// Escreve o conteúdo no arquivo temporário
	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("erro ao escrever no arquivo temporário: %w", err)
	}

	tempFile.Close()
	return tempFile.Name(), nil
}

// GetRepoNameFromURL extrai o nome do repositório a partir da URL
func GetRepoNameFromURL(url string) string {
	// Remove o .git do final, se presente
	url = strings.TrimSuffix(url, ".git")

	// Divide a URL por / e pega o último segmento
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown-repo"
}

// ApplyLabFromExternalRepo aplica um lab específico de um repositório externo
func ApplyLabFromExternalRepo(repo config.ExternalLabRepository, labName string, verboseMode bool) error {
	// Clona o repositório
	repoPath, _, err := git.CloneRepository(repo.URL, repo.Branch)
	if err != nil {
		return fmt.Errorf("erro ao clonar repositório: %w", err)
	}
	defer git.CleanupRepo(repoPath)

	// Lê o arquivo de manifesto
	manifestData, err := git.GetFile(repoPath, repo.ManifestPath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo de manifesto: %w", err)
	}

	// Analisa o manifesto
	manifest, err := ParseLabManifest(manifestData)
	if err != nil {
		return fmt.Errorf("erro ao analisar manifesto: %w", err)
	}

	// Procura o lab pelo nome
	var labEntry *LabEntry
	for i := range manifest.Labs {
		if manifest.Labs[i].Name == labName {
			labEntry = &manifest.Labs[i]
			break
		}
	}

	if labEntry == nil {
		return fmt.Errorf("laboratório '%s' não encontrado no repositório", labName)
	}

	// Extrai o lab para um arquivo temporário
	tempFile, err := ExtractLabFile(repoPath, labEntry.Path)
	if err != nil {
		return fmt.Errorf("erro ao extrair laboratório: %w", err)
	}
	defer os.Remove(tempFile)

	// Aplica o lab usando a função que não reinicia o backend
	if err := ApplySingleLabFile(tempFile, verboseMode); err != nil {
		return fmt.Errorf("erro ao aplicar laboratório: %w", err)
	}

	// Reinicia o backend uma única vez após aplicar o lab
	if err := RestartBackendAfterLabsApplied(verboseMode); err != nil {
		return fmt.Errorf("aviso ao reiniciar backend: %w", err)
	}

	return nil
}

// LoadExternalLabs carrega laboratórios de repositórios externos
func LoadExternalLabs(verboseMode bool) (int, error) {
	// Obtém a lista de repositórios externos
	repos, err := config.GetExternalRepositories()
	if err != nil {
		return 0, fmt.Errorf("erro ao carregar configuração de repositórios: %w", err)
	}

	if len(repos) == 0 {
		// Nenhum repositório configurado
		return 0, nil
	}

	// Conta o total de laboratórios aplicados
	totalApplied := 0
	allAppliedLabs := []string{}

	fmt.Printf("🔍 Carregando repositórios de laboratórios externos (%d)...\n", len(repos))

	// Processa cada repositório, aplicando os labs mas sem reiniciar o backend
	for _, repo := range repos {
		fmt.Printf("\n📦 Repositório: %s\n", repo.URL)

		repoPath, _, err := git.CloneRepository(repo.URL, repo.Branch)
		if err != nil {
			fmt.Printf("⚠️  Erro ao clonar repositório %s: %v\n", repo.URL, err)
			continue
		}
		defer git.CleanupRepo(repoPath)

		manifestData, err := git.GetFile(repoPath, repo.ManifestPath)
		if err != nil {
			fmt.Printf("⚠️  Erro ao ler arquivo de manifesto %s: %v\n", repo.ManifestPath, err)
			continue
		}

		manifest, err := ParseLabManifest(manifestData)
		if err != nil {
			fmt.Printf("⚠️  Erro ao analisar manifesto %s: %v\n", repo.ManifestPath, err)
			continue
		}

		if err := ValidateManifest(manifest); err != nil {
			fmt.Printf("⚠️  Manifesto inválido %s: %v\n", repo.ManifestPath, err)
			continue
		}

		// Processa os laboratórios sem reiniciar o backend
		fmt.Printf("✅ Encontrados %d laboratórios no repositório\n", len(manifest.Labs))
		appliedCount := 0

		for _, labEntry := range manifest.Labs {
			fmt.Printf("   - Processando laboratório: %s\n", labEntry.Name)

			labFilePath := filepath.Join(repoPath, labEntry.Path)
			if _, err := os.Stat(labFilePath); os.IsNotExist(err) {
				fmt.Printf("⚠️  Arquivo de lab não encontrado: %s\n", labEntry.Path)
				continue
			}

			tempFile, err := ExtractLabFile(repoPath, labEntry.Path)
			if err != nil {
				fmt.Printf("❌ Erro ao extrair laboratório %s: %v\n", labEntry.Name, err)
				continue
			}

			if err := ApplySingleLabFile(tempFile, verboseMode); err != nil {
				fmt.Printf("❌ Erro ao aplicar laboratório %s: %v\n", labEntry.Name, err)
				os.Remove(tempFile)
				continue
			}

			os.Remove(tempFile)
			appliedCount++
			allAppliedLabs = append(allAppliedLabs, labEntry.Name)
		}

		fmt.Printf("✅ %d laboratórios aplicados do repositório %s\n", appliedCount, GetRepoNameFromURL(repo.URL))
		totalApplied += appliedCount
	}

	// Se pelo menos um laboratório foi aplicado, reinicia o backend uma única vez
	if totalApplied > 0 {
		if err := RestartBackendAfterLabsApplied(verboseMode); err != nil {
			fmt.Printf("⚠️ Aviso ao reiniciar backend: %v\n", err)
			fmt.Println("   Os laboratórios foram aplicados, mas pode ser necessário reiniciar o backend manualmente:")
			fmt.Println("   kubectl rollout restart deployment/girus-backend -n girus")
		}
	}

	return totalApplied, nil
}

// ApplyExternalLabs processa todos os repositórios externos configurados
func ApplyExternalLabs(verboseMode bool) (int, []string, error) {
	// Obtém a lista de repositórios externos
	repos, err := config.GetExternalRepositories()
	if err != nil {
		return 0, nil, fmt.Errorf("erro ao carregar configuração de repositórios: %w", err)
	}

	if len(repos) == 0 {
		// Nenhum repositório configurado
		return 0, nil, nil
	}

	// Lista para armazenar nomes de laboratórios aplicados
	allAppliedLabs := []string{}

	// Processa cada repositório
	for _, repo := range repos {
		applied, err := ProcessExternalRepo(repo, verboseMode)
		if err != nil {
			return len(allAppliedLabs), allAppliedLabs, fmt.Errorf("erro no repositório %s: %w", repo.URL, err)
		}

		// Adiciona os laboratórios aplicados à lista geral
		for _, labName := range applied {
			repoName := GetRepoNameFromURL(repo.URL)
			allAppliedLabs = append(allAppliedLabs, fmt.Sprintf("%s (de %s)", labName, repoName))
		}
	}

	return len(allAppliedLabs), allAppliedLabs, nil
}
