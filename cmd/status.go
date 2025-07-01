package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/badtuxx/girus-cli/internal/common"
	"github.com/badtuxx/girus-cli/internal/k8s"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Estrutura para armazenar informações sobre os serviços expostos
type ServiceInfo struct {
	Name      string
	Type      string
	ClusterIP string
	Ports     string
	Age       string
}

// Estrutura para armazenar informações do pod
type PodInfo struct {
	Name     string
	Ready    string
	Status   string
	Restarts string
	Age      string
}

// Estrutura para armazenar uso de recursos
type ResourceUsage struct {
	CPU    string
	Memory string
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: common.T("Exibe o status atual do GIRUS", "Muestra el estado actual del GIRUS"),
	Long: common.T(`Exibe informações detalhadas sobre o estado atual do GIRUS, incluindo:
- Status do cluster
- Pods em execução (backend e frontend)
- Serviços expostos e portas
- Laboratórios instalados
- Uso de recursos
- Versão do CLI`, `Muestra información detallada sobre el estado actual de GIRUS, incluyendo:
- Estado del cluster
- Pods en ejecución (backend y frontend)
- Servicios expuestos y puertos
- Laboratorios instalados
- Uso de recursos
- Versión de la CLI`),
	Run: func(cmd *cobra.Command, args []string) {
		// Criar formatadores de cores
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		magenta := color.New(color.FgMagenta).SprintFunc() // Para informações importantes

		// Criar formatador para títulos
		headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()

		// Exibir cabeçalho
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println(headerColor(common.T("GIRUS STATUS", "GIRUS ESTADO")))
		fmt.Println(strings.Repeat("─", 80))

		// Verificar versão da CLI
		fmt.Printf(common.T("%s: %s\n", "%s: %s\n"), bold(common.T("Versão da CLI", "Versión de la CLI")), magenta(common.Version))

		// Verificar se o cluster existe
		fmt.Println("\n" + headerColor(common.T("Verificando Cluster...", "Verificando Cluster...")))
		clusterExists, clusterName := checkClusterExists()

		if !clusterExists {
			fmt.Println(red(common.T("Nenhum cluster Girus encontrado.", "Ningún cluster Girus encontrado.")))
			fmt.Println(common.T("  Use 'girus create cluster' para criar um novo cluster.", "  Use 'girus create cluster' para crear un nuevo cluster."))
			return
		}

		fmt.Printf("%s Cluster Girus '%s' está ativo\n", green("ATIVO"), magenta(clusterName))

		// Verificar namespace girus
		fmt.Println("\n" + headerColor(common.T("Verificando Namespace...", "Verificando Namespace...")))
		namespaceExists := checkNamespaceExists()

		if !namespaceExists {
			fmt.Println(red(common.T("Namespace 'girus' não encontrado no cluster.", "El namespace 'girus' no se encontró en el cluster.")))
			fmt.Println(common.T("  O cluster pode não ter sido criado corretamente.", "  El cluster puede no haberse creado correctamente."))
			return
		}

		fmt.Printf(common.T("%s Namespace '%s' está presente\n", "%s Namespace '%s' está presente\n"), green(common.T("ATIVO", "ACTIVO")), magenta("girus"))

		// Obter informações sobre os pods
		fmt.Println("\n" + headerColor(common.T("Componentes da Aplicação:", "Componentes de la Aplicación:")))
		backendStatus, frontendStatus := checkComponentStatus()

		fmt.Printf("   %s: %s\n", bold("Backend"), backendStatus)
		fmt.Printf("   %s: %s\n", bold("Frontend"), frontendStatus)

		// Obter informações sobre os pods detalhadas
		pods := getPodDetails()
		if len(pods) > 0 {
			fmt.Println("\n" + headerColor("Detalhes dos Pods:"))
			fmt.Printf("   %-35s %-10s %-10s %-10s %-10s\n",
				cyan("NOME"),
				cyan("PRONTO"),
				cyan("STATUS"),
				cyan("RESTARTS"),
				cyan("IDADE"))
			for _, pod := range pods {
				fmt.Printf("   %-35s %-10s %-10s %-10s %-10s\n",
					magenta(pod.Name), pod.Ready, pod.Status, pod.Restarts, pod.Age)
			}
		}

		// Obter informações sobre os serviços expostos
		services := getServiceDetails()
		if len(services) > 0 {
			fmt.Println("\n" + headerColor("Serviços Expostos:"))
			fmt.Printf("   %-20s %-10s %-15s %-20s %-10s\n",
				cyan("NOME"),
				cyan("TIPO"),
				cyan("CLUSTER-IP"),
				cyan("PORTAS"),
				cyan("IDADE"))
			for _, svc := range services {
				fmt.Printf("   %-20s %-10s %-15s %-20s %-10s\n",
					magenta(svc.Name), svc.Type, svc.ClusterIP, magenta(svc.Ports), svc.Age)
			}
		}

		// Verificar port-forwards ativos
		portForwards := getActivePortForwards()
		if len(portForwards) > 0 {
			fmt.Println("\n" + headerColor("Port-Forwards Ativos:"))
			for _, pf := range portForwards {
				parts := strings.Fields(pf)
				if len(parts) >= 2 {
					fmt.Printf("   %s %s\n", parts[0], magenta(strings.Join(parts[1:], " ")))
				} else {
					fmt.Printf("   %s\n", magenta(pf))
				}
			}
		}

		// Listar laboratórios instalados
		labs := getInstalledLabs()
		if len(labs) > 0 {
			fmt.Println("\n" + headerColor("Laboratórios Instalados:"))
			for i, lab := range labs {
				parts := strings.SplitN(lab, " - ", 2)
				if len(parts) == 2 {
					fmt.Printf("   %d. %s - %s\n", i+1, magenta(parts[0]), parts[1])
				} else {
					fmt.Printf("   %d. %s\n", i+1, magenta(lab))
				}
			}
		} else {
			fmt.Println("\n" + headerColor("Laboratórios Instalados:") + " Nenhum")
			fmt.Println("   Use " + cyan("'girus lab install <nome-do-lab>'") + " para instalar um laboratório")
		}

		// Reporta o status dos laboratories
		client, err := k8s.NewKubernetesClient()
		if err != nil {
			fmt.Println("Erro ao criar o cliente Kubernetes:", err)
			return
		}
		laboratorios, _ := client.GetAllLabs(context.Background())
		fmt.Println()
		fmt.Printf("%s\n", headerColor("Progresso dos Laboratórios:"))
		for _, lab := range laboratorios {
			fmt.Printf("%s: ", lab.Name)
			if lab.Status == "in-progress" {
				fmt.Printf("⏳ %s\n", magenta(lab.Status))
			} else {
				fmt.Printf("✅ %s\n", green(lab.Status))
			}

		}

		// Obter uso de recursos
		nodeResources := getNodeResources()
		fmt.Println("\n" + headerColor("Recursos do Cluster:"))
		fmt.Printf("   %s: %s\n", bold("CPU"), magenta(nodeResources.CPU))
		fmt.Printf("   %s: %s\n", bold("Memória"), magenta(nodeResources.Memory))

		// URL de acesso
		fmt.Println("\n" + headerColor("Acesso à Aplicação:"))
		url := getAccessURL()
		fmt.Printf("   %s\n", magenta(url))

		// Dicas e informações adicionais
		fmt.Println("\n" + headerColor("Dicas Rápidas:"))
		fmt.Println("   • Para listar laboratórios disponíveis: " + magenta("girus lab list"))
		fmt.Println("   • Para instalar um laboratório: " + magenta("girus lab install <nome>"))
		fmt.Println("   • Para excluir o cluster: " + magenta("girus delete cluster"))
		fmt.Println("   • Para atualizar a CLI: " + magenta("girus update"))

		fmt.Println(strings.Repeat("─", 80))
	},
}

// checkClusterExists verifica se o cluster Kind existe
func checkClusterExists() (bool, string) {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return false, ""
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if cluster == "girus" {
			return true, cluster
		}
	}

	return false, ""
}

// checkNamespaceExists verifica se o namespace girus existe
func checkNamespaceExists() bool {
	cmd := exec.Command("kubectl", "get", "namespace", "girus", "--no-headers", "--ignore-not-found")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "girus")
}

// checkComponentStatus verifica o status dos componentes backend e frontend
func checkComponentStatus() (string, string) {
	// Criar formatadores de cores
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// Verificar o backend
	backendCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-backend", "-o", "jsonpath={.items[0].status.phase}")
	backendOutput, err := backendCmd.Output()
	var backendStatus string
	if err == nil && len(backendOutput) > 0 {
		status := string(backendOutput)
		if status == "Running" {
			// Verificar se todos os containers estão prontos
			readyCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-backend", "-o", "jsonpath={.items[0].status.containerStatuses[0].ready}")
			readyOutput, err := readyCmd.Output()
			if err == nil && string(readyOutput) == "true" {
				backendStatus = green("Pronto")
			} else {
				backendStatus = yellow("Inicializando")
			}
		} else {
			backendStatus = yellow(status)
		}
	} else {
		backendStatus = red("Não encontrado")
	}

	// Verificar o frontend
	frontendCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-frontend", "-o", "jsonpath={.items[0].status.phase}")
	frontendOutput, err := frontendCmd.Output()
	var frontendStatus string
	if err == nil && len(frontendOutput) > 0 {
		status := string(frontendOutput)
		if status == "Running" {
			// Verificar se todos os containers estão prontos
			readyCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-frontend", "-o", "jsonpath={.items[0].status.containerStatuses[0].ready}")
			readyOutput, err := readyCmd.Output()
			if err == nil && string(readyOutput) == "true" {
				frontendStatus = green("Pronto")
			} else {
				frontendStatus = yellow("Inicializando")
			}
		} else {
			frontendStatus = yellow(status)
		}
	} else {
		frontendStatus = red("Não encontrado")
	}

	return backendStatus, frontendStatus
}

// getPodDetails obtém detalhes sobre os pods
func getPodDetails() []PodInfo {
	cmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-o", "custom-columns=NAME:.metadata.name,READY:.status.containerStatuses[0].ready,STATUS:.status.phase,RESTARTS:.status.containerStatuses[0].restartCount,AGE:.metadata.creationTimestamp")
	output, err := cmd.Output()
	if err != nil {
		return []PodInfo{}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return []PodInfo{}
	}

	var pods []PodInfo
	for i, line := range lines {
		if i == 0 {
			// Pular o cabeçalho
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			// Calcular idade relativa
			timeStr := fields[4]
			t, err := time.Parse(time.RFC3339, timeStr)
			age := timeStr
			if err == nil {
				duration := time.Since(t)
				if duration.Hours() > 24 {
					days := int(duration.Hours() / 24)
					age = fmt.Sprintf("%dd", days)
				} else if duration.Hours() >= 1 {
					age = fmt.Sprintf("%dh", int(duration.Hours()))
				} else {
					age = fmt.Sprintf("%dm", int(duration.Minutes()))
				}
			}

			readyStatus := "False"
			if fields[1] == "true" {
				readyStatus = "True"
			}

			pods = append(pods, PodInfo{
				Name:     fields[0],
				Ready:    readyStatus,
				Status:   fields[2],
				Restarts: fields[3],
				Age:      age,
			})
		}
	}

	return pods
}

// getServiceDetails obtém detalhes sobre os serviços
func getServiceDetails() []ServiceInfo {
	cmd := exec.Command("kubectl", "get", "services", "-n", "girus", "-o", "custom-columns=NAME:.metadata.name,TYPE:.spec.type,CLUSTER-IP:.spec.clusterIP,PORT:.spec.ports[*].port,AGE:.metadata.creationTimestamp")
	output, err := cmd.Output()
	if err != nil {
		return []ServiceInfo{}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return []ServiceInfo{}
	}

	var services []ServiceInfo
	for i, line := range lines {
		if i == 0 {
			// Pular o cabeçalho
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			// Obter portas expostas
			portsCmd := exec.Command("kubectl", "get", "service", fields[0], "-n", "girus", "-o", "jsonpath={.spec.ports[*].port}:{.spec.ports[*].nodePort}")
			portsOutput, err := portsCmd.Output()
			ports := fields[3]
			if err == nil && len(portsOutput) > 0 {
				portParts := strings.Split(string(portsOutput), ":")
				if len(portParts) >= 2 && portParts[1] != "" {
					ports = fmt.Sprintf("%s:%s", portParts[0], portParts[1])
				}
			}

			// Calcular idade relativa
			timeStr := fields[4]
			t, err := time.Parse(time.RFC3339, timeStr)
			age := timeStr
			if err == nil {
				duration := time.Since(t)
				if duration.Hours() > 24 {
					days := int(duration.Hours() / 24)
					age = fmt.Sprintf("%dd", days)
				} else if duration.Hours() >= 1 {
					age = fmt.Sprintf("%dh", int(duration.Hours()))
				} else {
					age = fmt.Sprintf("%dm", int(duration.Minutes()))
				}
			}

			services = append(services, ServiceInfo{
				Name:      fields[0],
				Type:      fields[1],
				ClusterIP: fields[2],
				Ports:     ports,
				Age:       age,
			})
		}
	}

	return services
}

// getActivePortForwards obtém os port-forwards ativos
func getActivePortForwards() []string {
	cmd := exec.Command("ps", "-ef")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return []string{}
	}

	var portForwards []string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, "kubectl port-forward") && strings.Contains(line, "girus") {
			parts := strings.Fields(line)
			if len(parts) > 8 {
				portForwards = append(portForwards, strings.Join(parts[7:], " "))
			}
		}
	}

	return portForwards
}

// getInstalledLabs obtém os laboratórios instalados
func getInstalledLabs() []string {
	// Verificar se há um cluster Girus ativo
	if !checkNamespaceExists() {
		return []string{}
	}

	// Verificar se o backend está pronto
	backendCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-backend", "-o", "jsonpath={.items[0].status.phase}")
	backendOutput, err := backendCmd.Output()
	if err != nil || string(backendOutput) != "Running" {
		return []string{}
	}

	// Fazer uma solicitação para a API para obter a lista de laboratórios
	apiCmd := exec.Command("kubectl", "exec", "-n", "girus", "deploy/girus-backend", "--",
		"wget", "-q", "-O-", "http://localhost:8080/api/v1/templates")
	apiOutput, err := apiCmd.Output()

	if err != nil {
		return []string{}
	}

	// Processar a resposta JSON
	var response struct {
		Templates []struct {
			Name        string `json:"name"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"templates"`
	}

	if err := json.Unmarshal(apiOutput, &response); err != nil {
		return []string{}
	}

	var labs []string
	for _, template := range response.Templates {
		labs = append(labs, fmt.Sprintf("%s - %s", template.Name, template.Title))
	}

	return labs
}

// getNodeResources obtém informações sobre os recursos do cluster
func getNodeResources() ResourceUsage {
	cpuUsage := "Não disponível"
	memoryUsage := "Não disponível"

	// Abordagem 1: Tentar kubectl top nodes
	topNodesCmd := exec.Command("kubectl", "top", "nodes", "--no-headers")
	topNodesOutput, err := topNodesCmd.Output()
	if err == nil && len(topNodesOutput) > 0 {
		fields := strings.Fields(string(topNodesOutput))
		if len(fields) >= 3 {
			// Formatar CPU
			cpuStr := fields[2]
			if strings.HasSuffix(cpuStr, "m") {
				// Converter milicores para cores
				cpuMilli, _ := strconv.Atoi(strings.TrimSuffix(cpuStr, "m"))
				cpuUsage = fmt.Sprintf("%.2f cores (em uso)", float64(cpuMilli)/1000.0)
			} else {
				cpuUsage = fmt.Sprintf("%s cores (em uso)", cpuStr)
			}

			// Formatar memória
			memStr := fields[3]
			memUsage := formatMemory(memStr)
			memoryUsage = fmt.Sprintf("%s (em uso)", memUsage)

			return ResourceUsage{
				CPU:    cpuUsage,
				Memory: memoryUsage,
			}
		}
	}

	// Abordagem 2: Obter recursos através dos pods
	topPodsCmd := exec.Command("kubectl", "top", "pods", "-n", "girus", "--no-headers")
	topPodsOutput, err := topPodsCmd.Output()
	if err == nil && len(topPodsOutput) > 0 {
		lines := strings.Split(strings.TrimSpace(string(topPodsOutput)), "\n")
		totalCPU := 0
		totalMemory := 0.0
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// Extrair valores de CPU (em m)
				cpuStr := fields[1]
				cpuStr = strings.TrimSuffix(cpuStr, "m")
				var cpu int
				fmt.Sscanf(cpuStr, "%d", &cpu)
				totalCPU += cpu

				// Extrair valores de memória (em Mi)
				memStr := fields[2]
				memStr = strings.TrimSuffix(memStr, "Mi")
				var mem float64
				fmt.Sscanf(memStr, "%f", &mem)
				totalMemory += mem
			}
		}
		if len(lines) > 0 {
			// Formatar CPU
			if totalCPU >= 1000 {
				cpuUsage = fmt.Sprintf("%.2f cores (pods em execução)", float64(totalCPU)/1000.0)
			} else {
				cpuUsage = fmt.Sprintf("%dm (%.2f cores) (pods em execução)", totalCPU, float64(totalCPU)/1000.0)
			}

			// Formatar memória
			if totalMemory >= 1024 {
				memoryUsage = fmt.Sprintf("%.2f GB (pods em execução)", totalMemory/1024)
			} else {
				memoryUsage = fmt.Sprintf("%.1f MB (pods em execução)", totalMemory)
			}

			return ResourceUsage{
				CPU:    cpuUsage,
				Memory: memoryUsage,
			}
		}
	}

	// Abordagem 3: Obter informações através do kubectl describe node
	describeNodeCmd := exec.Command("kubectl", "describe", "node")
	describeOutput, err := describeNodeCmd.Output()
	if err == nil {
		describeStr := string(describeOutput)

		// Procurar alocação total de recursos
		cpuTotalIdx := strings.Index(describeStr, "Capacity:")
		cpuTotal := ""
		memTotal := ""

		if cpuTotalIdx >= 0 {
			// Extrair seção Capacity
			capacitySection := describeStr[cpuTotalIdx : cpuTotalIdx+200]
			capacityLines := strings.Split(capacitySection, "\n")

			for _, line := range capacityLines {
				if strings.Contains(line, "cpu:") {
					cpuFields := strings.Fields(line)
					if len(cpuFields) >= 2 {
						cpuTotal = cpuFields[1]
					}
				}
				if strings.Contains(line, "memory:") {
					memFields := strings.Fields(line)
					if len(memFields) >= 2 {
						memTotal = memFields[1]
					}
				}
			}
		}

		// Procurar alocação usada de recursos
		cpuAllocIdx := strings.Index(describeStr, "Allocated resources:")
		cpuAlloc := ""
		memAlloc := ""

		if cpuAllocIdx >= 0 {
			// Extrair seção Allocated resources
			allocSection := describeStr[cpuAllocIdx : cpuAllocIdx+500]
			allocLines := strings.Split(allocSection, "\n")

			for _, line := range allocLines {
				if strings.Contains(line, "cpu") && !strings.Contains(line, "limits") {
					cpuFields := strings.Fields(line)
					if len(cpuFields) >= 2 {
						cpuAlloc = cpuFields[1]
					}
				}
				if strings.Contains(line, "memory") && !strings.Contains(line, "limits") {
					memFields := strings.Fields(line)
					if len(memFields) >= 2 {
						memAlloc = memFields[1]
					}
				}
			}
		}

		// Formatar saída com alocação/total quando disponível
		if cpuTotal != "" {
			if cpuAlloc != "" {
				cpuUsage = fmt.Sprintf("%s de %s cores alocados", cpuAlloc, cpuTotal)
			} else {
				cpuUsage = fmt.Sprintf("%s cores (total)", cpuTotal)
			}
		}

		if memTotal != "" {
			memTotalFormatted := formatMemory(memTotal)
			if memAlloc != "" {
				memAllocFormatted := formatMemory(memAlloc)
				memoryUsage = fmt.Sprintf("%s de %s alocados", memAllocFormatted, memTotalFormatted)
			} else {
				memoryUsage = fmt.Sprintf("%s (total)", memTotalFormatted)
			}
		}

		if cpuUsage != "Não disponível" || memoryUsage != "Não disponível" {
			return ResourceUsage{
				CPU:    cpuUsage,
				Memory: memoryUsage,
			}
		}
	}

	// Abordagem 4: Verificar a definição do nó Kind
	kindNodeCmd := exec.Command("kubectl", "get", "node", "-o", "jsonpath={.items[0].status.capacity}")
	kindOutput, err := kindNodeCmd.Output()
	if err == nil && len(kindOutput) > 0 {
		// Parsear a saída JSON
		capacityStr := string(kindOutput)

		// Extrair valores de CPU e memória
		if strings.Contains(capacityStr, "cpu") {
			cpuStart := strings.Index(capacityStr, "cpu") + 5
			cpuEnd := strings.Index(capacityStr[cpuStart:], "\"") + cpuStart
			if cpuEnd > cpuStart {
				cpuValue := capacityStr[cpuStart:cpuEnd]
				cpuUsage = fmt.Sprintf("%s cores (total)", cpuValue)
			}
		}

		if strings.Contains(capacityStr, "memory") {
			memStart := strings.Index(capacityStr, "memory") + 9
			memEnd := strings.Index(capacityStr[memStart:], "\"") + memStart
			if memEnd > memStart {
				memValue := capacityStr[memStart:memEnd]
				memFormatted := formatMemory(memValue)
				memoryUsage = fmt.Sprintf("%s (total)", memFormatted)
			}
		}
	}

	return ResourceUsage{
		CPU:    cpuUsage,
		Memory: memoryUsage,
	}
}

// formatMemory converte valores de memória para uma representação mais legível
func formatMemory(memStr string) string {
	// Remover unidades
	var value float64
	var unit string

	// Extrair número e unidade
	for i, char := range memStr {
		if !unicode.IsDigit(char) && char != '.' {
			numStr := memStr[:i]
			unit = memStr[i:]
			value, _ = strconv.ParseFloat(numStr, 64)
			break
		}
	}

	// Se não tiver unidade, assumir bytes
	if unit == "" {
		value, _ = strconv.ParseFloat(memStr, 64)
		unit = "B"
	}

	// Converter para unidade mais apropriada
	switch strings.ToUpper(unit) {
	case "KI", "KB", "K":
		if value > 1024 {
			return fmt.Sprintf("%.2f MB", value/1024)
		}
		return fmt.Sprintf("%.0f KB", value)
	case "MI", "MB", "M":
		if value > 1024 {
			return fmt.Sprintf("%.2f GB", value/1024)
		}
		return fmt.Sprintf("%.0f MB", value)
	case "GI", "GB", "G":
		if value > 1024 {
			return fmt.Sprintf("%.2f TB", value/1024)
		}
		return fmt.Sprintf("%.2f GB", value)
	case "TI", "TB", "T":
		return fmt.Sprintf("%.2f TB", value)
	case "B":
		if value > 1073741824 { // 1GB
			return fmt.Sprintf("%.2f GB", value/1073741824)
		}
		if value > 1048576 { // 1MB
			return fmt.Sprintf("%.2f MB", value/1048576)
		}
		if value > 1024 { // 1KB
			return fmt.Sprintf("%.2f KB", value/1024)
		}
		return fmt.Sprintf("%.0f B", value)
	default:
		return memStr
	}
}

// getAccessURL obtém a URL de acesso à aplicação
func getAccessURL() string {
	// Verificar se o serviço frontend existe
	frontendCmd := exec.Command("kubectl", "get", "service", "girus-frontend", "-n", "girus", "--no-headers", "--ignore-not-found")
	_, err := frontendCmd.Output()
	if err != nil {
		return "Não disponível"
	}

	// Verificar se há port-forward ativo
	portForwards := getActivePortForwards()
	for _, pf := range portForwards {
		if strings.Contains(pf, "girus-frontend") && strings.Contains(pf, "8000:") {
			return "http://localhost:8000"
		}
	}

	// Verificar nodePort
	nodePortCmd := exec.Command("kubectl", "get", "service", "girus-frontend", "-n", "girus", "-o", "jsonpath={.spec.ports[0].nodePort}")
	nodePortOutput, err := nodePortCmd.Output()
	if err == nil && len(nodePortOutput) > 0 {
		return fmt.Sprintf("http://localhost:%s", string(nodePortOutput))
	}

	// Se não encontrou nenhuma forma de acesso
	return "Execute 'kubectl port-forward svc/girus-frontend -n girus 8000:80' para acessar"
}
