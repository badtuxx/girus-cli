package k8s

import (
	"bytes"
	"context"
	"fmt"
	"k8s.io/utils/ptr"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubernetesClient wraper do cliente Kubernetes
type KubernetesClient struct {
	clientset *kubernetes.Clientset
}

type DeploymentStatus struct {
	Name      string
	Namespace string
	Ready     bool
}

// NewKubernetesClient cria um novo cliente Kubernetes
func NewKubernetesClient() (*KubernetesClient, error) {
	// Path padrão do arquivo kubeconfig
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Cria a configuração a partir do arquivo kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar configuração: %w", err)
	}

	// Cria o clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar o clientset: %w", err)
	}

	return &KubernetesClient{clientset: clientset}, nil
}

// IsPodRunning checa se um pod está em execução
func (k *KubernetesClient) IsPodRunning(ctx context.Context, namespace, podName string) (bool, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("falha ao checar o pod %s no namespace do GIRUS %s: %w", podName, namespace, err)
	}

	// Checa se o pod está em fase de execução
	return pod.Status.Phase == corev1.PodRunning, nil
}

// ListRunningPods retorna todos os pods em execução num determinado namespace
func (k *KubernetesClient) ListRunningPods(ctx context.Context, namespace string) ([]string, error) {
	pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar todos os pods rodando no namespace do GIRUS %s: %w", namespace, err)
	}

	var runningPods []string
	for _, pod := range pods.Items {
		runningPods = append(runningPods, pod.Name)
	}

	return runningPods, nil
}

func (k *KubernetesClient) ScaleDeploy(ctx context.Context, namespace, name string, replicas int32) error {
	fmt.Printf("Escalonando deployment %s para %d replicas...\n", name, replicas)
	deploy, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		_ = fmt.Errorf("falha ao buscar pelo deploy %s: %w", name, err)
		return err
	}
	// Escala o deployment para o numero replicas desejado
	deploy.Spec.Replicas = ptr.To(replicas)
	_, err = k.clientset.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	if err != nil {
		_ = fmt.Errorf("falha ao atualizar o deploy %s: %w", name, err)
		return err
	}

	return nil
}

// waitForPodsReady espera até que os pods do Girus (backend e frontend) estejam prontos
func WaitForPodsReady(namespace string, timeout time.Duration) error {
	// Criar formatadores de cores
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

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
				fmt.Printf("%s %s: Pronto\n", green("SUCESSO:"), magenta("Backend"))
			} else {
				fmt.Printf("%s %s: %s\n", red("ERRO:"), magenta("Backend"), backendMessage)
			}
			if frontendReady {
				fmt.Printf("%s %s: Pronto\n", green("SUCESSO:"), magenta("Frontend"))
			} else {
				fmt.Printf("%s %s: %s\n", red("ERRO:"), magenta("Frontend"), frontendMessage)
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
			fmt.Printf("\n%s %s: Pronto\n", green("SUCESSO:"), magenta("Backend"))
			fmt.Printf("%s %s: Pronto\n", green("SUCESSO:"), magenta("Frontend"))
			fmt.Printf("%s %s: Respondendo\n", green("SUCESSO:"), magenta("Aplicação"))
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
	// Criar formatador de cores
	green := color.New(color.FgGreen).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	// Matar todos os processos de port-forward relacionados ao Girus para começar limpo
	fmt.Println("   Limpando port-forwards existentes...")
	exec.Command("bash", "-c", "pkill -f 'kubectl.*port-forward.*girus' || true").Run()
	time.Sleep(1 * time.Second)

	// Port-forward do backend em background
	fmt.Println("   Configurando port-forward para o backend (" + magenta("8080") + ")...")
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
			fmt.Printf("   %s %s conectado com sucesso!\n", green("SUCESSO:"), magenta("Backend"))
			break
		}
		fmt.Println("   Tentativa", i+1, "falhou, aguardando...")
		time.Sleep(1 * time.Second)
	}

	if !backendOK {
		return fmt.Errorf("não foi possível conectar ao backend após várias tentativas")
	}

	// ------------------------------------------------------------------------
	// Port-forward do frontend - ABORDAGEM MAIS SIMPLES E DIRETA
	// ------------------------------------------------------------------------
	fmt.Println("   Configurando port-forward para o frontend (" + magenta("8000") + ")...")

	// Tentar encontrar o script auxiliar para port-forward
	scriptPath := filepath.Join(os.Getenv("HOME"), ".girus", "port-forward.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		// Usar o script auxiliar que executa kubectl em background
		fmt.Println("   Iniciando port-forward via script auxiliar...")
		frontendCmd := exec.Command(scriptPath, namespace, "frontend")
		frontendCmd.Stdout = &bytes.Buffer{}
		frontendCmd.Stderr = &bytes.Buffer{}
		err = frontendCmd.Run()
		if err != nil {
			return fmt.Errorf("erro ao executar script de port-forward: %v", err)
		}
		// Tentar extrair o PID do port-forward
		pidExtractCmd := exec.Command("pgrep", "-f", "kubectl.*port-forward.*frontend")
		var pidOut bytes.Buffer
		pidExtractCmd.Stdout = &pidOut
		if pidExtractCmd.Run() == nil {
			fmt.Printf("   Port-forward iniciado com PID: %s\n", strings.TrimSpace(pidOut.String()))
		}
	} else {
		// Usar abordagem direta com kubectl
		frontendCmd := fmt.Sprintf("kubectl port-forward -n %s svc/girus-frontend 8000:80 --address 0.0.0.0 > /dev/null 2>&1 &", namespace)
		err = exec.Command("bash", "-c", frontendCmd).Run()
		if err != nil {
			return fmt.Errorf("erro ao iniciar port-forward do frontend: %v", err)
		}
	}

	// Verificar se o frontend está acessível
	fmt.Println("   Verificando conectividade do frontend...")
	frontendOK := false
	for i := 0; i < 5; i++ {
		frontendCheckCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8000")
		var out bytes.Buffer
		frontendCheckCmd.Stdout = &out
		if frontendCheckCmd.Run() == nil {
			statusCode := strings.TrimSpace(out.String())
			if statusCode == "200" || statusCode == "301" || statusCode == "302" {
				frontendOK = true
				fmt.Printf("   %s %s conectado com sucesso!\n", green("SUCESSO:"), magenta("Frontend"))
				break
			}
		}
		fmt.Println("   Tentativa", i+1, "falhou, aguardando...")
		time.Sleep(1 * time.Second)
	}

	if !frontendOK {
		return fmt.Errorf("não foi possível conectar ao frontend após várias tentativas")
	}

	return nil
}
