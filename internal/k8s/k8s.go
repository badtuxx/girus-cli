package k8s

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// waitForPodsReady espera até que os pods do Girus (backend e frontend) estejam prontos
func WaitForPodsReady(namespace string, timeout time.Duration) error {
	fmt.Println("\nAguardando os pods do Girus inicializarem...")

	start := time.Now()
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Inicializando Girus..."),
		progressbar.OptionSetWidth(80),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)

	backendReady := false
	frontendReady := false
	backendMessage := ""
	frontendMessage := ""

	for {
		if time.Since(start) > timeout {
			bar.Finish()
			fmt.Println("\nStatus atual dos componentes:")
			if backendReady {
				fmt.Printf("✅ Backend: Pronto\n")
			} else {
				fmt.Printf("❌ Backend: %s\n", backendMessage)
			}
			if frontendReady {
				fmt.Printf("✅ Frontend: Pronto\n")
			} else {
				fmt.Printf("❌ Frontend: %s\n", frontendMessage)
			}
			return fmt.Errorf("timeout ao esperar pelos pods do Girus (5 minutos)")
		}

		// Verificar o backend
		if !backendReady {
			br, msg, err := getPodStatus(namespace, "app=girus-backend")
			if err == nil {
				backendReady = br
				backendMessage = msg
			}
		}

		// Verificar o frontend
		if !frontendReady {
			fr, msg, err := getPodStatus(namespace, "app=girus-frontend")
			if err == nil {
				frontendReady = fr
				frontendMessage = msg
			}
		}

		// Se ambos estiverem prontos, vamos verificar a conectividade
		if backendReady && frontendReady {
			// Verificar se conseguimos acessar a API
			healthy, err := checkHealthEndpoint()
			if err != nil || !healthy {
				// Ainda não está respondendo, vamos continuar tentando
				bar.Add(1)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			bar.Finish()
			fmt.Println("\n✅ Backend: Pronto")
			fmt.Println("✅ Frontend: Pronto")
			fmt.Println("✅ Aplicação: Respondendo")
			return nil
		}

		bar.Add(1)
		time.Sleep(500 * time.Millisecond)
	}
}

// getPodStatus verifica o status de um pod e retorna uma mensagem descritiva
func getPodStatus(namespace, selector string) (bool, string, error) {
	// Verificar se o pod existe
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", selector, "-o", "jsonpath={.items[0].metadata.name}")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return false, "Pod não encontrado", err
	}

	podName := strings.TrimSpace(out.String())
	if podName == "" {
		return false, "Pod ainda não criado", nil
	}

	// Verificar a fase atual do pod
	phaseCmd := exec.Command("kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.phase}")
	var phaseOut bytes.Buffer
	phaseCmd.Stdout = &phaseOut

	err = phaseCmd.Run()
	if err != nil {
		return false, "Erro ao verificar status", err
	}

	phase := strings.TrimSpace(phaseOut.String())
	if phase != "Running" {
		return false, fmt.Sprintf("Status: %s", phase), nil
	}

	// Verificar se todos os containers estão prontos
	readyCmd := exec.Command("kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
	var readyOut bytes.Buffer
	readyCmd.Stdout = &readyOut

	err = readyCmd.Run()
	if err != nil {
		return false, "Erro ao verificar prontidão", err
	}

	readyStatus := strings.TrimSpace(readyOut.String())
	if readyStatus != "True" {
		return false, "Containers inicializando", nil
	}

	return true, "Pronto", nil
}

// checkHealthEndpoint verifica se a aplicação está respondendo ao endpoint de saúde
func checkHealthEndpoint() (bool, error) {
	// Verificar o mapeamento de porta do serviço
	cmd := exec.Command("kubectl", "get", "svc", "-n", "girus", "girus-backend", "-o", "jsonpath={.spec.ports[0].nodePort}")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		// Tentar verificar diretamente o endpoint interno se não encontrarmos o NodePort
		healthCmd := exec.Command("kubectl", "exec", "-n", "girus", "deploy/girus-backend", "--", "wget", "-q", "-O-", "-T", "2", "http://localhost:8080/api/v1/health")
		return healthCmd.Run() == nil, nil
	}

	nodePort := strings.TrimSpace(out.String())
	if nodePort == "" {
		// Porta não encontrada, tentar verificar o serviço internamente
		healthCmd := exec.Command("kubectl", "exec", "-n", "girus", "deploy/girus-backend", "--", "wget", "-q", "-O-", "-T", "2", "http://localhost:8080/api/v1/health")
		return healthCmd.Run() == nil, nil
	}

	// Tentar acessar via NodePort
	healthCmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", fmt.Sprintf("http://localhost:%s/api/v1/health", nodePort))
	var healthOut bytes.Buffer
	healthCmd.Stdout = &healthOut

	err = healthCmd.Run()
	if err != nil {
		return false, err
	}

	statusCode := strings.TrimSpace(healthOut.String())
	return statusCode == "200", nil
}

// setupPortForward configura port-forward para os serviços do Girus
func SetupPortForward(namespace string) error {
	// Matar todos os processos de port-forward relacionados ao Girus para começar limpo
	fmt.Println("   Limpando port-forwards existentes...")
	exec.Command("bash", "-c", "pkill -f 'kubectl.*port-forward.*girus' || true").Run()
	time.Sleep(1 * time.Second)

	// Port-forward do backend em background
	fmt.Println("   Configurando port-forward para o backend (8080)...")
	backendCmd := fmt.Sprintf("kubectl port-forward -n %s svc/girus-backend 8080:8080 --address 0.0.0.0 > /dev/null 2>&1 &", namespace)
	err := exec.Command("bash", "-c", backendCmd).Run()
	if err != nil {
		return fmt.Errorf("erro ao iniciar port-forward do backend: %v", err)
	}

	// Verificar conectividade do backend
	fmt.Println("   Verificando conectividade do backend...")
	backendOK := false
	for i := 0; i < 5; i++ {
		healthCmd := exec.Command("curl", "-s", "--max-time", "2", "http://localhost:8080/api/v1/health")
		if healthCmd.Run() == nil {
			backendOK = true
			break
		}
		if i < 4 {
			fmt.Println("   Tentativa", i+1, "falhou, aguardando...")
			time.Sleep(1 * time.Second)
		}
	}

	if !backendOK {
		return fmt.Errorf("não foi possível conectar ao backend")
	}

	fmt.Println("   ✅ Backend conectado com sucesso!")

	// ------------------------------------------------------------------------
	// Port-forward do frontend - ABORDAGEM MAIS SIMPLES E DIRETA
	// ------------------------------------------------------------------------
	fmt.Println("   Configurando port-forward para o frontend (8000)...")

	// Método 1: Execução direta via bash para o frontend
	frontendSuccess := false

	// Criar um script temporário para garantir execução correta
	scriptContent := `#!/bin/bash
# Mata qualquer processo existente na porta 8000
kill $(lsof -t -i:8000) 2>/dev/null || true
sleep 1
# Inicia o port-forward
nohup kubectl port-forward -n NAMESPACE svc/girus-frontend 8000:80 --address 0.0.0.0 > /dev/null 2>&1 &
echo $!  # Retorna o PID
`

	// Substituir NAMESPACE pelo namespace real
	scriptContent = strings.Replace(scriptContent, "NAMESPACE", namespace, 1)

	// Salvar em arquivo temporário
	tmpFile := filepath.Join(os.TempDir(), "girus_frontend_portforward.sh")
	os.WriteFile(tmpFile, []byte(scriptContent), 0755)
	defer os.Remove(tmpFile)

	// Executar o script
	fmt.Println("   Iniciando port-forward via script auxiliar...")
	cmdOutput, err := exec.Command("bash", tmpFile).Output()
	if err == nil {
		pid := strings.TrimSpace(string(cmdOutput))
		fmt.Println("   Port-forward iniciado com PID:", pid)

		// Aguardar o port-forward inicializar
		time.Sleep(2 * time.Second)

		// Verificar conectividade
		for i := 0; i < 5; i++ {
			checkCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8000")
			var out bytes.Buffer
			checkCmd.Stdout = &out

			if err := checkCmd.Run(); err == nil {
				statusCode := strings.TrimSpace(out.String())
				if statusCode == "200" || statusCode == "301" || statusCode == "302" {
					frontendSuccess = true
					break
				}
			}

			fmt.Println("   Verificação", i+1, "falhou, aguardando...")
			time.Sleep(2 * time.Second)
		}
	}

	// Se falhou, tentar um método alternativo como último recurso
	if !frontendSuccess {
		fmt.Println("   ⚠️ Tentando método alternativo direto...")

		// Método direto: executar o comando diretamente
		cmd := exec.Command("kubectl", "port-forward", "-n", namespace, "svc/girus-frontend", "8000:80", "--address", "0.0.0.0")

		// Redirecionar saída para /dev/null
		devNull, _ := os.Open(os.DevNull)
		defer devNull.Close()
		cmd.Stdout = devNull
		cmd.Stderr = devNull

		// Iniciar em background - compatível com múltiplos sistemas operacionais
		startBackgroundCmd(cmd)

		// Verificar conectividade
		time.Sleep(3 * time.Second)
		for i := 0; i < 3; i++ {
			checkCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8000")
			var out bytes.Buffer
			checkCmd.Stdout = &out

			if err := checkCmd.Run(); err == nil {
				statusCode := strings.TrimSpace(out.String())
				if statusCode == "200" || statusCode == "301" || statusCode == "302" {
					frontendSuccess = true
					break
				}
			}
			time.Sleep(1 * time.Second)
		}
	}

	// Último recurso - método absolutamente direto com deployment em vez de service
	if !frontendSuccess {
		fmt.Println("   🔄 Último recurso: port-forward ao deployment...")
		// Método com deployment em vez de service, que pode ser mais estável
		finalCmd := fmt.Sprintf("kubectl port-forward -n %s deployment/girus-frontend 8000:80 --address 0.0.0.0 > /dev/null 2>&1 &", namespace)
		exec.Command("bash", "-c", finalCmd).Run()

		// Verificação final
		time.Sleep(3 * time.Second)
		checkCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8000")
		var out bytes.Buffer
		checkCmd.Stdout = &out

		if checkCmd.Run() == nil {
			statusCode := strings.TrimSpace(out.String())
			if statusCode == "200" || statusCode == "301" || statusCode == "302" {
				frontendSuccess = true
			}
		}
	}

	// Verificar status final e retornar
	if !frontendSuccess {
		return fmt.Errorf("não foi possível estabelecer port-forward para o frontend após múltiplas tentativas")
	}

	fmt.Println("   ✅ Frontend conectado com sucesso!")
	return nil
}

// startBackgroundCmd inicia um comando em segundo plano de forma compatível com múltiplos sistemas operacionais
func startBackgroundCmd(cmd *exec.Cmd) error {
	// Iniciar o processo sem depender de atributos específicos da plataforma
	// que podem não estar disponíveis em todas as implementações do Go

	// Redirecionar saída e erro para /dev/null ou nul (Windows)
	devNull, _ := os.Open(os.DevNull)
	if devNull != nil {
		cmd.Stdout = devNull
		cmd.Stderr = devNull
		defer devNull.Close()
	}

	// Iniciar o processo
	err := cmd.Start()
	if err != nil {
		return err
	}

	// Registrar o PID para referência
	if cmd.Process != nil {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			pidDir := filepath.Join(homeDir, ".girus")
			os.MkdirAll(pidDir, 0755)
			os.WriteFile(filepath.Join(pidDir, "frontend.pid"),
				[]byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)
		}

		// Separar o processo do atual para evitar que seja terminado quando o processo pai terminar
		// Isso é uma alternativa portable ao uso de Setpgid
		go func() {
			cmd.Process.Release()
		}()
	}

	return nil
}
