package cmd

import (
	"context"
	"fmt"

	"github.com/badtuxx/girus-cli/internal/common"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: common.T("Atualiza o progresso dos laboratórios instalados", "Actualiza el progresso de los laboratorios instalados"),
	Long: common.T(`Verifica e atualiza o progresso dos laboratórios instalados no arquivo progresso.yaml no diretório do cluster ("~/.girus/progresso.yaml"). Esse commando irá fazer um sync entre o ConfigMap do cluster chamado progresso.yaml, e o arquivo local progresso.yaml.`,
		`Verifica y actualiza el progresso de los laboratorios instalados en el archivo progresso.yaml en el directorio del cluster ("~/.girus/progresso.yaml"). Este commando irá hacer un sync entre el ConfigMap del cluster llamado progresso.yaml, y el archivo local progresso.yaml.`),
	Run: func(cmd *cobra.Command, args []string) {
		// Criando o cliente Kubernetes para conectar à API do cluster
		client, err := k8s.NewKubernetesClient()
		if err != nil {
			fmt.Printf("erro ao criar o cliente Kubernetes: %s\n", err)
			return
		}

		// Sincronizando a lista de laboratorios do ConfigMap com o arquivo progresso.yaml
		laboratorios, _ := client.GetAllLabs(context.Background())
		progress := common.Progress{}
		err = progress.SaveProgressToFile()
		if err != nil {
			fmt.Printf("erro ao salvar o progresso no arquivo yaml progress.yaml: %s\n", err)
			return
		}

		// Pega a lista de laboratórios instalados do ConfigMap
		fmt.Printf("%s\n", headerColor("Progresso dos Laboratórios:"))
		// Itera sobre a lista e printa no terminal o progresso de cada laboratório
		for _, lab := range laboratorios {
			fmt.Printf("%s: ", lab.Name)
			if lab.Status == "in-progress" {
				fmt.Printf("⏳ %s\n", magenta(lab.Status))
			} else {
				fmt.Printf("✅ %s\n", green(lab.Status))
			}
		}
	},
}
