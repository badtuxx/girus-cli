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
		// Verificar se o Docker está instalado e funcionando
		fmt.Println("🔄 Verificando pré-requisitos...")
		dockerCmd := exec.Command("docker", "--version")
		if err := dockerCmd.Run(); err != nil {
			fmt.Println("❌ Docker não encontrado ou não está em execução")
			fmt.Println("\nO Docker é necessário para criar um cluster Kind. Instruções de instalação:")

			// Detectar o sistema operacional para instruções específicas
			if runtime.GOOS == "darwin" {
				// macOS
				fmt.Println("\n📦 Para macOS, recomendamos usar Colima (alternativa leve ao Docker Desktop):")
				fmt.Println("1. Instale o Homebrew caso não tenha:")
				fmt.Println("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
				fmt.Println("2. Instale o Colima e o Docker CLI:")
				fmt.Println("   brew install colima docker")
				fmt.Println("3. Inicie o Colima:")
				fmt.Println("   colima start")
				fmt.Println("\nAlternativamente, você pode instalar o Docker Desktop para macOS de:")
				fmt.Println("https://www.docker.com/products/docker-desktop")
			} else if runtime.GOOS == "linux" {
				// Linux
				fmt.Println("\n📦 Para Linux, use o script de instalação oficial:")
				fmt.Println("   curl -fsSL https://get.docker.com | bash")
				fmt.Println("\nApós a instalação, adicione seu usuário ao grupo docker para evitar usar sudo:")
				fmt.Println("   sudo usermod -aG docker $USER")
				fmt.Println("   newgrp docker")
				fmt.Println("\nE inicie o serviço:")
				fmt.Println("   sudo systemctl enable docker")
				fmt.Println("   sudo systemctl start docker")
			} else {
				// Windows ou outros sistemas
				fmt.Println("\n📦 Visite https://www.docker.com/products/docker-desktop para instruções de instalação para seu sistema operacional")
			}

			fmt.Println("\nApós instalar o Docker, execute novamente este comando.")
			os.Exit(1)
		}

		// Verificar se o serviço Docker está rodando
		dockerInfoCmd := exec.Command("docker", "info")
		if err := dockerInfoCmd.Run(); err != nil {
			fmt.Println("❌ O serviço Docker não está em execução")

			if runtime.GOOS == "darwin" {
				fmt.Println("\nPara macOS com Colima:")
				fmt.Println("   colima start")
				fmt.Println("\nPara Docker Desktop:")
				fmt.Println("   Inicie o aplicativo Docker Desktop")
			} else if runtime.GOOS == "linux" {
				fmt.Println("\nInicie o serviço Docker:")
				fmt.Println("   sudo systemctl start docker")
			} else {
				fmt.Println("\nInicie o Docker Desktop ou o serviço Docker apropriado para seu sistema.")
			}

			fmt.Println("\nApós iniciar o Docker, execute novamente este comando.")
			os.Exit(1)
		}

		fmt.Println("✅ Docker detectado e funcionando")

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
				fmt.Println("   • Docker não está em execução")
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
					fmt.Println("   Erro: Permissão negada. Verifique as permissões do Docker.")
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

			// Criar um arquivo temporário para o template do laboratório Linux
			labTempFile, err := os.CreateTemp("", "basic-linux-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template Linux: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
				return
			}
			defer os.Remove(labTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			basicLinuxTemplate, err := templates.GetManifest("linux.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}

			// Escrever o conteúdo do template Linux no arquivo temporário
			if _, err := labTempFile.WriteString(string(basicLinuxTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template Linux no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
				return
			}
			labTempFile.Close()

			// Criar um arquivo temporário para o template do laboratório Kubernetes
			k8sTempFile, err := os.CreateTemp("", "kubernetes-basics-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template Kubernetes: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Kubernetes.")
				return
			}
			defer os.Remove(k8sTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			basicKubernetesTemplate, err := templates.GetManifest("kubernetes.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}

			// Escrever o conteúdo do template Kubernetes no arquivo temporário
			if _, err := k8sTempFile.WriteString(string(basicKubernetesTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template Kubernetes no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Kubernetes.")
				return
			}
			k8sTempFile.Close()

			// Criar um arquivo temporário para o template do laboratório Docker
			dockerTempFile, err := os.CreateTemp("", "docker-basics-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template Docker: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Docker.")
				return
			}
			defer os.Remove(dockerTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			basicDockerTemplate, err := templates.GetManifest("docker.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}
			// Escrever o conteúdo do template Docker no arquivo temporário
			if _, err := dockerTempFile.WriteString(string(basicDockerTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template Docker no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Docker.")
				return
			}
			dockerTempFile.Close()

			// Criar um arquivo temporário para o template de Administração de Usuários Linux
			linuxUsersTempFile, err := os.CreateTemp("", "linux-users-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Usuários Linux: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Usuários Linux.")
				return
			}
			defer os.Remove(linuxUsersTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			linuxUsersTemplate, err := templates.GetManifest("linux-users.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}
			// Escrever o conteúdo do template de Usuários Linux no arquivo temporário
			if _, err := linuxUsersTempFile.WriteString(string(linuxUsersTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Usuários Linux no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Usuários Linux.")
				return
			}
			linuxUsersTempFile.Close()

			// Criar um arquivo temporário para o template de Permissões de Arquivos Linux
			linuxPermsTempFile, err := os.CreateTemp("", "linux-perms-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Permissões Linux: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Permissões Linux.")
				return
			}
			defer os.Remove(linuxPermsTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			linuxPermsTemplate, err := templates.GetManifest("linux-permissions.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}
			// Escrever o conteúdo do template de Permissões Linux no arquivo temporário
			if _, err := linuxPermsTempFile.WriteString(string(linuxPermsTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Permissões Linux no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Permissões Linux.")
				return
			}
			linuxPermsTempFile.Close()

			// Criar um arquivo temporário para o template de Gerenciamento de Containers Docker
			dockerContainersTempFile, err := os.CreateTemp("", "docker-containers-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Containers Docker: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Containers Docker.")
				return
			}
			defer os.Remove(dockerContainersTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			dockerContainersTemplate, err := templates.GetManifest("containers.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}

			// Escrever o conteúdo do template de Containers Docker no arquivo temporário
			if _, err := dockerContainersTempFile.WriteString(string(dockerContainersTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Containers Docker no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Containers Docker.")
				return
			}
			dockerContainersTempFile.Close()

			// Criar um arquivo temporário para o template de Deployment Kubernetes
			k8sDeploymentTempFile, err := os.CreateTemp("", "k8s-deployment-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Deployment Kubernetes: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Deployment Kubernetes.")
				return
			}
			defer os.Remove(k8sDeploymentTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			k8sDeploymentTemplate, err := templates.GetManifest("deployment.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao carregar o template: %v\n", err)
				return
			}

			// Escrever o conteúdo do template de Deployment Kubernetes no arquivo temporário
			if _, err := k8sDeploymentTempFile.WriteString(string(k8sDeploymentTemplate)); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Deployment Kubernetes no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Deployment Kubernetes.")
				return
			}
			k8sDeploymentTempFile.Close()

			// Aplicar o template de laboratório Linux
			if verboseMode {
				// Executar normalmente mostrando o output
				fmt.Println("   Aplicando template de laboratório Linux...")
				applyLabCmd := exec.Command("kubectl", "apply", "-f", labTempFile.Name())
				applyLabCmd.Stdout = os.Stdout
				applyLabCmd.Stderr = os.Stderr

				if err := applyLabCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Linux: %v\n", err)
					fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Linux.")
				} else {
					fmt.Println("   ✅ Template de laboratório Linux Básico aplicado com sucesso!")
				}

				// Aplicar o template de laboratório Kubernetes
				fmt.Println("   Aplicando template de laboratório Kubernetes...")
				applyK8sCmd := exec.Command("kubectl", "apply", "-f", k8sTempFile.Name())
				applyK8sCmd.Stdout = os.Stdout
				applyK8sCmd.Stderr = os.Stderr

				if err := applyK8sCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Kubernetes: %v\n", err)
					fmt.Println("   A infraestrutura básica e o template Linux foram aplicados, mas sem o template de laboratório Kubernetes.")
				} else {
					fmt.Println("   ✅ Template de laboratório Fundamentos de Kubernetes aplicado com sucesso!")
				}

				// Aplicar o template de laboratório Docker
				fmt.Println("   Aplicando template de laboratório Docker...")
				applyDockerCmd := exec.Command("kubectl", "apply", "-f", dockerTempFile.Name())
				applyDockerCmd.Stdout = os.Stdout
				applyDockerCmd.Stderr = os.Stderr

				if err := applyDockerCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Docker: %v\n", err)
					fmt.Println("   A infraestrutura básica e os outros templates foram aplicados, mas sem o template de laboratório Docker.")
				} else {
					fmt.Println("   ✅ Template de laboratório Fundamentos de Docker aplicado com sucesso!")
				}

				// Aplicar o template de Usuários Linux
				fmt.Println("   Aplicando template de Administração de Usuários Linux...")
				applyLinuxUsersCmd := exec.Command("kubectl", "apply", "-f", linuxUsersTempFile.Name())
				applyLinuxUsersCmd.Stdout = os.Stdout
				applyLinuxUsersCmd.Stderr = os.Stderr

				if err := applyLinuxUsersCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Usuários Linux: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Administração de Usuários Linux aplicado com sucesso!")
				}

				// Aplicar o template de Permissões Linux
				fmt.Println("   Aplicando template de Permissões de Arquivos Linux...")
				applyLinuxPermsCmd := exec.Command("kubectl", "apply", "-f", linuxPermsTempFile.Name())
				applyLinuxPermsCmd.Stdout = os.Stdout
				applyLinuxPermsCmd.Stderr = os.Stderr

				if err := applyLinuxPermsCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Permissões Linux: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Permissões de Arquivos Linux aplicado com sucesso!")
				}

				// Aplicar o template de Containers Docker
				fmt.Println("   Aplicando template de Gerenciamento de Containers Docker...")
				applyDockerContainersCmd := exec.Command("kubectl", "apply", "-f", dockerContainersTempFile.Name())
				applyDockerContainersCmd.Stdout = os.Stdout
				applyDockerContainersCmd.Stderr = os.Stderr

				if err := applyDockerContainersCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Containers Docker: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Gerenciamento de Containers Docker aplicado com sucesso!")
				}

				// Aplicar o template de Deployment Kubernetes
				fmt.Println("   Aplicando template de Deployment Nginx Kubernetes...")
				applyK8sDeploymentCmd := exec.Command("kubectl", "apply", "-f", k8sDeploymentTempFile.Name())
				applyK8sDeploymentCmd.Stdout = os.Stdout
				applyK8sDeploymentCmd.Stderr = os.Stderr

				if err := applyK8sDeploymentCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Deployment Kubernetes: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Deployment Nginx Kubernetes aplicado com sucesso!")
				}
			} else {
				// Usar barra de progresso para os templates
				bar := progressbar.NewOptions(100,
					progressbar.OptionSetDescription("Aplicando templates de laboratório..."),
					progressbar.OptionSetWidth(80),
					progressbar.OptionShowBytes(false),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
				)

				// Executar comando para aplicar o template Linux
				applyLabCmd := exec.Command("kubectl", "apply", "-f", labTempFile.Name())
				var stderrLinux bytes.Buffer
				applyLabCmd.Stderr = &stderrLinux

				// Iniciar o comando
				err := applyLabCmd.Start()
				if err != nil {
					bar.Finish()
					fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar aplicação do template Linux: %v\n", err)
					fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
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
								time.Sleep(50 * time.Millisecond)
							}
						}
					}()

					// Aguardar o final do comando
					err = applyLabCmd.Wait()
					close(done)

					linuxSuccess := err == nil

					// Aplicar o template de Kubernetes
					applyK8sCmd := exec.Command("kubectl", "apply", "-f", k8sTempFile.Name())
					var stderrK8s bytes.Buffer
					applyK8sCmd.Stderr = &stderrK8s

					err = applyK8sCmd.Run()
					k8sSuccess := err == nil

					// Aplicar o template de Docker
					applyDockerCmd := exec.Command("kubectl", "apply", "-f", dockerTempFile.Name())
					var stderrDocker bytes.Buffer
					applyDockerCmd.Stderr = &stderrDocker

					err = applyDockerCmd.Run()
					dockerSuccess := err == nil

					// Aplicar os novos templates
					applyLinuxUsersCmd := exec.Command("kubectl", "apply", "-f", linuxUsersTempFile.Name())
					var stderrLinuxUsers bytes.Buffer
					applyLinuxUsersCmd.Stderr = &stderrLinuxUsers

					err = applyLinuxUsersCmd.Run()
					linuxUsersSuccess := err == nil

					applyLinuxPermsCmd := exec.Command("kubectl", "apply", "-f", linuxPermsTempFile.Name())
					var stderrLinuxPerms bytes.Buffer
					applyLinuxPermsCmd.Stderr = &stderrLinuxPerms

					err = applyLinuxPermsCmd.Run()
					linuxPermsSuccess := err == nil

					applyDockerContainersCmd := exec.Command("kubectl", "apply", "-f", dockerContainersTempFile.Name())
					var stderrDockerContainers bytes.Buffer
					applyDockerContainersCmd.Stderr = &stderrDockerContainers

					err = applyDockerContainersCmd.Run()
					dockerContainersSuccess := err == nil

					applyK8sDeploymentCmd := exec.Command("kubectl", "apply", "-f", k8sDeploymentTempFile.Name())
					var stderrK8sDeployment bytes.Buffer
					applyK8sDeploymentCmd.Stderr = &stderrK8sDeployment

					err = applyK8sDeploymentCmd.Run()
					k8sDeploymentSuccess := err == nil

					bar.Finish()

					if !linuxSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Linux: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrLinux.String())
						fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Linux.")
					}

					if !k8sSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Kubernetes: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrK8s.String())
						fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Kubernetes.")
					}

					if !dockerSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Docker: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrDocker.String())
						fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Docker.")
					}

					if !linuxUsersSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Usuários Linux: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrLinuxUsers.String())
					}

					if !linuxPermsSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Permissões Linux: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrLinuxPerms.String())
					}

					if !dockerContainersSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Containers Docker: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrDockerContainers.String())
					}

					if !k8sDeploymentSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Deployment Kubernetes: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrK8sDeployment.String())
					}

					if linuxSuccess && k8sSuccess && dockerSuccess &&
						linuxUsersSuccess && linuxPermsSuccess &&
						dockerContainersSuccess && k8sDeploymentSuccess {
						fmt.Println("✅ Todos os templates de laboratório aplicados com sucesso!")

						// Verificação de diagnóstico para confirmar que os templates estão visíveis
						fmt.Println("\n🔍 Verificando templates de laboratório instalados:")
						listLabsCmd := exec.Command("kubectl", "get", "configmap", "-n", "girus", "-l", "app=girus-lab-template", "-o", "custom-columns=NAME:.metadata.name")

						// Capturar output para apresentá-lo de forma mais organizada
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

					} else if linuxSuccess {
						fmt.Println("✅ Template de laboratório Linux aplicado com sucesso!")
					} else if k8sSuccess {
						fmt.Println("✅ Template de laboratório Kubernetes aplicado com sucesso!")
					} else if dockerSuccess {
						fmt.Println("✅ Template de laboratório Docker aplicado com sucesso!")
					}
				}
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

	// Flags para createLabCmd
	createLabCmd.Flags().StringVarP(&labFile, "file", "f", "", "Arquivo de manifesto do laboratório (ConfigMap)")
	createLabCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Modo detalhado com output completo em vez da barra de progresso")

	// definir o nome do cluster como "girus" sempre
	clusterName = "girus"
}
