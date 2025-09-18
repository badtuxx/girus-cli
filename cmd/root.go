package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/badtuxx/girus-cli/internal/common"
)

var rootCmd = &cobra.Command{
	Use:   "girus",
	Short: common.T("GIRUS - Plataforma de Laboratórios Interativos", "GIRUS - Plataforma de Laboratorios Interactivos"),
	Long: common.T(`GIRUS é uma plataforma open-source de laboratórios interativos que permite a criação,
gerenciamento e execução de ambientes de aprendizado prático para tecnologias como Linux,
Docker, Kubernetes, Terraform e outras ferramentas essenciais para profissionais de DevOps,
SRE, Dev e Platform Engineering.`,
		`GIRUS es una plataforma de código abierto para laboratorios interactivos que permite crear,
gestionar y ejecutar entornos de aprendizaje práctico para tecnologías como Linux,
Docker, Kubernetes, Terraform y otras herramientas esenciales para profesionales de DevOps,
SRE, Dev y Platform Engineering.`),
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
	rootCmd.SetUsageTemplate(fmt.Sprintf(`{{header "%s"}}

{{.Long}}

{{header "%s"}}
  {{magenta .Use}}

{{header "%s"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{magenta .Name | printf "%%-12s"}} {{.Short}}{{end}}{{end}}

{{header "%s"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

%s`,
		common.T("GIRUS - Plataforma de Laboratórios Interativos", "GIRUS - Plataforma de Laboratorios Interactivos"),
		common.T("Usage:", "Uso:"),
		common.T("Available Commands:", "Comandos Disponibles:"),
		common.T("Flags:", "Flags:"),
		common.T("Use \"girus [command] --help\" for more information about a command.", "Use \"girus [command] --help\" para obtener más información sobre un comando.")))

	// Template personalizado para o help de comandos
	rootCmd.SetHelpTemplate(fmt.Sprintf(`{{header .Name}} - {{.Short}}

{{.Long}}

{{header "%s"}}
  {{magenta .UseLine}}

{{if .HasAvailableSubCommands}}{{header "%s"}}{{range .Commands}}{{if .IsAvailableCommand}}
  {{magenta .Name | printf "%%-12s"}} {{.Short}}{{end}}{{end}}
{{end}}

{{if .HasAvailableLocalFlags}}{{header "%s"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasAvailableInheritedFlags}}{{header "%s"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`,
		common.T("Usage:", "Uso:"),
		common.T("Available Commands:", "Comandos Disponibles:"),
		common.T("Flags:", "Flags:"),
		common.T("Global Flags:", "Flags Globales:")))

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
	rootCmd.AddCommand(progressCmd)

	// Não adicionar updateCmd aqui, pois já é adicionado no update.go

	// Configura flags globais
	rootCmd.PersistentFlags().StringP("config", "c", "", common.T("arquivo de configuração (padrão: $HOME/.girus/config.yaml)", "archivo de configuración (predeterminado: $HOME/.girus/config.yaml)"))
}
