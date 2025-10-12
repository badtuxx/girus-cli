package k8s

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/badtuxx/girus-cli/internal/common"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	green = color.New(color.FgGreen).SprintFunc()
	bold  = color.New(color.Bold).SprintFunc()
)

// KubernetesClient wraper do cliente Kubernetes
type KubernetesClient struct {
	clientset *kubernetes.Clientset
}

// DeploymentConfig objeto que define as configurações de um deployment
type DeploymentConfig struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int32
	Port      int32
	Labels    map[string]string
	EnvVars   map[string]string
	Resources *ResourceConfig
}

// ResourceConfig defines resource requests and limits
type ResourceConfig struct {
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
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

// DeletePodGracefully remove o pod com grace period (permitindo que os containers terminem primeiro)
func (k *KubernetesClient) DeleteDeployGracefully(ctx context.Context, namespace, podName string) error {
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy:  &deletePolicy,
		GracePeriodSeconds: ptr.To(int64(30)),
	}

	err := k.clientset.AppsV1().Deployments(namespace).Delete(ctx, podName, deleteOptions)
	if err != nil {
		return fmt.Errorf("falha ao remover deploy %s no namespace %s: %w", podName, namespace, err)
	}

	return nil
}

func (k *KubernetesClient) DeleteDeploy(ctx context.Context, namespace, name string) error {
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
	}
	fmt.Printf("Removendo deployment %s...\n", name)
	err := k.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		_ = fmt.Errorf("falha ao buscar pelo deploy %s: %w", name, err)
		return err
	}

	return nil
}

// WaitForDeploymentDeletion espera pelo deployment ser removido
func (k *KubernetesClient) WaitForDeploymentDeletion(ctx context.Context, namespace, deployName string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, deployName, metav1.GetOptions{})
			if err != nil {
				// Se o deployment não for encontrado, significa que já foi removido
				if errors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("erro ao verificar deployment: %w", err)
			}
			// Deployment existe, espera pra checar novamente
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}
}

// StopDeployAndWait para o deployment e espera que seja removido
func (k *KubernetesClient) StopDeployAndWait(ctx context.Context, namespace, podName string) error {
	// First, delete the pod
	if err := k.DeleteDeployGracefully(ctx, namespace, podName); err != nil {
		return fmt.Errorf("falha ao esperar pelo deployment ser terminado gracefully: %w", err)
	}

	// Then wait for it to be completely removed
	if err := k.WaitForDeploymentDeletion(ctx, namespace, podName); err != nil {
		return fmt.Errorf("falha ao esperar pelo deployment ser terminado: %w", err)
	}

	return nil
}

// CreateDeployment cria um deployment do backend ou do frontend do girus
func (k *KubernetesClient) CreateDeployment(ctx context.Context, namespace, name string) error {
	image := fmt.Sprintf("linuxtips/%s:latest", name)
	labels := map[string]string{
		"app": name,
	}
	// Define o conteúdo das variáveis de ambiente dependendo de qual deployment está sendo criado
	var envs []corev1.EnvVar
	if name == "girus-backend" {
		envs = []corev1.EnvVar{
			{
				Name:  "PORT",
				Value: "8080",
			},
			{
				Name:  "GIN_MODE",
				Value: "release",
			},
			{
				Name: "LAB_DEFAULT_IMAGE",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						Optional: ptr.To(true),
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "girus-config",
						},
						Key: "lab.defaultImage",
					},
				},
			},
		}
	} else {
		envs = []corev1.EnvVar{}
	}

	// Define o conteúdo dos volumeMounts dependendo de qual deployment está sendo criado
	var volumeMount []corev1.VolumeMount
	if name == "girus-backend" {
		volumeMount = []corev1.VolumeMount{
			{
				Name:      "config-volume",
				MountPath: "/app/config",
			},
		}
	} else {
		volumeMount = []corev1.VolumeMount{
			{
				Name:      "nginx-config",
				MountPath: "/etc/nginx/conf.d",
			},
		}
	}

	// Define o conteúdo dos volumes montados dependendo de qual deployment está sendo criado
	var volumes []corev1.Volume
	if name == "girus-backend" {
		volumes = []corev1.Volume{
			{
				Name: "config-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						DefaultMode: ptr.To(int32(420)),
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "girus-config",
						},
					},
				},
			},
		}
	} else {
		volumes = []corev1.Volume{
			{
				Name: "nginx-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						DefaultMode: ptr.To(int32(420)),
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "nginx-config",
						},
					},
				},
			},
		}
	}

	// Define quais portas expor dependendo de qual deployment está sendo criado
	var containerPorts []corev1.ContainerPort
	if name == "girus-backend" {
		containerPorts = []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      "TCP",
			},
		}
	} else {
		containerPorts = []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 80,
				Protocol:      "TCP",
			},
		}
	}

	// Define o deployment que será criado
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.String, // Pode ser intstr.Int também
						StrVal: "25%",
					},
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.String, // Pode ser intstr.Int também
						StrVal: "25%",
					},
				},
			},
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "girus-sa",
					RestartPolicy:      corev1.RestartPolicyAlways,
					DNSPolicy:          corev1.DNSClusterFirst,
					Containers: []corev1.Container{
						{
							Name:                   strings.ReplaceAll(name, "girus-", ""), // Remove o "girus-" do nome do container
							Image:                  image,
							ImagePullPolicy:        corev1.PullIfNotPresent,
							Env:                    envs,
							Ports:                  containerPorts,
							VolumeMounts:           volumeMount,
							TerminationMessagePath: "/dev/termination-log",
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	_, err := k.clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("falha ao criar o deploy %s no namespace %s: %w", name, namespace, err)
	}
	fmt.Printf("%s: Deploy %s criado com sucesso!\n", green("SUCESSO:"), bold(name))
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

// UnmarshalConfigMapData Faz o parse dos dados do campo data do ConfigMap para um tipo genérico (T)
func UnmarshalConfigMapData[T any](data string) (T, error) {
	var result T
	if err := yaml.Unmarshal([]byte(data), &result); err != nil {
		return result, fmt.Errorf("falha ao fazer unmarshal do data: %w", err)
	}
	return result, nil
}

// GetConfigMapDataByKey Retorna o conteúdo do campo data de um ConfigMap
func (k *KubernetesClient) GetConfigMapDataByKey(ctx context.Context, namespace, name, key string) (string, error) {
	configMap, err := k.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("falha ao pegar o ConfigMap %s: %w", name, err)
	}

	data, exists := configMap.Data[key]
	if !exists {
		return "", fmt.Errorf("chave (key) %s não encontrada no ConfigMap %s", key, name)
	}

	return data, nil
}

// GetProgressFromConfigMap retrieves lab progress from a ConfigMap
func (k *KubernetesClient) GetProgressFromConfigMap(ctx context.Context, name string) (common.Progress, error) {
	// First, retrieve the specific data key from the ConfigMap
	progressData, err := k.GetConfigMapDataByKey(ctx, "girus", name, "progress")
	if err != nil {
		return common.Progress{}, fmt.Errorf("falha ao coletar os dados de progresso: %w", err)
	}

	// Then unmarshal the data into the LabProgress structure
	progress, err := UnmarshalConfigMapData[common.Progress](progressData)
	if err != nil {
		return common.Progress{}, fmt.Errorf("falha ao fazer parse/unmarshal dos dados de progresso do ConfigMap: %w", err)
	}

	return progress, nil
}

func (k *KubernetesClient) GetAllLabs(ctx context.Context) ([]common.Lab, error) {
	data, err := k.GetProgressFromConfigMap(ctx, "progresso.yaml")
	if err != nil {
		return nil, fmt.Errorf("falha ao coletar os dados de progresso: %w", err)
	}

	var labs []common.Lab
	for _, status := range data.Labs {
		labs = append(labs, common.Lab{Name: status.Name, Status: status.Status})
	}

	return labs, nil
}
