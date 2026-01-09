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

	"github.com/badtuxx/girus-cli/internal/common"
	"github.com/badtuxx/girus-cli/internal/helpers"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/badtuxx/girus-cli/internal/lab"
	"github.com/badtuxx/girus-cli/internal/repo"
	"github.com/badtuxx/girus-cli/internal/templates"
	"github.com/fatih/color"
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
	repoIndexURL    string
)

var createCmd = &cobra.Command{
	Use:   "create [subcommand]",
	Short: common.T("Comandos para criar recursos", "Comandos para crear recursos"),
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
		// Criar formatadores de cores
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		magenta := color.New(color.FgMagenta).SprintFunc()
		headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()

		// Exibir cabeçalho
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println(headerColor(common.T("GIRUS CREATE", "GIRUS CREAR")))
		fmt.Println(strings.Repeat("─", 80))

		// Verificar se há atualização disponível para o CLI
		fmt.Println(headerColor(common.T("Verificando atualizações...", "Verificando actualizaciones...")))

		currentVersion := common.Version

		latestVersion, err := GetLatestGitHubVersion("badtuxx/girus-cli")

		if err == nil && IsNewerVersion(latestVersion, currentVersion) {
			fmt.Printf(common.T("%s versão %s disponível (atual: %s)\n", "%s versión %s disponible (actual: %s)\n"), yellow("AVISO:"), magenta(latestVersion), magenta(currentVersion))
			fmt.Print(common.T("Deseja atualizar antes de criar o cluster? [S/n]: ", "¿Desea actualizar antes de crear el cluster? [S/n]: "))

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "" || response == "s" || response == "sim" || response == "y" || response == "yes" {
				// Criar comando de atualização
				updateCmd := exec.Command("girus", "update")
				updateCmd.Stdout = os.Stdout
				updateCmd.Stderr = os.Stderr
				updateCmd.Stdin = os.Stdin

				if err := updateCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "%s erro ao executar atualização: %v\n", red("ERRO:"), err)
					fmt.Println(common.T("Continuando com a versão atual...", "Continuando con la versión actual..."))
				} else {
					fmt.Printf(common.T("%s Atualização concluída. Por favor, execute o comando novamente.\n", "%s Actualización completada. Por favor, ejecute el comando de nuevo.\n"), green("SUCESSO:"))
					os.Exit(0)
				}
			}
		}

		// Verificar se o containerEngine está instalado e funcionando
		fmt.Println("\n" + headerColor(common.T("Verificando pré-requisitos...", "Verificando requisitos previos...")))
		containerEngineCmd := exec.Command(containerEngine, "--version")
		if err := containerEngineCmd.Run(); err != nil {
			fmt.Printf(common.T("%s %s não encontrado ou não está em execução\n", "%s %s no encontrado o no está en ejecución\n"), red("ERRO:"), containerEngine)
			fmt.Println(common.T("\nO "+containerEngine+" é necessário para criar um cluster Kind. Instruções de instalação:", "\n"+containerEngine+" es necesario para crear un cluster Kind. Instrucciones de instalación:"))

			// Detectar o sistema operacional para instruções específicas
			if runtime.GOOS == "darwin" && containerEngine == "docker" {
				// macOS docker
				fmt.Println("\nPara macOS, recomendamos usar Colima (alternativa leve ao Docker Desktop):")
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
				fmt.Println("\nPara Linux, use o script de instalação oficial:")
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
				fmt.Println("\nPara macOS, recomendamos Podman:")
				fmt.Println("1. Instale o Homebrew caso não tenha:")
				fmt.Println("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
				fmt.Println("2. Instale o Podman")
				fmt.Println("   brew install podman")
				fmt.Println("3. Inicie o Podman:")
				fmt.Println("   podman machine init")
				fmt.Println("   podman machine start")
			} else if runtime.GOOS == "linux" && containerEngine == "podman" {
				// Linux podman
				fmt.Println("\nPara Linux, use o script de instalação oficial:")
				fmt.Println("   curl -fsSL https://get.docker.com | bash")
				fmt.Println("\nE inicie o serviço:")
				fmt.Println("   sudo systemctl enable podman")
				fmt.Println("   sudo systemctl start podman")
				fmt.Println("\nOpcional: Após a instalação, para utilizar podman, rootless evitando sudo:")
				fmt.Println("   Siga as instruções do site oficial:")
				fmt.Println("   https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md")
			} else if containerEngine == "podman" {
				// Windows ou outros sistemas
				fmt.Println("\nVisite https://github.com/containers/podman/blob/main/docs/tutorials/podman-for-windows.md para instruções de instalação para seu sistema operacional")
			} else {
				// Windows ou outros sistemas
				fmt.Println("\nVisite https://www.docker.com/products/docker-desktop para instruções de instalação para seu sistema operacional")
			}

			fmt.Println("\nApós instalar o " + containerEngine + " execute novamente este comando.")
			os.Exit(1)
		}

		// Verificar se o serviço containerEngine está rodando
		containerEngineInfoCmd := exec.Command(containerEngine, "info")
		if err := containerEngineInfoCmd.Run(); err != nil {
			fmt.Printf(common.T("%s O serviço %s não está em execução\n", "%s El servicio %s no está en ejecución\n"), red("ERRO:"), containerEngine)

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

		fmt.Printf("%s %s detectado e funcionando\n", green("ATIVO"), magenta(containerEngine))

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
				fmt.Printf("%s %s\n", yellow(common.T("AVISO:", "AVISO:")), common.T("Cluster Girus já existe.", "El cluster Girus ya existe."))
				fmt.Print(common.T("Deseja substituí-lo? [s/N]: ", "¿Desea reemplazarlo? [s/N]: "))

				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.ToLower(strings.TrimSpace(response))

				if response != "s" && response != "sim" && response != "y" && response != "yes" {
					fmt.Println(common.T("Operação cancelada.", "Operación cancelada."))
					return
				}

				// Excluir o cluster existente
				fmt.Println(headerColor("Excluindo cluster Girus existente..."))

				deleteCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
				if verboseMode {
					deleteCmd.Stdout = os.Stdout
					deleteCmd.Stderr = os.Stderr
					if err := deleteCmd.Run(); err != nil {
						fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao excluir o cluster existente: %v\n", err)
						fmt.Println("   Por favor, exclua manualmente com 'kind delete cluster --name girus' e tente novamente.")
						os.Exit(1)
					}
				} else {
					// Usar barra de progresso
					barConfig := helpers.ProgressBarConfig{
						Total:            100,
						Description:      "Excluindo cluster existente...",
						Width:            80,
						Throttle:         65,
						SpinnerType:      15,
						RenderBlankState: true,
						ShowBytes:        false,
						SetPredictTime:   false,
					}
					bar := helpers.CreateProgressBar(barConfig)

					var stderr bytes.Buffer
					deleteCmd.Stderr = &stderr

					// Iniciar o comando
					err := deleteCmd.Start()
					if err != nil {
						fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao iniciar exclusão: %v\n", err)
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
						fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao excluir o cluster existente: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderr.String())
						fmt.Println("   Por favor, exclua manualmente com 'kind delete cluster --name girus' e tente novamente.")
						os.Exit(1)
					}
				}

				fmt.Println("\n" + green(common.T("SUCESSO:", "ÉXITO:")) + " " + common.T("Cluster existente excluído com sucesso.", "Cluster existente eliminado con éxito."))
			}
		}

		// Criar o cluster Kind
		fmt.Println("\n" + headerColor(common.T("Criando cluster Girus...", "Creando cluster Girus...")))

		if verboseMode {
			// Executar normalmente mostrando o output
			createClusterCmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
			createClusterCmd.Stdout = os.Stdout
			createClusterCmd.Stderr = os.Stderr

			if err := createClusterCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao criar o cluster Girus: %v\n", err)
				fmt.Println("   Possíveis causas:")
				fmt.Println("   • " + bold(containerEngine) + " não está em execução")
				fmt.Println("   • Permissões insuficientes")
				fmt.Println("   • Conflito com cluster existente")
				os.Exit(1)
			}
		} else {
			// Usar barra de progresso (padrão)
			barConfig := helpers.ProgressBarConfig{
				Total:            100,
				Description:      common.T("Criando cluster...", "Creando cluster..."),
				Width:            80,
				Throttle:         65,
				SpinnerType:      14,
				RenderBlankState: true,
				ShowBytes:        false,
				SetPredictTime:   false,
			}
			bar := helpers.CreateProgressBar(barConfig)

			// Executar comando sem mostrar saída
			createClusterCmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
			var stderr bytes.Buffer
			createClusterCmd.Stderr = &stderr

			// Iniciar o comando
			err := createClusterCmd.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao iniciar o comando: %v\n", err)
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
				fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao criar o cluster Girus: %v\n", err)

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

		fmt.Println("\n" + green("SUCESSO:") + " Cluster Girus criado com sucesso!")

		// Aplicar o manifesto de deployment do Girus
		fmt.Println("\n" + headerColor("Implantando o Girus no cluster..."))

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
			fmt.Printf("%s Usando arquivo de deployment: %s\n", cyan("INFO:"), magenta(deployFile))

			// Aplicar arquivo de deployment completo (já contém o template do lab)
			if verboseMode {
				// Executar normalmente mostrando o output
				applyCmd := exec.Command("kubectl", "apply", "-f", deployFile)
				applyCmd.Stdout = os.Stdout
				applyCmd.Stderr = os.Stderr

				if err := applyCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao aplicar o manifesto do Girus: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Usar barra de progresso
				barConfig := helpers.ProgressBarConfig{
					Total:            100,
					Description:      "Implantando Girus...",
					Width:            80,
					Throttle:         65,
					SpinnerType:      14,
					RenderBlankState: true,
					ShowBytes:        false,
					SetPredictTime:   false,
				}
				bar := helpers.CreateProgressBar(barConfig)

				// Executar comando sem mostrar saída
				applyCmd := exec.Command("kubectl", "apply", "-f", deployFile)
				var stderr bytes.Buffer
				applyCmd.Stderr = &stderr

				// Iniciar o comando
				err := applyCmd.Start()
				if err != nil {
					fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao iniciar o comando: %v\n", err)
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
					fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao aplicar o manifesto do Girus: %v\n", err)
					fmt.Println("   Detalhes técnicos:", stderr.String())
					os.Exit(1)
				}
			}

			fmt.Println("\n" + green(common.T("SUCESSO:", "ÉXITO:")) + " " + common.T("Infraestrutura e template de laboratório aplicados com sucesso!", "¡Infraestructura y plantilla de laboratorio aplicadas con éxito!"))
		} else {
			// Usar o deployment embutido como fallback
			// fmt.Println("⚠️  Arquivo girus-kind-deploy.yaml não encontrado, usando deployment embutido.")

			// Criar um arquivo temporário para o deployment principal
			tempFile, err := os.CreateTemp("", "girus-deploy-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao criar arquivo temporário: %v\n", err)
				os.Exit(1)
			}
			defer os.Remove(tempFile.Name()) // Limpar o arquivo temporário ao finalizar

			defaultDeployment, err := templates.GetManifest("defaultDeployment.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao carregar o template: %v\n", err)
				return
			}

			// Escrever o conteúdo no arquivo temporário
			if _, err := tempFile.WriteString(string(defaultDeployment)); err != nil {
				fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao escrever no arquivo temporário: %v\n", err)
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
					fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao aplicar o manifesto do Girus: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Usar barra de progresso para o deploy (padrão)
				barConfig := helpers.ProgressBarConfig{
					Total:            100,
					Description:      "Implantando infraestrutura...",
					Width:            80,
					Throttle:         65,
					SpinnerType:      14,
					RenderBlankState: true,
					ShowBytes:        false,
					SetPredictTime:   false,
				}
				bar := helpers.CreateProgressBar(barConfig)

				// Executar comando sem mostrar saída
				applyCmd := exec.Command("kubectl", "apply", "-f", tempFile.Name())
				var stderr bytes.Buffer
				applyCmd.Stderr = &stderr

				// Iniciar o comando
				err := applyCmd.Start()
				if err != nil {
					fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao iniciar o comando: %v\n", err)
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
					fmt.Fprintf(os.Stderr, red("ERRO:")+" Erro ao aplicar o manifesto do Girus: %v\n", err)
					fmt.Println("   Detalhes técnicos:", stderr.String())
					os.Exit(1)
				}
			}

			fmt.Println("\n" + green(common.T("SUCESSO:", "ÉXITO:")) + " " + common.T("Infraestrutura básica aplicada com sucesso!", "¡Infraestructura básica aplicada con éxito!"))

			// Agora vamos aplicar o template de laboratório que está embutido no binário
			fmt.Println("\n" + headerColor(common.T("Aplicando templates de laboratório...", "Aplicando plantillas de laboratorio...")))

			// Listar todos os arquivos YAML dentro de manifests/
			manifestFiles, err := templates.ListManifests()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s Erro ao listar templates embutidos: %v\n", red("ERRO:"), err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
			} else if len(manifestFiles) == 0 {
				fmt.Printf("   %s Nenhum template de laboratório embutido encontrado.\n", yellow("AVISO:"))
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
							fmt.Fprintf(os.Stderr, "     %s Erro ao carregar o template %s: %v\n", red("ERRO:"), manifestName, err)
							allTemplatesApplied = false
							continue
						}

						// Criar arquivo temporário
						tempLabFile, err := os.CreateTemp("", "girus-template-*.yaml")
						if err != nil {
							fmt.Fprintf(os.Stderr, "     %s Erro ao criar arquivo temporário para %s: %v\n", red("ERRO:"), manifestName, err)
							allTemplatesApplied = false
							continue
						}
						tempPath := tempLabFile.Name() // Guardar o path antes de fechar

						// Escrever e fechar arquivo temporário
						if _, err := tempLabFile.Write(manifestContent); err != nil {
							fmt.Fprintf(os.Stderr, "     %s Erro ao escrever template %s no arquivo temporário: %v\n", red("ERRO:"), manifestName, err)
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
							fmt.Fprintf(os.Stderr, "     %s Erro ao aplicar o template %s: %v\n", red("ERRO:"), manifestName, err)
							allTemplatesApplied = false
						} else {
							fmt.Printf("     %s Template %s aplicado com sucesso!\n", green("SUCESSO:"), manifestName)
						}
						os.Remove(tempPath) // Remover o temporário após o uso
					}

					if allTemplatesApplied {
						fmt.Printf(common.T("%s Todos os templates de laboratório embutidos aplicados com sucesso!\n", "%s ¡Todas las plantillas de laboratorio embebidas aplicadas con éxito!\n"), green(common.T("SUCESSO:", "ÉXITO:")))
					} else {
						fmt.Printf(common.T("%s Alguns templates de laboratório não puderam ser aplicados.\n", "%s Algunas plantillas de laboratorio no se pudieron aplicar.\n"), yellow(common.T("AVISO:", "AVISO:")))
					}

				} else {
					// Modo com barra de progresso: Aplicar cada template individualmente
					barConfig := helpers.ProgressBarConfig{
						Total:            len(manifestFiles),
						Description:      common.T("Aplicando templates de laboratório...", "Aplicando plantillas de laboratorio..."),
						Width:            80,
						Throttle:         65,
						SpinnerType:      14,
						RenderBlankState: true,
						ShowBytes:        false,
						SetPredictTime:   false,
					}
					bar := helpers.CreateProgressBar(barConfig)

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
						fmt.Println("\n" + green(common.T("SUCESSO:", "ÉXITO:")) + " " + common.T("Todos os templates de laboratório aplicados com sucesso!", "¡Todas las plantillas de laboratorio aplicadas con éxito!"))
					} else {
						fmt.Println("\n" + yellow(common.T("AVISO:", "AVISO:")) + " " + common.T("Alguns templates de laboratório não puderam ser aplicados. Use --verbose para detalhes.", "Algunas plantillas de laboratorio no se pudieron aplicar. Use --verbose para más detalles."))
					}

					// Verificação de diagnóstico para confirmar que os templates estão visíveis
					fmt.Println("\n" + headerColor(common.T("Verificando templates de laboratório instalados:", "Verificando plantillas de laboratorio instaladas:")))
					listLabsCmd := exec.Command("kubectl", "get", "configmap", "-n", "girus", "-l", "app=girus-lab-template", "-o", "custom-columns=NAME:.metadata.name")
					var labsOutput bytes.Buffer
					listLabsCmd.Stdout = &labsOutput
					listLabsCmd.Stderr = &labsOutput

					if err := listLabsCmd.Run(); err == nil {
						labs := strings.Split(strings.TrimSpace(labsOutput.String()), "\n")
						if len(labs) > 1 { // Primeira linha é o cabeçalho "NAME"
							fmt.Println(common.T("   Templates encontrados:", "   Plantillas encontradas:"))
							for i, lab := range labs {
								if i > 0 { // Pular o cabeçalho
									fmt.Printf("   %s %s\n", green("ATIVO"), strings.TrimSpace(lab))
								}
							}
						} else {
							fmt.Printf("   %s %s\n", yellow(common.T("AVISO:", "AVISO:")), common.T("Nenhum template de laboratório encontrado!", "¡Ninguna plantilla de laboratorio encontrada!"))
						}
					} else {
						fmt.Printf("   %s %s\n", yellow(common.T("AVISO:", "AVISO:")), common.T("Não foi possível verificar os templates instalados", "No fue posible verificar las plantillas instaladas"))
					}
				}

				// Reiniciar o backend para carregar os templates
				fmt.Println("\n" + headerColor(common.T("Reiniciando o backend para carregar os templates...", "Reiniciando el backend para cargar las plantillas...")))
				restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
				restartCmd.Run()

				// Aguardar o reinício completar
				fmt.Println(common.T("   Aguardando o reinício do backend completar...", "   Esperando a que el backend reinicie..."))
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
				fmt.Printf("\r   %s %s            \n", green(common.T("SUCESSO:", "ÉXITO:")), common.T("Backend reiniciado com sucesso!", "¡Backend reiniciado con éxito!"))

				// Aguardar mais alguns segundos para o backend inicializar completamente
				fmt.Println(common.T("   Aguardando inicialização completa...", "   Esperando a que la inicialización complete..."))
				time.Sleep(5 * time.Second)
			}
		}

		// Aguardar os pods do Girus ficarem prontos
		if err := k8s.WaitForPodsReady("girus", 5*time.Minute); err != nil {
			fmt.Fprintf(os.Stderr, "%s %v\n", yellow("AVISO:"), err)
			fmt.Println("Recomenda-se verificar o estado dos pods com 'kubectl get pods -n girus'")
		} else {
			fmt.Printf("%s Todos os componentes do Girus estão prontos e em execução!\n", green("SUCESSO:"))
		}

		fmt.Printf("%s Girus implantado com sucesso no cluster!\n", green("SUCESSO:"))

		// Configurar port-forward automaticamente (a menos que --skip-port-forward tenha sido especificado)
		if !skipPortForward {
			fmt.Print("\n" + headerColor(common.T("Configurando acesso aos serviços do Girus...", "Configurando el acceso a los servicios de Girus...")) + " ")

			if err := k8s.SetupPortForward("girus"); err != nil {
				fmt.Printf("%s\n", yellow(common.T("AVISO:", "AVISO:")))
				fmt.Printf(common.T("%s Não foi possível configurar o acesso automático: %v\n", "%s No fue posible configurar el acceso automático: %v\n"), yellow(common.T("AVISO:", "AVISO:")), err)
				fmt.Println(common.T("\nVocê pode tentar configurar manualmente com os comandos:", "\nPuede intentar configurar manualmente con los comandos:"))
				fmt.Println("kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
				fmt.Println("kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
			} else {
				fmt.Printf("%s\n", green(common.T("SUCESSO:", "ÉXITO:")))
				fmt.Println(common.T("Acesso configurado com sucesso!", "¡Acceso configurado con éxito!"))
				fmt.Println(bold("Backend:") + " http://localhost:8080")
				fmt.Println(bold("Frontend:") + " http://localhost:8000")

				// Abrir o navegador se não foi especificado para pular
				if !skipBrowser {
					fmt.Println("\n" + headerColor("Abrindo navegador com o Girus..."))
					if err := helpers.OpenBrowser("http://localhost:8000"); err != nil {
						fmt.Printf("%s Não foi possível abrir o navegador: %v\n", yellow("AVISO:"), err)
						fmt.Println("   Acesse manualmente: http://localhost:8000")
					}
				}
			}
		} else {
			fmt.Println("\n" + yellow(common.T("AVISO:", "AVISO:")) + " " + common.T("Port-forward ignorado conforme solicitado", "Port-forward ignorado según lo solicitado"))
			fmt.Println(common.T("\nPara acessar o Girus posteriormente, execute:", "\nPara acceder a Girus más tarde, ejecute:"))
			fmt.Println("kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
			fmt.Println("kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
		}

		// Inicializar o arquivo de progresso com todos os laboratórios como "not-started"
		fmt.Println("\n" + headerColor("Inicializando arquivo de progresso..."))
		labNames, err := lab.ListAllLabs()
		if err != nil {
			fmt.Printf("%s Erro ao listar laboratórios: %v\n", yellow("AVISO:"), err)
		} else {
			progress := common.NewProgress("")
			for _, name := range labNames {
				progress.AddLab(name, "not-started")
			}
			if err := progress.SaveProgressToFile(); err != nil {
				fmt.Printf("%s Erro ao salvar arquivo de progresso: %v\n", yellow("AVISO:"), err)
			} else {
				fmt.Printf("%s Arquivo de progresso inicializado com %d laboratórios.\n", green("SUCESSO:"), len(labNames))
			}
		}

		// Exibir mensagem de conclusão
		fmt.Println("\n" + strings.Repeat("─", 60))
		fmt.Println(headerColor(common.T("GIRUS PRONTO PARA USO!", "GIRUS LISTO PARA USARSE!")))
		fmt.Println(strings.Repeat("─", 60))

		// Exibir acesso ao navegador como próximo passo
		fmt.Println(bold(common.T("PRÓXIMOS PASSOS:", "PRÓXIMOS PASOS:")))
		fmt.Println(common.T("  • Acesse o Girus no navegador:", "  • Acceda a Girus en el navegador:"))
		fmt.Println("    http://localhost:8000")

		// Instruções para laboratórios
		fmt.Println(common.T("\n  • Para aplicar mais templates de laboratórios com o Girus:", "\n  • Para aplicar más plantillas de laboratorio con Girus:"))
		fmt.Println(common.T("    girus create lab -f caminho/para/lab.yaml", "    girus create lab -f ruta/al/lab.yaml"))

		fmt.Println(common.T("\n  • Para ver todos os laboratórios disponíveis:", "\n  • Para ver todos los laboratorios disponibles:"))
		fmt.Println("    girus list labs")

		fmt.Println(strings.Repeat("─", 60))
	},
}

var createLabCmd = &cobra.Command{
	Use:   "lab [lab-id] ou -f [arquivo]",
	Short: "Cria um novo laboratório no Girus",
	Long:  "Adiciona um novo laboratório ao Girus a partir de um arquivo de manifesto ConfigMap, ou cria um ambiente de laboratório a partir de um ID de template existente.\nOs templates de laboratório são armazenados no diretório /labs na raiz do projeto.",
	Run: func(cmd *cobra.Command, args []string) {
		// Criar formatadores de cores
		red := color.New(color.FgRed).SprintFunc()

		// Verificar qual modo estamos
		if labFile != "" {
			// Modo de adicionar template a partir de arquivo
			lab.AddLabFromFile(labFile, verboseMode)
		} else if len(args) > 0 {
			// Modo de adicionar template a partir do repositório remoto
			labID := args[0]
			createLabFromRepo(labID, repoIndexURL, verboseMode)
		} else {
			fmt.Fprintf(os.Stderr, "%s %s\n", red("ERRO:"), common.T("Você deve especificar um ID de laboratório ou um arquivo com a flag -f", "Debe especificar un ID de laboratorio o un archivo con la opción -f"))
			fmt.Println(common.T("\nExemplos:", "\nEjemplos:"))
			fmt.Println(common.T("  girus create lab linux-monitoramento-sistema  # Instala um laboratório do repositório remoto", "  girus create lab linux-monitoramento-sistema  # Instala un laboratorio del repositorio remoto"))
			fmt.Println(common.T("  girus create lab -f meulaboratorio.yaml       # Adiciona um novo template a partir do arquivo", "  girus create lab -f mi-lab.yaml             # Añade una nueva plantilla desde el archivo"))
			os.Exit(1)
		}
	},
}

// createLabFromRepo baixa e aplica um laboratório do repositório remoto pelo ID
func createLabFromRepo(labID string, indexURL string, verboseMode bool) {
	// Criar formatadores de cores
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf(common.T("%s Buscando laboratório '%s'...\n", "%s Buscando laboratorio '%s'...\n"), cyan("INFO:"), magenta(labID))

	// Buscar o laboratório no index.yaml
	labInfo, err := repo.FindLabByID(labID, indexURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", red("ERRO:"), err)
		fmt.Println(common.T("\nPara ver os laboratórios disponíveis, use:", "\nPara ver los laboratorios disponibles, use:"))
		fmt.Println("  girus list repo-labs")
		os.Exit(1)
	}

	fmt.Printf(common.T("%s Baixando o template de '%s'...\n", "%s Descargando la plantilla de '%s'...\n"), cyan("INFO:"), magenta(labInfo.Title))

	// Fazer o download do lab.yaml
	tempFile, err := repo.DownloadLabYAML(labInfo.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", red("ERRO:"), err)
		os.Exit(1)
	}
	defer os.Remove(tempFile) // Garantir que o arquivo temporário seja removido ao final

	// Aplicar o laboratório
	fmt.Println(headerColor(common.T("Aplicando laboratório no cluster GIRUS...", "Aplicando laboratorio en el cluster GIRUS...")))
	lab.AddLabFromFile(tempFile, verboseMode)
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
	createLabCmd.Flags().StringVarP(&repoIndexURL, "url", "u", "", "URL do arquivo index.yaml (opcional)")

	// definir o nome do cluster como "girus" sempre
	clusterName = "girus"
}
