package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var (
	// Criar formatadores de cores
	red         = color.New(color.FgRed).SprintFunc()
	cyan        = color.New(color.FgCyan).SprintFunc()
	green       = color.New(color.FgGreen).SprintFunc()
	yellow      = color.New(color.FgYellow).SprintFunc()
	magenta     = color.New(color.FgMagenta).SprintFunc()
	headerColor = color.New(color.FgCyan, color.Bold).SprintFunc()
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Parar o ambiente do GIRUS",
	Long:  "Parar o ambiente do GIRUS CLI, removendo todos os recursos criados pelo GIRUS CLI.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s Você está prestes a parar o %s e o %s no cluster %s.\n",
			yellow("AVISO:"), magenta("frontend"), magenta("backend"), magenta(clusterName))
		fmt.Print("Deseja continuar? [s/N]: ")

		reader := bufio.NewReader(os.Stdin)
		confirmStr, _ := reader.ReadString('\n')
		confirm := strings.TrimSpace(strings.ToLower(confirmStr))

		if confirm != "s" && confirm != "sim" && confirm != "y" && confirm != "yes" {
			fmt.Println("Operação cancelada pelo usuário.")
			return
		}
		// Define os nomes dos deployments
		frontendDeploymentName := "girus-frontend"
		backendDeploymentName := "girus-backend"
		// Criando um client para interagir com o cluster do Kubernetes
		client, err := k8s.NewKubernetesClient()
		if err != nil {
			fmt.Printf("Erro ao criar cliente Kubernetes: %v\n", err)
			return
		}

		ctx := context.Background()
		// Pega todos os pods do namespace do girus
		pods, err := client.ListRunningPods(ctx, "girus")
		if err != nil {
			fmt.Printf("Erro ao tentar pegar a lista de pods: %v\n", err)
			return
		}

		// Pega o nome dos pods do frontend e do backend
		var frontendPod string
		var backendPod string
		for _, pod := range pods {
			if strings.Contains(pod, "frontend") {
				frontendPod = pod
			} else if strings.Contains(pod, "backend") {
				backendPod = pod
			}
		}

		// Verifica se o backend está em execução, se sim, parar o deploy e remover o serviço
		if isRunning, _ := client.IsPodRunning(ctx, "girus", backendPod); isRunning {
			err := stopDeployment(client, ctx, backendDeploymentName)
			if err != nil {
				fmt.Printf("falha ao tentar parar o deploy do backend do GIRUS: %v\n", err)
				return
			}
			fmt.Println("✅ Backend parado com sucesso.")

		} else {
			fmt.Println("⚠️ O backend não está em execução.")
		}

		// Verifica se o frontend está em execução, se sim, parar o deploy e remover o serviço
		if isRunning, _ := client.IsPodRunning(ctx, "girus", frontendPod); isRunning {
			err := stopDeployment(client, ctx, frontendDeploymentName)
			if err != nil {
				fmt.Printf("falha ao tentar parar o deploy do frontend do GIRUS: %v\n", err)
				return
			}
			fmt.Println("✅ Frontend parado com sucesso.")
		} else {
			fmt.Println("⚠️ O frontend não está em execução..")
		}
	},
}

func stopDeployment(client *k8s.KubernetesClient, ctx context.Context, deploymentName string) error {
	err := client.ScaleDeploy(ctx, "girus", deploymentName, 0)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Erro ao tentar parar o pod: %v\n", err)
		if err != nil {
			return err
		}
		fmt.Printf("%s Você quer forçar a parada do deployment %s?\n",
			yellow("AVISO:"), magenta(deploymentName))
		fmt.Print("Deseja continuar? [s/N]: ")

		reader := bufio.NewReader(os.Stdin)
		confirmStr, _ := reader.ReadString('\n')
		confirm := strings.TrimSpace(strings.ToLower(confirmStr))

		if confirm != "s" && confirm != "sim" && confirm != "y" && confirm != "yes" {
			fmt.Println("Operação cancelada pelo usuário.")
			return err
		}
		return err
	}

	return nil
}
