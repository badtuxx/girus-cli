package cmd

import (
	"context"
	"fmt"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/spf13/cobra"
	"strings"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Inicia o ambiente do GIRUS",
	Long:  "Inicia o ambiente do GIRUS CLI, reiniciando o deployment do backend e do frontend.",
	Run: func(cmd *cobra.Command, args []string) {
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

		pods, err := client.ListRunningPods(ctx, "girus")
		if err != nil {
			fmt.Printf("Erro ao tentar pegar a lista de pods em execução no namespace do GIRUS: %v\n", err)
			fmt.Println("Leia o erro, se você não conseguir resolvê-lo, recrie o cluster.")
			return
		}
		// Pega todos os pods do namespace do girus
		var frontendPod string
		var backendPod string
		for _, pod := range pods {
			if strings.Contains(pod, "frontend") {
				frontendPod = pod
			} else if strings.Contains(pod, "backend") {
				backendPod = pod
			}
		}
		// Checa se o frontend já está executando antes de tentar iniciar o deployment
		isFrontendRunning, err := client.IsPodRunning(ctx, "girus", frontendPod)
		if isFrontendRunning {
			fmt.Println("O pod de frontend já está em execução.")
			fmt.Println("Tente abrir o browser e navegar até http://localhost:8000.")
		}
		if err != nil {
			fmt.Println("Nenhum pod do frontend encontrado no namespace do GIRUS...")
		}

		// Checa se o backend já está executando antes de tentar iniciar o deployment
		isBackendRunning, err := client.IsPodRunning(ctx, "girus", backendPod)
		if isBackendRunning {
			fmt.Println("O pod de backend já está em execução.")
			fmt.Println("Tente abrir o browser e navegar até http://localhost:8000.")
			fmt.Printf("%s Cancelando.\n", yellow("AVISO"))
			return
		}
		if err != nil {
			fmt.Println("Nenhum pod do backend encontrado no namespace do GIRUS...")
		}
		err = startDeployment(client, ctx, backendDeploymentName)
		if err != nil {
			fmt.Printf("Erro ao tentar iniciar o backend: %v\n", err)
			return
		}
		err = startDeployment(client, ctx, frontendDeploymentName)
		if err != nil {
			fmt.Printf("Erro ao tentar iniciar o frontend: %v\n", err)
			return
		}
	},
}

func startDeployment(client *k8s.KubernetesClient, ctx context.Context, deploymentName string) error {
	err := client.ScaleDeploy(ctx, "girus", deploymentName, 1)
	if err != nil {
		fmt.Printf("Erro ao tentar iniciar o deploy %s: %v\n", magenta(deploymentName), yellow(err))
		fmt.Println("Leia o erro, se você não conseguir resolvê-lo, recrie o cluster.")
		return err
	}

	return nil
}
