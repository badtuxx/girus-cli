package helpers

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// portInUse verifica se uma porta está em uso
func PortInUse(port int) bool {
	checkCmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	return checkCmd.Run() == nil
}

// openBrowser abre o navegador com a URL especificada
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("não foi possível abrir o navegador (sistema operacional não suportado)")
	}

	return cmd.Start()
}

// checkPortForwardNeeded verifica se o port-forward para o backend precisa ser reconfigurado
func CheckPortForwardNeeded() bool {
	backendNeeded := false
	frontendNeeded := false

	// Verificar se a porta 8080 (backend) está em uso
	backendPortCmd := exec.Command("lsof", "-i", ":8080")
	if backendPortCmd.Run() != nil {
		// Porta 8080 não está em uso, precisamos de port-forward
		backendNeeded = true
	} else {
		// Porta está em uso, mas precisamos verificar se é o kubectl port-forward e se está funcional
		// Verificar se o processo é kubectl port-forward
		backendProcessCmd := exec.Command("sh", "-c", "ps -eo pid,cmd | grep 'kubectl port-forward' | grep '8080' | grep -v grep")
		if backendProcessCmd.Run() != nil {
			// Não encontrou processo de port-forward ativo ou válido
			backendNeeded = true
		} else {
			// Verificar se a conexão com o backend está funcionando
			backendHealthCmd := exec.Command("curl", "-s", "--head", "--max-time", "2", "http://localhost:8080/api/v1/health")
			backendNeeded = backendHealthCmd.Run() != nil // Retorna true (precisa de port-forward) se o comando falhar
		}
	}

	// Verificar se a porta 8000 (frontend) está em uso
	frontendPortCmd := exec.Command("lsof", "-i", ":8000")
	if frontendPortCmd.Run() != nil {
		// Porta 8000 não está em uso, precisamos de port-forward
		frontendNeeded = true
	} else {
		// Porta está em uso, mas precisamos verificar se é o kubectl port-forward e se está funcional
		// Verificar se o processo é kubectl port-forward
		frontendProcessCmd := exec.Command("sh", "-c", "ps -eo pid,cmd | grep 'kubectl port-forward' | grep '8000' | grep -v grep")
		if frontendProcessCmd.Run() != nil {
			// Não encontrou processo de port-forward ativo ou válido
			frontendNeeded = true
		} else {
			// Verificar se a conexão com o frontend está funcionando
			frontendCheckCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8000")
			var out bytes.Buffer
			frontendCheckCmd.Stdout = &out
			if frontendCheckCmd.Run() != nil {
				frontendNeeded = true
			} else {
				statusCode := strings.TrimSpace(out.String())
				frontendNeeded = (statusCode != "200" && statusCode != "301" && statusCode != "302")
			}
		}
	}

	// Se qualquer um dos serviços precisar de port-forward, retorne true
	return backendNeeded || frontendNeeded
}
