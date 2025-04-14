package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/badtuxx/girus-cli/internal/config"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var forceDelete bool
var verboseDelete bool
var clearExternalLabs bool

var deleteCmd = &cobra.Command{
	Use:   "delete [subcommand]",
	Short: "Comandos para excluir recursos",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var deleteClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Exclui o cluster Girus",
	Long:  "Exclui o cluster Girus do sistema, incluindo todos os recursos do Girus.",
	Run: func(cmd *cobra.Command, args []string) {
		clusterName := "girus"

		// Verificar se o cluster existe
		checkCmd := exec.Command("kind", "get", "clusters")
		output, err := checkCmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao obter lista de clusters: %v\n", err)
			os.Exit(1)
		}

		clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
		clusterExists := false
		for _, cluster := range clusters {
			if cluster == clusterName {
				clusterExists = true
				break
			}
		}

		if !clusterExists {
			fmt.Fprintf(os.Stderr, "Erro: cluster 'girus' não encontrado\n")
			os.Exit(1)
		}

		// Confirmar a exclusão se -f/--force não estiver definido
		if !forceDelete {
			fmt.Printf("Você está prestes a excluir o cluster Girus. Esta ação é irreversível.\n")
			fmt.Print("Deseja continuar? (s/N): ")

			reader := bufio.NewReader(os.Stdin)
			confirmStr, _ := reader.ReadString('\n')
			confirm := strings.TrimSpace(strings.ToLower(confirmStr))

			if confirm != "s" && confirm != "sim" && confirm != "y" && confirm != "yes" {
				fmt.Println("Operação cancelada pelo usuário.")
				return
			}
		}

		fmt.Println("Excluindo o cluster Girus...")

		if verboseDelete {
			// Excluir o cluster mostrando o output normal
			deleteCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
			deleteCmd.Stdout = os.Stdout
			deleteCmd.Stderr = os.Stderr

			if err := deleteCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao excluir o cluster Girus: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Usando barra de progresso (padrão)
			bar := progressbar.NewOptions(100,
				progressbar.OptionSetDescription("Excluindo cluster..."),
				progressbar.OptionSetWidth(50),
				progressbar.OptionShowBytes(false),
				progressbar.OptionSetPredictTime(false),
				progressbar.OptionThrottle(65*time.Millisecond),
				progressbar.OptionShowCount(),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionFullWidth(),
			)

			// Executar comando sem mostrar saída
			deleteCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
			var stderr bytes.Buffer
			deleteCmd.Stderr = &stderr

			// Iniciar o comando
			err := deleteCmd.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao iniciar o comando: %v\n", err)
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
						time.Sleep(150 * time.Millisecond)
					}
				}
			}()

			// Aguardar o final do comando
			err = deleteCmd.Wait()
			close(done)
			bar.Finish()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao excluir o cluster Girus: %v\n%s\n", err, stderr.String())
				os.Exit(1)
			}
		}

		fmt.Println("Cluster Girus excluído com sucesso!")

		// Verificar se existem repositórios de labs externos configurados
		repos, err := config.GetExternalRepositories()
		if err == nil && len(repos) > 0 {
			shouldClear := clearExternalLabs

			// Se a flag não foi especificada, perguntar ao usuário
			if !shouldClear && !forceDelete {
				fmt.Printf("\nForam encontrados %d repositórios de labs externos configurados.\n", len(repos))
				fmt.Print("Deseja remover as configurações de labs externos também? (s/N): ")

				reader := bufio.NewReader(os.Stdin)
				confirmStr, _ := reader.ReadString('\n')
				confirm := strings.TrimSpace(strings.ToLower(confirmStr))

				shouldClear = (confirm == "s" || confirm == "sim" || confirm == "y" || confirm == "yes")
			}

			if shouldClear {
				// Limpar a configuração (remover todos os repositórios)
				configObj := &config.Config{
					ExternalRepositories: []config.ExternalLabRepository{},
				}
				if err := config.SaveConfig(configObj); err != nil {
					fmt.Printf("\n⚠️  Aviso: não foi possível limpar a configuração de labs externos: %v\n", err)
				} else {
					fmt.Println("\n✅ Configurações de labs externos removidas com sucesso!")
				}
			} else {
				fmt.Printf("\nℹ️  As configurações de %d repositórios de labs externos foram mantidas.\n", len(repos))
				fmt.Println("   Para removê-los posteriormente, use o comando 'girus delete cluster --clear-external-labs'")
				fmt.Println("   Você também pode editar ou remover manualmente o arquivo ~/.girus/config.yaml")
			}
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteClusterCmd)

	// Flag para forçar a exclusão sem confirmação
	deleteClusterCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Força a exclusão sem confirmação")

	// Flag para modo detalhado com output completo
	deleteClusterCmd.Flags().BoolVarP(&verboseDelete, "verbose", "v", false, "Modo detalhado com output completo em vez da barra de progresso")

	// Flag para limpar configurações de labs externos
	deleteClusterCmd.Flags().BoolVarP(&clearExternalLabs, "clear-external-labs", "", false, "Remove também as configurações de labs externos")
}
