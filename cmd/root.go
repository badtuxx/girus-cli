package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "girus",
	Short: "GIRUS - Plataforma de Laboratórios Interativos",
	Long: `GIRUS é uma plataforma open-source de laboratórios interativos que permite a criação,
gerenciamento e execução de ambientes de aprendizado prático para tecnologias como Linux,
Docker, Kubernetes, Terraform e outras ferramentas essenciais para profissionais de DevOps,
SRE, Dev e Platform Engineering.`,
}

// Execute executa o comando raiz
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Criar formatadores de cores
	magenta := color.New(color.FgMagenta).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()

	// Customizar templates de help
	cobra.AddTemplateFunc("magenta", func(s string) string {
		return magenta(s)
	})

	cobra.AddTemplateFunc("cyan", func(s string) string {
		return cyan(s)
	})

	cobra.AddTemplateFunc("header", func(s string) string {
		return headerColor(s)
	})

	// Template personalizado para o help principal
	rootCmd.SetUsageTemplate(`{{header "GIRUS - Plataforma de Laboratórios Interativos"}}

{{.Long}}

{{header "Usage:"}}
  {{magenta .Use}}

{{header "Available Commands:"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{magenta .Name | printf "%-12s"}} {{.Short}}{{end}}{{end}}

{{header "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{magenta "girus [command] --help"}}" for more information about a command.
`)

	// Template personalizado para o help de comandos
	rootCmd.SetHelpTemplate(`{{header .Name}} - {{.Short}}

{{.Long}}

{{header "Usage:"}}
  {{magenta .UseLine}}

{{if .HasAvailableSubCommands}}{{header "Available Commands:"}}{{range .Commands}}{{if .IsAvailableCommand}}
  {{magenta .Name | printf "%-12s"}} {{.Short}}{{end}}{{end}}
{{end}}

{{if .HasAvailableLocalFlags}}{{header "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasAvailableInheritedFlags}}{{header "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`)

	// Adiciona os comandos
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(labCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(startCmd)

	// Não adicionar updateCmd aqui, pois já é adicionado no update.go

	// Configura flags globais
	rootCmd.PersistentFlags().StringP("config", "c", "", "arquivo de configuração (padrão: $HOME/.girus/config.yaml)")
}
