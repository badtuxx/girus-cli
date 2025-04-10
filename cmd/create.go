package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/badtuxx/girus-cli/internal/helpers"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/badtuxx/girus-cli/internal/lab"
	"github.com/badtuxx/girus-cli/internal/templates"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	deployFile      string
	clusterName     string
	verboseMode     bool
	containerEngine string
	labFile         string
	skipPortForward bool
	skipBrowser     bool
)

var createCmd = &cobra.Command{
	Use:   "create [subcommand]",
	Short: "Comandos para criar recursos",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Cria o cluster Girus",
	Long: `Cria um cluster Kind com o nome "girus" e implanta todos os componentes necessários.
Por padrão, o deployment embutido no binário é utilizado.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Verificar se o containerEngine está instalado e funcionando
		fmt.Println("🔄 Verificando pré-requisitos...")
		containerEngineCmd := exec.Command(containerEngine, "--version")
		if err := containerEngineCmd.Run(); err != nil {
			fmt.Println("❌ " + containerEngine + " não encontrado ou não está em execução")
			fmt.Println("\nO " + containerEngine + " é necessário para criar um cluster Kind. Instruções de instalação:")

			// Detectar o sistema operacional para instruções específicas
			if runtime.GOOS == "darwin" && containerEngine == "docker" {
				// macOS docker
				fmt.Println("\n📦 Para macOS, recomendamos usar Colima (alternativa leve ao Docker Desktop):")
				fmt.Println("1. Instale o Homebrew caso não tenha:")
				fmt.Println("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
				fmt.Println("2. Instale o Colima e o Docker CLI:")
				fmt.Println("   brew install colima docker")
				fmt.Println("3. Inicie o Colima:")
				fmt.Println("   colima start")
				fmt.Println("\nAlternativamente, você pode instalar o Docker Desktop para macOS de:")
				fmt.Println("https://www.docker.com/products/docker-desktop")
			} else if runtime.GOOS == "linux" && containerEngine == "docker" {
				// Linux docker
				fmt.Println("\n📦 Para Linux, use o script de instalação oficial:")
				fmt.Println("   curl -fsSL https://get.docker.com | bash")
				fmt.Println("\nApós a instalação, adicione seu usuário ao grupo docker para evitar usar sudo:")
				fmt.Println("   sudo usermod -aG docker $USER")
				fmt.Println("   newgrp docker")
				fmt.Println("\nE inicie o serviço:")
				fmt.Println("   sudo systemctl enable docker")
				fmt.Println("   sudo systemctl start docker")
			}
			if runtime.GOOS == "darwin" && containerEngine == "podman" {
				// macOS podman
				fmt.Println("\n📦 Para macOS, recomendamos Podman:")
				fmt.Println("1. Instale o Homebrew caso não tenha:")
				fmt.Println("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
				fmt.Println("2. Instale o Podman")
				fmt.Println("   brew install podman")
				fmt.Println("3. Inicie o Podman:")
				fmt.Println("   podman machine init")
				fmt.Println("   podman machine start")
			} else if runtime.GOOS == "linux" && containerEngine == "podman" {
				// Linux podman
				fmt.Println("\n📦 Para Linux, use o script de instalação oficial:")
				fmt.Println("   curl -fsSL https://get.docker.com | bash")
				fmt.Println("\nE inicie o serviço:")
				fmt.Println("   sudo systemctl enable podman")
				fmt.Println("   sudo systemctl start podman")
				fmt.Println("\nOpicional: Após a instalação, para utilizar podman, rootless evitando sudo:")
				fmt.Println("   Siga as instruções do site oficial:")
				fmt.Println("   https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md")
			} else if containerEngine == "podman" {
				// Windows ou outros sistemas
				fmt.Println("\n📦 Visite https://github.com/containers/podman/blob/main/docs/tutorials/podman-for-windows.md para instruções de instalação para seu sistema operacional")
			} else {
				// Windows ou outros sistemas
				fmt.Println("\n📦 Visite https://www.docker.com/products/docker-desktop para instruções de instalação para seu sistema operacional")
			}

			fmt.Println("\nApós instalar o " + containerEngine + " execute novamente este comando.")
			os.Exit(1)
		}

		// Verificar se o serviço containerEngine está rodando
		containerEngineInfoCmd := exec.Command(containerEngine, "info")
		if err := containerEngineInfoCmd.Run(); err != nil {
			fmt.Println("❌ O serviço " + containerEngine + " não está em execução")

			if runtime.GOOS == "darwin" && containerEngine == "docker" {
				fmt.Println("\nPara macOS com Colima:")
				fmt.Println("   colima start")
				fmt.Println("\nPara Docker Desktop:")
				fmt.Println("   Inicie o aplicativo Docker Desktop")
			} else if runtime.GOOS == "darwin" && containerEngine == "podman" {
				fmt.Println("\nPara Podman:")
				fmt.Println("   Inicie a machine com: podman machine start")
			} else if runtime.GOOS == "linux" && containerEngine == "docker" {
				fmt.Println("\nInicie o serviço Docker:")
				fmt.Println("   sudo systemctl start docker")
			} else if runtime.GOOS == "linux" && containerEngine == "podman" {
				fmt.Println("\nInicie o serviço Podman:")
				fmt.Println("   sudo systemctl start podman")
			} else {
				fmt.Println("\nInicie o serviço de containers apropriado para seu sistema.")
			}

			fmt.Println("\nApós iniciar o " + containerEngine + ", execute novamente este comando.")
			os.Exit(1)
		}

		fmt.Println("✅ " + containerEngine + " detectado e funcionando")

		// Verificar silenciosamente se o cluster já existe
		checkCmd := exec.Command("kind", "get", "clusters")
		output, err := checkCmd.Output()

		// Ignorar erros na checagem, apenas assumimos que não há clusters
		if err == nil {
			clusters := strings.Split(strings.TrimSpace(string(output)), "\n")

			// Verificar se o cluster "girus" já existe
			clusterExists := false
			for _, cluster := range clusters {
				if cluster == clusterName {
					clusterExists = true
					break
				}
			}

			if clusterExists {
				fmt.Printf("⚠️  Cluster Girus já existe.\n")
				fmt.Print("Deseja substituí-lo? [s/N]: ")

				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.ToLower(strings.TrimSpace(response))

				if response != "s" && response != "sim" && response != "y" && response != "yes" {
					fmt.Println("Operação cancelada.")
					return
				}

				// Excluir o cluster existente
				fmt.Printf("Excluindo cluster Girus existente...\n")

				deleteCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
				if verboseMode {
					deleteCmd.Stdout = os.Stdout
					deleteCmd.Stderr = os.Stderr
					if err := deleteCmd.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "❌ Erro ao excluir o cluster existente: %v\n", err)
						fmt.Println("   Por favor, exclua manualmente com 'kind delete cluster --name girus' e tente novamente.")
						os.Exit(1)
					}
				} else {
					// Usar barra de progresso
					bar := progressbar.NewOptions(100,
						progressbar.OptionSetDescription("Excluindo cluster existente..."),
						progressbar.OptionSetWidth(80),
						progressbar.OptionShowBytes(false),
						progressbar.OptionSetPredictTime(false),
						progressbar.OptionThrottle(65*time.Millisecond),
						progressbar.OptionSetRenderBlankState(true),
						progressbar.OptionSpinnerType(14),
						progressbar.OptionFullWidth(),
					)

					var stderr bytes.Buffer
					deleteCmd.Stderr = &stderr

					// Iniciar o comando
					err := deleteCmd.Start()
					if err != nil {
						fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar exclusão: %v\n", err)
						os.Exit(1)
					}

					// Atualizar a barra de progresso
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
					err = deleteCmd.Wait()
					close(done)
					bar.Finish()

					if err != nil {
						fmt.Fprintf(os.Stderr, "❌ Erro ao excluir o cluster existente: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderr.String())
						fmt.Println("   Por favor, exclua manualmente com 'kind delete cluster --name girus' e tente novamente.")
						os.Exit(1)
					}
				}

				fmt.Println("✅ Cluster existente excluído com sucesso.")
			}
		}

		// Criar o cluster Kind
		fmt.Println("🔄 Criando cluster Girus...")

		if verboseMode {
			// Executar normalmente mostrando o output
			createClusterCmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
			createClusterCmd.Stdout = os.Stdout
			createClusterCmd.Stderr = os.Stderr

			if err := createClusterCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar o cluster Girus: %v\n", err)
				fmt.Println("   Possíveis causas:")
				fmt.Println("   • " + containerEngine + " não está em execução")
				fmt.Println("   • Permissões insuficientes")
				fmt.Println("   • Conflito com cluster existente")
				os.Exit(1)
			}
		} else {
			// Usando barra de progresso (padrão)
			bar := progressbar.NewOptions(100,
				progressbar.OptionSetDescription("Criando cluster..."),
				progressbar.OptionSetWidth(80),
				progressbar.OptionShowBytes(false),
				progressbar.OptionSetPredictTime(false),
				progressbar.OptionThrottle(65*time.Millisecond),
				progressbar.OptionSetRenderBlankState(true),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionFullWidth(),
			)

			// Executar comando sem mostrar saída
			createClusterCmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
			var stderr bytes.Buffer
			createClusterCmd.Stderr = &stderr

			// Iniciar o comando
			err := createClusterCmd.Start()
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
						time.Sleep(200 * time.Millisecond)
					}
				}
			}()

			// Aguardar o final do comando
			err = createClusterCmd.Wait()
			close(done)
			bar.Finish()

			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar o cluster Girus: %v\n", err)

				// Traduzir mensagens de erro comuns
				errMsg := stderr.String()

				if strings.Contains(errMsg, "node(s) already exist for a cluster with the name") {
					fmt.Println("   Erro: Já existe um cluster com o nome 'girus' no sistema.")
					fmt.Println("   Por favor, exclua-o primeiro com 'kind delete cluster --name girus'")
				} else if strings.Contains(errMsg, "permission denied") {
					fmt.Println("   Erro: Permissão negada. Verifique as permissões do " + containerEngine + ".")
				} else if strings.Contains(errMsg, "Cannot connect to the Docker daemon") {
					fmt.Println("   Erro: Não foi possível conectar ao serviço Docker.")
					fmt.Println("   Verifique se o Docker está em execução com 'systemctl status docker'")
				} else {
					fmt.Println("   Detalhes técnicos:", errMsg)
				}

				os.Exit(1)
			}
		}

		fmt.Println("✅ Cluster Girus criado com sucesso!")

		// Aplicar o manifesto de deployment do Girus
		fmt.Println("\n📦 Implantando o Girus no cluster...")

		// Verificar se existe o arquivo girus-kind-deploy.yaml
		deployYamlPath := "girus-kind-deploy.yaml"
		foundDeployFile := false

		// Verificar em diferentes locais possíveis
		possiblePaths := []string{
			deployYamlPath,                      // No diretório atual
			filepath.Join("..", deployYamlPath), // Um nível acima
			filepath.Join(os.Getenv("HOME"), "REPOS", "strigus", deployYamlPath), // Caminho comum
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				deployFile = path
				foundDeployFile = true
				break
			}
		}

		if foundDeployFile {
			fmt.Printf("🔍 Usando arquivo de deployment: %s\n", deployFile)

			// Aplicar arquivo de deployment completo (já contém o template do lab)
			if verboseMode {
				// Executar normalmente mostrando o output
				applyCmd := exec.Command("kubectl", "apply", "-f", deployFile)
				applyCmd.Stdout = os.Stdout
				applyCmd.Stderr = os.Stderr

				if err := applyCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Usar barra de progresso
				bar := progressbar.NewOptions(100,
					progressbar.OptionSetDescription("Implantando Girus..."),
					progressbar.OptionSetWidth(80),
					progressbar.OptionShowBytes(false),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
				)

				// Executar comando sem mostrar saída
				applyCmd := exec.Command("kubectl", "apply", "-f", deployFile)
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
							time.Sleep(100 * time.Millisecond)
						}
					}
				}()

				// Aguardar o final do comando
				err = applyCmd.Wait()
				close(done)
				bar.Finish()

				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					fmt.Println("   Detalhes técnicos:", stderr.String())
					os.Exit(1)
				}
			}

			fmt.Println("✅ Infraestrutura e template de laboratório aplicados com sucesso!")
		} else {
			// Usar o deployment embutido como fallback
			// fmt.Println("⚠️  Arquivo girus-kind-deploy.yaml não encontrado, usando deployment embutido.")

			// Criar um arquivo temporário para o deployment principal
			tempFile, err := os.CreateTemp("", "girus-deploy-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário: %v\n", err)
				os.Exit(1)
			}
			defer os.Remove(tempFile.Name()) // Limpar o arquivo temporário ao finalizar

			defaultDeployment, err := templates.GetManifest("defaultDeployment.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}

			// Escrever o conteúdo no arquivo temporário
			if _, err := tempFile.WriteString(string(defaultDeployment)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever no arquivo temporário: %v\n", err)
				os.Exit(1)
			}
			tempFile.Close()

			// Aplicar o deployment principal
			if verboseMode {
				// Executar normalmente mostrando o output
				applyCmd := exec.Command("kubectl", "apply", "-f", tempFile.Name())
				applyCmd.Stdout = os.Stdout
				applyCmd.Stderr = os.Stderr

				if err := applyCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Usar barra de progresso para o deploy (padrão)
				bar := progressbar.NewOptions(100,
					progressbar.OptionSetDescription("Implantando infraestrutura..."),
					progressbar.OptionSetWidth(80),
					progressbar.OptionShowBytes(false),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
				)

				// Executar comando sem mostrar saída
				applyCmd := exec.Command("kubectl", "apply", "-f", tempFile.Name())
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
							time.Sleep(100 * time.Millisecond)
						}
					}
				}()

				// Aguardar o final do comando
				err = applyCmd.Wait()
				close(done)
				bar.Finish()

				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					fmt.Println("   Detalhes técnicos:", stderr.String())
					os.Exit(1)
				}
			}

			fmt.Println("✅ Infraestrutura básica aplicada com sucesso!")

			// Agora vamos aplicar o template de laboratório que está embutido no binário
			fmt.Println("\n🔬 Aplicando templates de laboratório...")

			// Listar todos os arquivos YAML dentro de manifests/
			manifestFiles, err := templates.ListManifests()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao listar templates embutidos: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
			} else if len(manifestFiles) == 0 {
				fmt.Println("   ⚠️ Nenhum template de laboratório embutido encontrado.")
			} else {
				// Temos templates para aplicar
				if verboseMode {
					// Modo detalhado: Aplicar cada template individualmente mostrando logs
					fmt.Printf("   Encontrados %d templates para aplicar:\n", len(manifestFiles))
					allTemplatesApplied := true
					for _, manifestName := range manifestFiles {
						fmt.Printf("   - Aplicando %s...\n", manifestName)
						// Ler o conteúdo do manifesto
						manifestContent, err := templates.GetManifest(manifestName)
						if err != nil {
							fmt.Fprintf(os.Stderr, "     ❌ Erro ao carregar o template %s: %v\n", manifestName, err)
							allTemplatesApplied = false
							continue
						}

						// Criar arquivo temporário
						tempLabFile, err := os.CreateTemp("", "girus-template-*.yaml")
						if err != nil {
							fmt.Fprintf(os.Stderr, "     ❌ Erro ao criar arquivo temporário para %s: %v\n", manifestName, err)
							allTemplatesApplied = false
							continue
						}
						tempPath := tempLabFile.Name() // Guardar o path antes de fechar

						// Escrever e fechar arquivo temporário
						if _, err := tempLabFile.Write(manifestContent); err != nil {
							fmt.Fprintf(os.Stderr, "     ❌ Erro ao escrever template %s no arquivo temporário: %v\n", manifestName, err)
							tempLabFile.Close() // Fechar mesmo em caso de erro
							os.Remove(tempPath) // Remover o temporário
							allTemplatesApplied = false
							continue
						}
						tempLabFile.Close()

						// Aplicar com kubectl
						applyCmd := exec.Command("kubectl", "apply", "-f", tempPath)
						applyCmd.Stdout = os.Stdout
						applyCmd.Stderr = os.Stderr
						if err := applyCmd.Run(); err != nil {
							fmt.Fprintf(os.Stderr, "     ❌ Erro ao aplicar o template %s: %v\n", manifestName, err)
							allTemplatesApplied = false
						} else {
							fmt.Printf("     ✅ Template %s aplicado com sucesso!\n", manifestName)
						}
						os.Remove(tempPath) // Remover o temporário após o uso
					}

					if allTemplatesApplied {
						fmt.Println("✅ Todos os templates de laboratório embutidos aplicados com sucesso!")
					} else {
						fmt.Println("⚠️ Alguns templates de laboratório não puderam ser aplicados.")
					}

				} else {
					// Modo com barra de progresso: Aplicar cada template individualmente
					bar := progressbar.NewOptions(len(manifestFiles),
						progressbar.OptionSetDescription("Aplicando templates de laboratório..."),
						progressbar.OptionSetWidth(80),
						progressbar.OptionShowCount(),
						progressbar.OptionSetPredictTime(false),
						progressbar.OptionThrottle(65*time.Millisecond),
						progressbar.OptionSetRenderBlankState(true),
						progressbar.OptionSpinnerType(14),
						progressbar.OptionFullWidth(),
					)

					allSuccess := true
					for _, manifestName := range manifestFiles {
						// Ler o conteúdo do manifesto
						manifestContent, err := templates.GetManifest(manifestName)
						if err != nil {
							bar.Add(1) // Incrementar a barra mesmo com erro
							allSuccess = false
							continue
						}

						// Criar arquivo temporário
						tempLabFile, err := os.CreateTemp("", "girus-template-*.yaml")
						if err != nil {
							bar.Add(1) // Incrementar a barra mesmo com erro
							allSuccess = false
							continue
						}
						tempPath := tempLabFile.Name()

						// Escrever e fechar arquivo temporário
						if _, err := tempLabFile.Write(manifestContent); err != nil {
							tempLabFile.Close()
							os.Remove(tempPath)
							bar.Add(1) // Incrementar a barra mesmo com erro
							allSuccess = false
							continue
						}
						tempLabFile.Close()

						// Aplicar com kubectl
						applyCmd := exec.Command("kubectl", "apply", "-f", tempPath)
						var stderr bytes.Buffer
						applyCmd.Stderr = &stderr
						if err := applyCmd.Run(); err != nil {
							os.Remove(tempPath)
							bar.Add(1) // Incrementar a barra mesmo com erro
							allSuccess = false
							continue
						}

						os.Remove(tempPath)
						bar.Add(1) // Incrementar a barra após sucesso
					}
					bar.Finish()

					if allSuccess {
						fmt.Println("✅ Todos os templates de laboratório aplicados com sucesso!")
					} else {
						fmt.Println("⚠️ Alguns templates de laboratório não puderam ser aplicados. Use --verbose para detalhes.")
					}

					// Verificação de diagnóstico para confirmar que os templates estão visíveis
					fmt.Println("\n🔍 Verificando templates de laboratório instalados:")
					listLabsCmd := exec.Command("kubectl", "get", "configmap", "-n", "girus", "-l", "app=girus-lab-template", "-o", "custom-columns=NAME:.metadata.name")
					var labsOutput bytes.Buffer
					listLabsCmd.Stdout = &labsOutput
					listLabsCmd.Stderr = &labsOutput

					if err := listLabsCmd.Run(); err == nil {
						labs := strings.Split(strings.TrimSpace(labsOutput.String()), "\n")
						if len(labs) > 1 { // Primeira linha é o cabeçalho "NAME"
							fmt.Println("   Templates encontrados:")
							for i, lab := range labs {
								if i > 0 { // Pular o cabeçalho
									fmt.Printf("   ✅ %s\n", strings.TrimSpace(lab))
								}
							}
						} else {
							fmt.Println("   ⚠️ Nenhum template de laboratório encontrado!")
						}
					} else {
						fmt.Println("   ⚠️ Não foi possível verificar os templates instalados")
					}
				}

				// Reiniciar o backend para carregar os templates
				fmt.Println("\n🔄 Reiniciando o backend para carregar os templates...")
				restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
				restartCmd.Run()

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

				// Aguardar mais alguns segundos para o backend inicializar completamente
				fmt.Println("   Aguardando inicialização completa...")
				time.Sleep(5 * time.Second)
			}
		}

		// Aguardar os pods do Girus ficarem prontos
		if err := k8s.WaitForPodsReady("girus", 5*time.Minute); err != nil {
			fmt.Fprintf(os.Stderr, "Aviso: %v\n", err)
			fmt.Println("Recomenda-se verificar o estado dos pods com 'kubectl get pods -n girus'")
		} else {
			fmt.Println("Todos os componentes do Girus estão prontos e em execução!")
		}

		fmt.Println("Girus implantado com sucesso no cluster!")

		// Configurar port-forward automaticamente (a menos que --skip-port-forward tenha sido especificado)
		if !skipPortForward {
			fmt.Print("\n🔌 Configurando acesso aos serviços do Girus... ")

			if err := k8s.SetupPortForward("girus"); err != nil {
				fmt.Println("⚠️")
				fmt.Printf("Não foi possível configurar o acesso automático: %v\n", err)
				fmt.Println("\nVocê pode tentar configurar manualmente com os comandos:")
				fmt.Println("kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
				fmt.Println("kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
			} else {
				fmt.Println("✅")
				fmt.Println("Acesso configurado com sucesso!")
				fmt.Println("📊 Backend: http://localhost:8080")
				fmt.Println("🖥️  Frontend: http://localhost:8000")

				// Abrir o navegador se não foi especificado para pular
				if !skipBrowser {
					fmt.Println("\n🌐 Abrindo navegador com o Girus...")
					if err := helpers.OpenBrowser("http://localhost:8000"); err != nil {
						fmt.Printf("⚠️  Não foi possível abrir o navegador: %v\n", err)
						fmt.Println("   Acesse manualmente: http://localhost:8000")
					}
				}
			}
		} else {
			fmt.Println("\n⏩ Port-forward ignorado conforme solicitado")
			fmt.Println("\nPara acessar o Girus posteriormente, execute:")
			fmt.Println("kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
			fmt.Println("kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
		}

		// Exibir mensagem de conclusão
		fmt.Println("\n" + strings.Repeat("─", 60))
		fmt.Println("✅ GIRUS PRONTO PARA USO!")
		fmt.Println(strings.Repeat("─", 60))

		// Exibir acesso ao navegador como próximo passo
		fmt.Println("📋 PRÓXIMOS PASSOS:")
		fmt.Println("  • Acesse o Girus no navegador:")
		fmt.Println("    http://localhost:8000")

		// Instruções para laboratórios
		fmt.Println("\n  • Para aplicar mais templates de laboratórios com o Girus:")
		fmt.Println("    girus create lab -f caminho/para/lab.yaml")

		fmt.Println("\n  • Para ver todos os laboratórios disponíveis:")
		fmt.Println("    girus list labs")

		fmt.Println(strings.Repeat("─", 60))
	},
}

var createLabCmd = &cobra.Command{
	Use:   "lab [lab-id] ou -f [arquivo]",
	Short: "Cria um novo laboratório no Girus",
	Long:  "Adiciona um novo laboratório ao Girus a partir de um arquivo de manifesto ConfigMap, ou cria um ambiente de laboratório a partir de um ID de template existente.\nOs templates de laboratório são armazenados no diretório /labs na raiz do projeto.",
	Run: func(cmd *cobra.Command, args []string) {
		// Verificar qual modo estamos
		if labFile != "" {
			// Modo de adicionar template a partir de arquivo
			lab.AddLabFromFile(labFile, verboseMode)
		} else {
			fmt.Fprintf(os.Stderr, "Erro: Você deve especificar um arquivo de laboratório com a flag -f\n")
			fmt.Println("\nExemplo:")
			fmt.Println("  girus create lab -f meulaboratorio.yaml      # Adiciona um novo template a partir do arquivo")
			fmt.Println("  girus create lab -f /home/user/REPOS/strigus/labs/basic-linux.yaml      # Adiciona um template do diretório /labs")
			os.Exit(1)
		}
	},
}

func init() {
	createCmd.AddCommand(createClusterCmd)
	createCmd.AddCommand(createLabCmd)

	// Flags para createClusterCmd
	createClusterCmd.Flags().StringVarP(&deployFile, "file", "f", "", "Arquivo YAML para deployment do Girus (opcional)")
	createClusterCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Modo detalhado com output completo em vez da barra de progresso")
	createClusterCmd.Flags().BoolVarP(&skipPortForward, "skip-port-forward", "", false, "Não perguntar sobre configurar port-forwarding")
	createClusterCmd.Flags().BoolVarP(&skipBrowser, "skip-browser", "", false, "Não abrir o navegador automaticamente")

	createClusterCmd.Flags().StringVarP(&containerEngine, "container-engine", "e", "docker", "Engine de container (docker ou podman)")

	// Flags para createLabCmd
	createLabCmd.Flags().StringVarP(&labFile, "file", "f", "", "Arquivo de manifesto do laboratório (ConfigMap)")
	createLabCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Modo detalhado com output completo em vez da barra de progresso")

	// definir o nome do cluster como "girus" sempre
	clusterName = "girus"
}
