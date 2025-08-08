package cmd

import (
	"fmt"

	"github.com/badtuxx/girus-cli/internal/common"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: common.T("Exibe a versão do Girus CLI", "Muestra la versión del Girus CLI"),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(common.GetVersion())
	},
}
