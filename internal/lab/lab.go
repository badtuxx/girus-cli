// Package lab implementa funcionalidades relacionadas aos laboratórios do Girus
package lab

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/badtuxx/girus-cli/internal/helpers"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/schollz/progressbar/v3"
)

// AddLabFromFile adiciona um novo template de laboratório a partir de um arquivo
func AddLabFromFile(labFile string, verboseMode bool) {
	// Verificar se o arquivo existe
	if _, err := os.Stat(labFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Erro: arquivo '%s' não encontrado\n", labFile)
		os.Exit(1)
	}

	fmt.Println("🔍 Verificando ambiente Girus...")

	// Verificar se há um cluster Girus ativo
	checkCmd := exec.Command("kubectl", "get", "namespace", "girus", "--no-headers", "--ignore-not-found")
	checkOutput, err := checkCmd.Output()
	if err != nil || !strings.Contains(string(checkOutput), "girus") {
		fmt.Fprintf(os.Stderr, "❌ Nenhum cluster Girus ativo encontrado\n")
		fmt.Println("   Use 'girus create cluster' para criar um cluster ou 'girus list clusters' para ver os disponíveis.")
		os.Exit(1)
	}

	// Verificar o pod do backend (silenciosamente, só mostra mensagem em caso de erro)
	backendCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-backend", "-o", "jsonpath={.items[0].status.phase}")
	backendOutput, err := backendCmd.Output()
	if err != nil || string(backendOutput) != "Running" {
		fmt.Fprintf(os.Stderr, "❌ O backend do Girus não está em execução\n")
		fmt.Println("   Verifique o status dos pods com 'kubectl get pods -n girus'")
		os.Exit(1)
	}

	// Ler o arquivo para verificar se é um ConfigMap válido
	content, err := os.ReadFile(labFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Erro ao ler o arquivo '%s': %v\n", labFile, err)
		os.Exit(1)
	}

	// Verificação simples se o arquivo parece ser um ConfigMap válido
	fileContent := string(content)
	if !strings.Contains(fileContent, "kind: ConfigMap") ||
		!strings.Contains(fileContent, "app: girus-lab-template") {
		fmt.Fprintf(os.Stderr, "❌ O arquivo não é um manifesto de laboratório válido\n")
		fmt.Println("   O arquivo deve ser um ConfigMap com a label 'app: girus-lab-template'")
		os.Exit(1)
	}

	// Verificar se está instalando o lab do Docker e se o Docker está disponível
	if strings.Contains(fileContent, "docker-basics") {
		fmt.Println("🐳 Detectado laboratório de Docker, verificando dependências...")

		// Verificar se o Docker está instalado
		dockerCmd := exec.Command("docker", "--version")
		dockerInstalled := dockerCmd.Run() == nil

		// Verificar se o serviço está rodando
		dockerRunning := false
		if dockerInstalled {
			infoCmd := exec.Command("docker", "info")
			dockerRunning = infoCmd.Run() == nil
		}

		if !dockerInstalled || !dockerRunning {
			fmt.Println("⚠️  Aviso: Docker não está instalado ou não está em execução")
			fmt.Println("   O laboratório de Docker será instalado, mas requer Docker para funcionar corretamente.")
			fmt.Println("   Para instalar o Docker:")

			if runtime.GOOS == "darwin" {
				fmt.Println("\n   📦 macOS (via Colima):")
				fmt.Println("      brew install colima docker")
				fmt.Println("      colima start")
			} else if runtime.GOOS == "linux" {
				fmt.Println("\n   📦 Linux:")
				fmt.Println("      curl -fsSL https://get.docker.com | bash")
				fmt.Println("      sudo usermod -aG docker $USER")
				fmt.Println("      sudo systemctl start docker")
			} else {
				fmt.Println("\n   📦 Visite: https://www.docker.com/products/docker-desktop")
			}

			fmt.Println("\n   Você deseja continuar com a instalação do template? [s/N]")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response != "s" && response != "sim" && response != "y" && response != "yes" {
				fmt.Println("Instalação cancelada.")
				os.Exit(0)
			}

			fmt.Println("Continuando com a instalação do template Docker...")
		} else {
			fmt.Println("✅ Docker detectado e funcionando")
		}
	}

	fmt.Printf("📦 Processando laboratório: %s\n", labFile)

	// Aplicar o ConfigMap no cluster usando kubectl apply
	if verboseMode {
		fmt.Println("   Aplicando ConfigMap no cluster...")
	}

	// Aplicar o ConfigMap no cluster
	if verboseMode {
		// Executar normalmente mostrando o output
		applyCmd := exec.Command("kubectl", "apply", "-f", labFile)
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
		if err := applyCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o laboratório: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Usar barra de progresso
		bar := progressbar.NewOptions(100,
			progressbar.OptionSetDescription("   Aplicando laboratório"),
			progressbar.OptionSetWidth(80),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
		)

		// Executar comando sem mostrar saída
		applyCmd := exec.Command("kubectl", "apply", "-f", labFile)
		var stderr bytes.Buffer
		applyCmd.Stderr = &stderr

		// Iniciar o comando
		err := applyCmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar o comando: %v\n", err)
			os.Exit(1)
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
					time.Sleep(50 * time.Millisecond)
				}
			}
		}()

		// Aguardar o final do comando
		err = applyCmd.Wait()
		close(done)
		bar.Finish()

		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Erro ao aplicar o laboratório: %v\n", err)
			if verboseMode {
				fmt.Fprintf(os.Stderr, "   Detalhes: %s\n", stderr.String())
			}
			os.Exit(1)
		}
	}

	// Extrair o ID do lab (name) do arquivo YAML para mostrar na mensagem
	var labID string
	// Procurar pela linha 'name:' dentro do bloco lab.yaml:
	labNameCmd := exec.Command("sh", "-c", fmt.Sprintf("grep -A10 'lab.yaml:' %s | grep 'name:' | head -1", labFile))
	labNameOutput, err := labNameCmd.Output()
	if err == nil {
		nameLine := strings.TrimSpace(string(labNameOutput))
		parts := strings.SplitN(nameLine, "name:", 2)
		if len(parts) >= 2 {
			labID = strings.TrimSpace(parts[1])
		}
	}

	// Extrair também o título para exibição
	var labTitle string
	labTitleCmd := exec.Command("sh", "-c", fmt.Sprintf("grep -A10 'lab.yaml:' %s | grep 'title:' | head -1", labFile))
	labTitleOutput, err := labTitleCmd.Output()
	if err == nil {
		titleLine := strings.TrimSpace(string(labTitleOutput))
		parts := strings.SplitN(titleLine, "title:", 2)
		if len(parts) >= 2 {
			labTitle = strings.TrimSpace(parts[1])
			labTitle = strings.Trim(labTitle, "\"'")
		}
	}

	fmt.Println("\n🔄 Reiniciando backend para carregar o template...")

	// O backend apenas carrega os templates na inicialização
	if verboseMode {
		// Mostrar o output da reinicialização
		fmt.Println("   (O backend do Girus carrega os templates apenas na inicialização)")
		restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
		restartCmd.Stdout = os.Stdout
		restartCmd.Stderr = os.Stderr
		if err := restartCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Erro ao reiniciar o backend: %v\n", err)
			fmt.Println("   O template foi aplicado, mas pode ser necessário reiniciar o backend manualmente:")
			fmt.Println("   kubectl rollout restart deployment/girus-backend -n girus")
		}

		// Aguardar o reinício completar
		fmt.Println("   Aguardando o reinício do backend completar...")
		waitCmd := exec.Command("kubectl", "rollout", "status", "deployment/girus-backend", "-n", "girus", "--timeout=60s")
		// Redirecionar saída para não exibir detalhes do rollout
		var waitOutput bytes.Buffer
		waitCmd.Stdout = &waitOutput
		waitCmd.Stderr = &waitOutput

		// Iniciar indicador de progresso simples
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinIdx := 0
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					fmt.Printf("\r   %s Aguardando... ", spinChars[spinIdx])
					spinIdx = (spinIdx + 1) % len(spinChars)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()

		// Executar e aguardar
		waitCmd.Run()
		close(done)
		fmt.Println("\r   ✅ Backend reiniciado com sucesso!            ")
	} else {
		// Usar barra de progresso
		bar := progressbar.NewOptions(100,
			progressbar.OptionSetDescription("   Reiniciando backend"),
			progressbar.OptionSetWidth(80),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
		)

		// Reiniciar o deployment do backend
		restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
		var stderr bytes.Buffer
		restartCmd.Stderr = &stderr

		err := restartCmd.Run()
		if err != nil {
			bar.Finish()
			fmt.Fprintf(os.Stderr, "\n⚠️  Erro ao reiniciar o backend: %v\n", err)
			if verboseMode {
				fmt.Fprintf(os.Stderr, "   Detalhes: %s\n", stderr.String())
			}
			fmt.Println("   O template foi aplicado, mas pode ser necessário reiniciar o backend manualmente:")
			fmt.Println("   kubectl rollout restart deployment/girus-backend -n girus")
		} else {
			// Aguardar o reinício completar
			waitCmd := exec.Command("kubectl", "rollout", "status", "deployment/girus-backend", "-n", "girus", "--timeout=60s")

			// Redirecionar saída para não exibir detalhes do rollout
			var waitOutput bytes.Buffer
			waitCmd.Stdout = &waitOutput
			waitCmd.Stderr = &waitOutput

			// Iniciar o comando
			err = waitCmd.Start()
			if err != nil {
				bar.Finish()
				fmt.Fprintf(os.Stderr, "\n⚠️  Erro ao verificar status do reinício: %v\n", err)
			} else {
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
				fmt.Println("\r   ✅ Backend reiniciado com sucesso!            ")
			}
			bar.Finish()
		}
	}

	// Aguardar mais alguns segundos para que o backend reinicie completamente
	fmt.Println("   Aguardando inicialização completa...")
	time.Sleep(3 * time.Second)

	// Após reiniciar o backend, verificar se precisamos recriar o port-forward
	portForwardStatus := helpers.CheckPortForwardNeeded()

	// Se port-forward é necessário, configurá-lo corretamente
	if portForwardStatus {
		fmt.Println("\n🔌 Reconfigurando port-forwards após reinício do backend...")

		// Usar a função setupPortForward para garantir que ambos os serviços estejam acessíveis
		err := k8s.SetupPortForward("girus")
		if err != nil {
			fmt.Println("⚠️ Aviso:", err)
			fmt.Println("   Para configurar manualmente, execute:")
			fmt.Println("   kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
			fmt.Println("   kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
		} else {
			fmt.Println("✅ Port-forwards configurados com sucesso!")
			fmt.Println("   🔹 Backend: http://localhost:8080")
			fmt.Println("   🔹 Frontend: http://localhost:8000")
		}
	} else {
		// Verificar conexão com o frontend mesmo que o port-forward não seja necessário
		checkCmd := exec.Command("curl", "-s", "--max-time", "1", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8000")
		var out bytes.Buffer
		checkCmd.Stdout = &out

		if checkCmd.Run() != nil || !strings.Contains(strings.TrimSpace(out.String()), "200") {
			fmt.Println("\n⚠️ Detectado problema na conexão com o frontend.")
			fmt.Println("   Reconfigurando port-forwards para garantir acesso...")

			// Forçar reconfiguração de port-forwards
			err := k8s.SetupPortForward("girus")
			if err != nil {
				fmt.Println("   ⚠️", err)
				fmt.Println("   Configure manualmente: kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
			} else {
				fmt.Println("   ✅ Port-forwards reconfigurados com sucesso!")
			}
		}
	}

	// Desenhar uma linha separadora
	fmt.Println("\n" + strings.Repeat("─", 60))

	// Exibir informações sobre o laboratório adicionado
	fmt.Println("✅ LABORATÓRIO ADICIONADO COM SUCESSO!")

	if labTitle != "" && labID != "" {
		fmt.Printf("\n📚 Título: %s\n", labTitle)
		fmt.Printf("🏷️  ID: %s\n", labID)
	} else if labID != "" {
		fmt.Printf("\n🏷️  ID do Laboratório: %s\n", labID)
	}

	fmt.Println("\n📋 PRÓXIMOS PASSOS:")
	fmt.Println("  • Acesse o Girus no navegador para usar o novo laboratório:")
	fmt.Println("    http://localhost:8000")

	fmt.Println("\n  • Para ver todos os laboratórios disponíveis via CLI:")
	fmt.Println("    girus list labs")

	fmt.Println("\n  • Para verificar detalhes do template adicionado:")
	if labID != "" {
		fmt.Printf("    kubectl describe configmap -n girus | grep -A20 %s\n", labID)
	} else {
		fmt.Println("    kubectl get configmaps -n girus -l app=girus-lab-template")
		fmt.Println("    kubectl describe configmap <nome-do-configmap> -n girus")
	}

	// Linha final
	fmt.Println(strings.Repeat("─", 60))
}
