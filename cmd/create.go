package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	deployFile      string
	clusterName     string
	verboseMode     bool
	useExternalFile bool
	labFile         string
	skipPortForward bool
	skipBrowser     bool
)

// Conteúdo do template básico do Linux
const basicLinuxTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: basic-linux-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: linux-basics
    title: "Introdução ao Linux"
    description: "Laboratório básico para praticar comandos Linux essenciais e conceitos fundamentais"
    duration: 10m
    image: "linuxtips/girus-devops:0.1"
    tasks:
      - name: "Navegação básica"
        description: "Aprenda os comandos essenciais para navegar no sistema de arquivos Linux"
        steps:
          - "Comece verificando qual é seu diretório atual com o comando:"
          - "` + "`" + `pwd` + "`" + `"
          - "Liste todos os arquivos (incluindo ocultos) do diretório atual:"
          - "` + "`" + `ls -la` + "`" + `"
          - "Crie um novo diretório para praticar:"
          - "` + "`" + `mkdir lab-practice` + "`" + `"
          - "Entre no diretório criado:"
          - "` + "`" + `cd lab-practice` + "`" + `"
          - "Crie alguns arquivos para praticar:"
          - "` + "`" + `touch file1.txt file2.txt file3.txt` + "`" + `"
        tips:
          - type: "info"
            title: "Dica: Atalhos úteis"
            content: "Use cd .. para voltar um diretório acima, e cd ~ para ir direto para seu diretório home. O comando ls tem muitas opções úteis: ls -l (formato detalhado), ls -a (mostra arquivos ocultos), ls -h (tamanhos legíveis por humanos)."
        validation:
          - command: "test -d lab-practice && echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "Diretório 'lab-practice' não foi criado"
          - command: "test -f lab-practice/file1.txt && echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "Arquivos de teste não foram criados"

      - name: "Manipulação de arquivos"
        description: "Aprenda a criar, editar e gerenciar arquivos no Linux"
        steps:
          - "Crie um arquivo de texto usando o editor Vim:"
          - "` + "`" + `vim notes.txt` + "`" + `"
          - "Para adicionar texto, pressione 'i' para entrar no modo de inserção e comece a digitar. Pressione 'Esc' para sair do modo de inserção e 'Shift+:' para salvar e sair."
          - "Salve e saia do Vim com 'Shift+:' e digite 'wq'"
          - "Visualize o conteúdo do arquivo:"
          - "` + "`" + `cat notes.txt` + "`" + `"
          - "Copie um arquivo para outro nome:"
          - "` + "`" + `cp notes.txt notes-backup.txt` + "`" + `"
          - "Compare os dois arquivos:"
          - "` + "`" + `diff notes.txt notes-backup.txt` + "`" + `"
          - "Adicione mais conteúdo ao arquivo original:"
          - "` + "`" + `echo 'Nova linha adicionada!' >> notes.txt` + "`" + `"
          - "Compare novamente os arquivos:"
          - "` + "`" + `diff notes.txt notes-backup.txt` + "`" + `"
        tips:
          - type: "warning"
            title: "Atenção: Redirecionamentos"
            content: "O símbolo > redireciona a saída e sobrescreve o arquivo existente, enquanto >> adiciona ao final do arquivo sem apagar o conteúdo anterior."
        validation:
          - command: "test -f lab-practice/notes.txt && echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "Arquivo notes.txt não foi criado"
          - command: "test -f lab-practice/notes-backup.txt && echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "Arquivo de backup não foi criado"

      - name: "Processos e monitoramento"
        description: "Aprenda a monitorar e gerenciar processos no Linux"
        steps:
          - "Veja os processos em execução:"
          - "` + "`" + `ps aux` + "`" + `"
          - "Monitore os processos e recursos em tempo real:"
          - "` + "`" + `htop` + "`" + `"
          - "Pressione 'q' para sair do top"
          - "Execute um processo em segundo plano:"
          - "` + "`" + `sleep 300 &` + "`" + `"
          - "Veja o processo em execução:"
          - "` + "`" + `ps aux | grep sleep` + "`" + `"
          - "Termine o processo sleep:"
          - "` + "`" + `pkill sleep` + "`" + `"
        tips:
          - type: "tip"
            title: "Alternativa ao top"
            content: "O comando htop é uma versão melhorada do top com interface colorida e interativa. Instale-o com 'apt install htop' em sistemas Debian/Ubuntu."
        validation:
          - command: "ps aux | grep -v grep | grep -q sleep || echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "O processo sleep não foi encerrado corretamente"`

// Conteúdo do template do Kubernetes
const basicKubernetesTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: kubernetes-basics-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: kubernetes-basics
    title: "Fundamentos de Kubernetes"
    description: "Aprenda comandos básicos do Kubernetes para gerenciar recursos em um cluster"
    duration: 15m
    image: "linuxtips/girus-kind-single-node:0.1"
    tasks:
      - name: "Explorando o Cluster"
        description: "Aprenda a verificar os componentes básicos de um cluster Kubernetes"
        steps:
          - "Verifique se o kubectl está instalado:"
          - "` + "`" + `kubectl version` + "`" + `"
          - "Verifique os nós do cluster executando:"
          - "` + "`" + `kubectl get nodes` + "`" + `"
          - "Veja informações mais detalhadas sobre os nós:"
          - "` + "`" + `kubectl get nodes -o wide` + "`" + `"
          - "Verifique os namespaces disponíveis:"
          - "` + "`" + `kubectl get namespaces` + "`" + `"
          - "Veja os pods do namespace kube-system (componentes internos do K8s):"
          - "` + "`" + `kubectl get pods -n kube-system` + "`" + `"
          - "Examine os detalhes de um nó específico (substitua [nome-do-nó] pelo nome real):"
          - "` + "`" + `kubectl describe node [nome-do-nó]` + "`" + `"
        tips:
          - type: "info"
            title: "kubectl - Sua ferramenta principal"
            content: "O kubectl é a ferramenta de linha de comando para interagir com o Kubernetes. Sempre que tiver dúvidas sobre um comando, use kubectl --help ou kubectl [comando] --help."
          - type: "tip"
            title: "Formatos de saída"
            content: "Você pode mudar o formato de saída de qualquer comando kubectl usando -o yaml, -o json, -o wide. Para visualizações compactas, -o custom-columns é útil."
        validation:
          - command: "kubectl get nodes -o name | wc -l | tr -d ' '"
            expectedOutput: "1"
            errorMessage: "Não foi possível listar os nós do cluster"
      
      - name: "Criando um Pod"
        description: "Aprenda a criar e gerenciar pods, que são a menor unidade executável no Kubernetes"
        steps:
          - "Crie um namespace para o exercício:"
          - "` + "`" + `kubectl create namespace k8s-demo` + "`" + `"
          - "Crie um arquivo pod.yaml com o seguinte conteúdo:"
          - |
            ` + "```yaml" + `
            apiVersion: v1
            kind: Pod
            metadata:
              name: nginx-pod
              namespace: k8s-demo
              labels:
                app: nginx
            spec:
              containers:
              - name: nginx
                image: nginx:latest
                ports:
                - containerPort: 80
            ` + "```" + `
          - "Crie o pod executando:"
          - "` + "`" + `kubectl apply -f pod.yaml` + "`" + `"
          - "Verifique se o pod está rodando:"
          - "` + "`" + `kubectl get pods -n k8s-demo` + "`" + `"
          - "Verifique os logs do pod:"
          - "` + "`" + `kubectl logs nginx-pod -n k8s-demo` + "`" + `"
          - "Acesse o shell do pod:"
          - "` + "`" + `kubectl exec -it nginx-pod -n k8s-demo -- /bin/bash` + "`" + `"
          - "Dentro do container, verifique se o nginx está rodando:"
          - "` + "`" + `curl localhost:80` + "`" + `"
          - "Digite 'exit' para sair do container"
        tips:
          - type: "warning"
            title: "Cuidado com imagens 'latest'"
            content: "Em ambientes de produção, evite usar a tag 'latest' para imagens. Prefira versões específicas para garantir consistência e evitar surpresas em atualizações automáticas."
          - type: "info"
            title: "Namespaces"
            content: "Os namespaces são uma forma de criar isolamento virtual dentro do cluster. Eles permitem separar recursos, definir cotas e gerenciar permissões para diferentes equipes ou aplicações."
        validation:
          - command: "kubectl get pod nginx-pod -n k8s-demo -o jsonpath='{.status.phase}' 2>/dev/null || echo ''"
            expectedOutput: "Running"
            errorMessage: "O Pod nginx-pod não está no estado Running"`

// Conteúdo do template do Docker
const basicDockerTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: docker-basics-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: docker-basics
    title: "Fundamentos de Docker"
    description: "Aprenda comandos básicos do Docker para criar, gerenciar e executar containers"
    duration: 15m
    timerEnabled: true
    maxDuration: 15m
    image: "linuxtips/girus-devops:0.1"
    youtubeVideo: "https://www.youtube.com/watch?v=0cDj7citEjE"
    privileged: true
    tasks:
      - name: "Explorando o Ambiente Docker"
        description: "Aprenda a verificar o ambiente Docker e seus componentes básicos"
        steps:
          - "Verifique a versão do Docker instalada:"
          - "` + "`" + `docker --version` + "`" + `"
          - "Verifique informações detalhadas sobre a instalação do Docker:"
          - "` + "`" + `docker info` + "`" + `"
          - "Liste as imagens disponíveis localmente:"
          - "` + "`" + `docker images` + "`" + `"
          - "Liste todos os containers (incluindo os parados):"
          - "` + "`" + `docker ps -a` + "`" + `"
          - "Verifique as redes Docker disponíveis:"
          - "` + "`" + `docker network ls` + "`" + `"
        tips:
          - type: "info"
            title: "Docker CLI - Sua ferramenta principal"
            content: "O comando docker é a ferramenta de linha de comando para interagir com o Docker. Sempre que tiver dúvidas sobre um comando, use docker --help ou docker [comando] --help."
          - type: "tip"
            title: "Formatos de saída"
            content: "Você pode mudar o formato de saída de qualquer comando docker usando --format. Por exemplo: docker ps --format '{{.Names}} {{.Status}}'"
        validation:
          - command: "docker info &>/dev/null && echo 'success' || echo 'error'"
            expectedOutput: "success"
            errorMessage: "Não foi possível acessar o daemon Docker. Verifique se o serviço está em execução."
      
      - name: "Executando Containers"
        description: "Aprenda a executar e gerenciar containers Docker"
        steps:
          - "Execute um container hello-world para testar o ambiente:"
          - "` + "`" + `docker run hello-world` + "`" + `"
          - "Execute um container nginx em modo detached (background):"
          - "` + "`" + `docker run -d --name meu-nginx -p 8080:80 nginx` + "`" + `"
          - "Verifique se o container está em execução:"
          - "` + "`" + `docker ps` + "`" + `"
          - "Acesse o nginx através do navegador ou usando curl:"
          - "` + "`" + `curl localhost:8080` + "`" + `"
          - "Veja os logs do container:"
          - "` + "`" + `docker logs meu-nginx` + "`" + `"
          - "Pare o container:"
          - "` + "`" + `docker stop meu-nginx` + "`" + `"
          - "Inicie o container novamente:"
          - "` + "`" + `docker start meu-nginx` + "`" + `"
          - "Execute um container interativo (e descartável) do Ubuntu:"
          - "` + "`" + `docker run -it --rm ubuntu bash` + "`" + `"
          - "No terminal do container, execute alguns comandos:"
          - "` + "`" + `ls -la` + "`" + `"
          - "` + "`" + `cat /etc/os-release` + "`" + `"
          - "Digite 'exit' para sair do container"
        tips:
          - type: "warning"
            title: "Portas expostas"
            content: "Lembre-se que para acessar serviços dentro de um container a partir do host, você precisa mapear as portas com a flag -p. Exemplo: -p [porta-host]:[porta-container]"
          - type: "info"
            title: "Modos de execução"
            content: "O Docker permite executar containers em modo detached (-d), interativo (-it) ou com uma combinação de flags. Use --rm para remover automaticamente o container quando ele for finalizado."
        validation:
          - command: "docker ps -a --format '{{.Names}}' | grep -w meu-nginx || echo ''"
            expectedOutput: "meu-nginx"
            errorMessage: "O container meu-nginx não foi criado"`

// Conteúdo do template de Administração de Usuários Linux
const linuxUsersTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: linux-users-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: linux-users-admin
    title: "Administrando Usuários no Linux"
    description: "Aprenda a criar, modificar e gerenciar contas de usuários e grupos no Linux"
    duration: 10m
    timerEnabled: true
    maxDuration: 10m
    image: "linuxtips/girus-devops:0.1"
    tasks:
      - name: "Gerenciamento de Usuários"
        description: "Aprenda a criar, modificar e remover contas de usuários"
        steps:
          - "Visualize informações sobre o usuário atual:"
          - "` + "`" + `id` + "`" + `"
          - "Veja a lista de todos os usuários no sistema:"
          - "` + "`" + `cat /etc/passwd` + "`" + `"
          - "Crie um novo usuário chamado 'testuser':"
          - "` + "`" + `sudo useradd testuser` + "`" + `"
          - "Defina uma senha para o novo usuário:"
          - "` + "`" + `sudo passwd testuser` + "`" + `"
          - "Veja as informações do usuário criado:"
          - "` + "`" + `id testuser` + "`" + `"
          - "Modifique o shell padrão do usuário para bash:"
          - "` + "`" + `sudo usermod -s /bin/bash testuser` + "`" + `"
          - "Adicione um comentário (nome completo) ao usuário:"
          - "` + "`" + `sudo usermod -c \"Usuário de Teste\" testuser` + "`" + `"
          - "Veja as alterações no arquivo /etc/passwd:"
          - "` + "`" + `grep testuser /etc/passwd` + "`" + `"
        tips:
          - type: "info"
            title: "Comandos para Gerenciamento de Usuários"
            content: "Os comandos principais para gerenciar usuários são: useradd (criar), usermod (modificar), userdel (remover), passwd (definir senha) e id (visualizar informações)."
          - type: "warning"
            title: "Segurança"
            content: "Sempre use senhas fortes e gerencie com cuidado as contas de usuários com privilégios administrativos."
        validation:
          - command: "grep testuser /etc/passwd | wc -l"
            expectedOutput: "1"
            errorMessage: "O usuário testuser não foi criado corretamente"
      
      - name: "Gerenciamento de Grupos"
        description: "Aprenda a criar e gerenciar grupos de usuários"
        steps:
          - "Veja a lista de todos os grupos no sistema:"
          - "` + "`" + `cat /etc/group` + "`" + `"
          - "Crie um novo grupo chamado 'projeto':"
          - "` + "`" + `sudo groupadd projeto` + "`" + `"
          - "Adicione o usuário 'testuser' ao novo grupo:"
          - "` + "`" + `sudo usermod -aG projeto testuser` + "`" + `"
          - "Verifique se o usuário está no grupo:"
          - "` + "`" + `groups testuser` + "`" + `"
          - "Crie outro usuário para o grupo projeto:"
          - "` + "`" + `sudo useradd -G projeto colaborador` + "`" + `"
          - "Defina uma senha para o novo usuário:"
          - "` + "`" + `sudo passwd colaborador` + "`" + `"
          - "Liste todos os usuários do grupo projeto:"
          - "` + "`" + `grep projeto /etc/group` + "`" + `"
        tips:
          - type: "info"
            title: "Comandos para Gerenciamento de Grupos"
            content: "Os comandos principais para gerenciar grupos são: groupadd (criar), groupmod (modificar), groupdel (remover) e groups (listar grupos de um usuário)."
          - type: "tip"
            title: "Opção -a no usermod"
            content: "Ao adicionar um usuário a um grupo com usermod, sempre use a opção -a junto com -G para adicionar aos grupos existentes, sem removê-lo de outros grupos."
        validation:
          - command: "grep projeto /etc/group | grep -E 'testuser|colaborador' | wc -l"
            expectedOutput: "1"
            errorMessage: "O grupo 'projeto' não foi criado corretamente ou os usuários não foram adicionados"`

// Conteúdo do template de Permissões de Arquivos Linux
const linuxPermsTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: linux-permissions-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: linux-file-permissions
    title: "Permissões de Arquivos no Linux"
    description: "Aprenda a visualizar e modificar permissões de arquivos e diretórios no Linux"
    duration: 15m
    image: "linuxtips/girus-devops:0.1"
    youtubeVideo: "https://www.youtube.com/watch?v=tT69ipXOzfc"
    tasks:
      - name: "Entendendo Permissões Básicas"
        description: "Aprenda a visualizar e interpretar permissões de arquivos e diretórios"
        steps:
          - "Crie um diretório para o exercício:"
          - "` + "`" + `mkdir ~/permissoes` + "`" + `"
          - "Entre no diretório:"
          - "` + "`" + `cd ~/permissoes` + "`" + `"
          - "Crie arquivos de teste:"
          - "` + "`" + `touch arquivo1.txt arquivo2.txt` + "`" + `"
          - "Visualize as permissões atuais:"
          - "` + "`" + `ls -la` + "`" + `"
          - "Observe o formato das permissões: [tipo][dono][grupo][outros]"
          - "Exemplo: -rw-r--r-- significa arquivo comum, dono pode ler e escrever, grupo e outros só podem ler"
          - "Crie um script simples:"
          - "` + "`" + `echo '#!/bin/bash' > script.sh` + "`" + `"
          - "` + "`" + `echo 'echo \"Hello, World!\"' >> script.sh` + "`" + `"
          - "Tente executar o script:"
          - "` + "`" + `./script.sh` + "`" + `"
          - "Você verá uma mensagem de permissão negada"
        tips:
          - type: "info"
            title: "Formato das permissões"
            content: "As permissões são representadas por 10 caracteres: o primeiro indica o tipo (- para arquivo, d para diretório), seguido por três grupos de rwx (read, write, execute) para dono, grupo e outros."
          - type: "tip"
            title: "Significado dos modos"
            content: "r (4) = permissão de leitura, w (2) = permissão de escrita, x (1) = permissão de execução. Os valores numéricos são somados para definir as permissões em octal."
        validation:
          - command: "test -f ~/permissoes/script.sh && echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "O arquivo script.sh não foi criado corretamente"
      
      - name: "Modificando Permissões"
        description: "Aprenda a modificar permissões de arquivos usando chmod"
        steps:
          - "Adicione permissão de execução ao script criado anteriormente:"
          - "` + "`" + `chmod +x script.sh` + "`" + `"
          - "Verifique as novas permissões:"
          - "` + "`" + `ls -la script.sh` + "`" + `"
          - "Agora execute o script:"
          - "` + "`" + `./script.sh` + "`" + `"
          - "Defina permissões usando notação octal:"
          - "` + "`" + `chmod 644 arquivo1.txt` + "`" + `"
          - "` + "`" + `chmod 640 arquivo2.txt` + "`" + `"
          - "Verifique as permissões após a mudança:"
          - "` + "`" + `ls -la arquivo*.txt` + "`" + `"
          - "Use chmod recursivo para modificar permissões em lote:"
          - "` + "`" + `mkdir -p subdir/subsubdir` + "`" + `"
          - "` + "`" + `touch subdir/file1 subdir/subsubdir/file2` + "`" + `"
          - "` + "`" + `chmod -R 750 subdir` + "`" + `"
          - "Verifique as permissões recursivas:"
          - "` + "`" + `ls -la subdir` + "`" + `"
          - "` + "`" + `ls -la subdir/subsubdir` + "`" + `"
        tips:
          - type: "info"
            title: "Notação simbólica vs octal"
            content: "Você pode usar notação simbólica (chmod u+x) ou octal (chmod 755). Em octal, cada dígito representa as permissões para dono, grupo e outros, respectivamente."
          - type: "warning"
            title: "Permissões e segurança"
            content: "Permissões muito abertas (ex: 777) representam riscos de segurança. Use sempre as permissões mínimas necessárias."
        validation:
          - command: "test -x ~/permissoes/script.sh && echo 'ok'"
            expectedOutput: "ok"
            errorMessage: "O script não tem permissão de execução"
          - command: "stat -c %a ~/permissoes/arquivo1.txt"
            expectedOutput: "644"
            errorMessage: "O arquivo1.txt não tem as permissões corretas (644)"
      
      - name: "Propriedade de Arquivos e umask"
        description: "Aprenda a alterar o proprietário dos arquivos e configurar o umask"
        steps:
          - "Veja o proprietário atual dos arquivos:"
          - "` + "`" + `ls -la` + "`" + `"
          - "Verifique o umask atual:"
          - "` + "`" + `umask` + "`" + `"
          - "Crie um novo arquivo com o umask padrão:"
          - "` + "`" + `touch arquivo_umask_padrao.txt` + "`" + `"
          - "` + "`" + `ls -la arquivo_umask_padrao.txt` + "`" + `"
          - "Mude temporariamente o umask para 027:"
          - "` + "`" + `umask 027` + "`" + `"
          - "Crie outro arquivo com o novo umask:"
          - "` + "`" + `touch arquivo_umask_027.txt` + "`" + `"
          - "` + "`" + `ls -la arquivo_umask_027.txt` + "`" + `"
          - "Volte ao umask padrão (geralmente 022):"
          - "` + "`" + `umask 022` + "`" + `"
          - "Se você tiver permissão de sudo, altere o proprietário de um arquivo:"
          - "` + "`" + `sudo chown root:root arquivo1.txt` + "`" + `"
          - "Verifique a mudança de proprietário:"
          - "` + "`" + `ls -la arquivo1.txt` + "`" + `"
        tips:
          - type: "info"
            title: "Umask explicado"
            content: "O umask define quais permissões são removidas por padrão ao criar novos arquivos e diretórios. Um umask de 022 remove escrita para grupo e outros, resultando em 644 para arquivos e 755 para diretórios."
          - type: "tip"
            title: "Permissão padrão"
            content: "A permissão padrão para diretórios é 777 e para arquivos é 666. O valor do umask é subtraído dessas permissões padrão."
        validation:
          - command: "umask"
            expectedOutput: "0022"
            errorMessage: "O umask não foi revertido para o valor padrão"
          - command: "stat -c %a ~/permissoes/arquivo_umask_027.txt 2>/dev/null || echo 'ausente'"
            expectedOutput: "640"
            errorMessage: "O arquivo com umask 027 não tem as permissões esperadas ou não foi criado"`

// Conteúdo do template de Gerenciamento de Containers Docker
const dockerContainersTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: docker-containers-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: docker-containers-management
    title: "Criando, Listando e Removendo Containers"
    description: "Aprenda a gerenciar o ciclo de vida de containers Docker: criação, listagem e remoção"
    duration: 20m
    image: "linuxtips/girus-devops:0.1"
    privileged: true
    tasks:
      - name: "Criando Containers Docker"
        description: "Aprenda a criar containers com diferentes configurações"
        steps:
          - "Verifique se o Docker está em execução:"
          - "` + "`" + `docker info` + "`" + `"
          - "Crie um container simples do Nginx:"
          - "` + "`" + `docker run --name webserver -d nginx` + "`" + `"
          - "Crie um container com mapeamento de portas:"
          - "` + "`" + `docker run --name webserver-ports -d -p 8080:80 nginx` + "`" + `"
          - "Crie um container com variáveis de ambiente:"
          - "` + "`" + `docker run --name env-test -e MINHA_VAR=test -d alpine sh -c 'while true; do sleep 10; done'` + "`" + `"
          - "Crie um container com limite de recursos:"
          - "` + "`" + `docker run --name limited-resources -d --memory=100m --cpus=0.2 nginx` + "`" + `"
          - "Crie um container efêmero (que se auto-remove):"
          - "` + "`" + `docker run --rm alpine echo \"Este container será removido automaticamente\"` + "`" + `"
        tips:
          - type: "info"
            title: "Opções comuns do docker run"
            content: "-d (detached), -p (mapeamento de portas), -e (variáveis de ambiente), -v (volumes), --name (nomear o container), --rm (auto-remover)"
          - type: "warning"
            title: "Recursos limitados"
            content: "Em ambientes de produção, sempre limite os recursos (memória/CPU) que um container pode usar para evitar que um único container consume todos os recursos do host."
        validation:
          - command: "docker ps -a --format '{{.Names}}' | grep -E 'webserver|env-test|limited-resources' | wc -l"
            expectedOutput: "4"
            errorMessage: "Alguns containers não foram criados corretamente"
      
      - name: "Listando e Inspecionando Containers"
        description: "Aprenda a listar, filtrar e inspecionar containers"
        steps:
          - "Liste todos os containers em execução:"
          - "` + "`" + `docker ps` + "`" + `"
          - "Liste todos os containers (incluindo os parados):"
          - "` + "`" + `docker ps -a` + "`" + `"
          - "Liste apenas os IDs dos containers:"
          - "` + "`" + `docker ps -q` + "`" + `"
          - "Filtre containers por nome:"
          - "` + "`" + `docker ps -a --filter \"name=webserver\"` + "`" + `"
          - "Filtre containers por status:"
          - "` + "`" + `docker ps -a --filter \"status=running\"` + "`" + `"
          - "Use formatação personalizada para a saída:"
          - "` + "`" + `docker ps --format \"{{.Names}} - {{.Status}} - {{.Image}}\"` + "`" + `"
          - "Inspecione detalhes de um container específico:"
          - "` + "`" + `docker inspect webserver` + "`" + `"
          - "Extraia uma informação específica com formato:"
          - "` + "`" + `docker inspect --format='{{.State.Status}}' webserver` + "`" + `"
          - "Verifique os logs de um container:"
          - "` + "`" + `docker logs webserver` + "`" + `"
          - "Verifique estatísticas de uso de recursos:"
          - "` + "`" + `docker stats --no-stream` + "`" + `"
        tips:
          - type: "tip"
            title: "Formatação no Docker"
            content: "O Docker suporta templates Go para formatar a saída de comandos como ps e inspect. Use --format para personalizar o que você quer ver."
          - type: "info"
            title: "Dica para filtros"
            content: "Você pode combinar múltiplos filtros com --filter, e eles atuam como condição 'AND' (todos devem ser verdadeiros)."
        validation:
          - command: "docker ps -a --format '{{.Names}}' | sort | tr '\n' ',' | grep -E 'webserver,|env-test,|limited-resources,' | wc -l"
            expectedOutput: "1"
            errorMessage: "Não foi possível listar todos os containers corretamente"
      
      - name: "Gerenciando o Ciclo de Vida dos Containers"
        description: "Aprenda a iniciar, parar, reiniciar e remover containers"
        steps:
          - "Pare um container em execução:"
          - "` + "`" + `docker stop webserver` + "`" + `"
          - "Verifique se o container foi parado:"
          - "` + "`" + `docker ps -a --filter name=webserver` + "`" + `"
          - "Inicie novamente o container parado:"
          - "` + "`" + `docker start webserver` + "`" + `"
          - "Reinicie um container:"
          - "` + "`" + `docker restart webserver-ports` + "`" + `"
          - "Pause um container (congele seu estado):"
          - "` + "`" + `docker pause env-test` + "`" + `"
          - "Despause um container congelado:"
          - "` + "`" + `docker unpause env-test` + "`" + `"
          - "Remova um container (deve estar parado):"
          - "` + "`" + `docker stop limited-resources` + "`" + `"
          - "` + "`" + `docker rm limited-resources` + "`" + `"
          - "Remova um container forçadamente (mesmo em execução):"
          - "` + "`" + `docker rm -f webserver-ports` + "`" + `"
          - "Remova todos os containers parados:"
          - "` + "`" + `docker container prune -f` + "`" + `"
          - "Crie e remova um container em um único comando:"
          - "` + "`" + `docker run --rm alpine echo \"Criado e removido automaticamente\"` + "`" + `"
        tips:
          - type: "warning"
            title: "Remoção forçada"
            content: "O parâmetro -f (force) para docker rm pode causar corrupção de dados se o container estiver gravando informações. Use com cuidado em ambientes de produção."
          - type: "info"
            title: "Tempo de espera"
            content: "Por padrão, docker stop espera 10 segundos antes de enviar SIGKILL. Você pode ajustar isso com --time=segundos."
        validation:
          - command: "docker ps -a --format '{{.Names}}' | grep -E 'limited-resources|webserver-ports' | wc -l"
            expectedOutput: "0"
            errorMessage: "Os containers não foram removidos corretamente"`

// Conteúdo do template de Deployment no Kubernetes
const k8sDeploymentTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: k8s-deployment-lab
  namespace: girus
  labels:
    app: girus-lab-template
data:
  lab.yaml: |
    name: k8s-nginx-deployment
    title: "Deployment Nginx no Kubernetes"
    description: "Aprenda a criar um deployment do Nginx e verificar os detalhes dos pods no Kubernetes"
    duration: 15m
    timerEnabled: true
    maxDuration: 15m
    image: "linuxtips/girus-kind-single-node:0.1"
    youtubeVideo: "https://www.youtube.com/watch?v=sU235yW6QJM"
    tasks:
      - name: "Criando um Deployment do Nginx"
        description: "Aprenda a criar um deployment do Nginx usando kubectl"
        steps:
          - "Verifique se o cluster Kubernetes está em execução:"
          - "` + "`" + `kubectl get nodes` + "`" + `"
          - "Crie um namespace para o exercício:"
          - "` + "`" + `kubectl create namespace nginx-example` + "`" + `"
          - "Crie um deployment do Nginx de forma imperativa:"
          - "` + "`" + `kubectl create deployment nginx-deployment --image=nginx:latest --replicas=2 -n nginx-example` + "`" + `"
          - "Verifique se o deployment foi criado:"
          - "` + "`" + `kubectl get deployments -n nginx-example` + "`" + `"
          - "Verifique o status da implantação:"
          - "` + "`" + `kubectl rollout status deployment/nginx-deployment -n nginx-example` + "`" + `"
          - "Agora, vamos criar outro deployment usando um arquivo YAML:"
          - "Crie um arquivo deployment.yaml:"
          - |
            ` + "```yaml" + `
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: nginx-declarative
              namespace: nginx-example
              labels:
                app: nginx-declarative
            spec:
              replicas: 3
              selector:
                matchLabels:
                  app: nginx-declarative
              template:
                metadata:
                  labels:
                    app: nginx-declarative
                spec:
                  containers:
                  - name: nginx
                    image: nginx:stable
                    ports:
                    - containerPort: 80
                    resources:
                      limits:
                        cpu: "0.3"
                        memory: "128Mi"
                      requests:
                        cpu: "0.1"
                        memory: "64Mi"
            ` + "```" + `
          - "Aplique o arquivo YAML:"
          - "` + "`" + `kubectl apply -f deployment.yaml` + "`" + `"
          - "Verifique se ambos os deployments estão em execução:"
          - "` + "`" + `kubectl get deployments -n nginx-example` + "`" + `"
        tips:
          - type: "info"
            title: "Abordagens de criação"
            content: "No Kubernetes, você pode criar recursos de forma imperativa (com comandos diretos) ou declarativa (usando arquivos YAML). A abordagem declarativa é preferida para ambientes de produção, pois facilita o controle de versão."
          - type: "warning"
            title: "Seletor de labels"
            content: "É essencial que o seletor no deployment corresponda exatamente às labels dos pods, caso contrário, o deployment não conseguirá gerenciar os pods."
        validation:
          - command: "kubectl get deployments -n nginx-example -o name | wc -l"
            expectedOutput: "2"
            errorMessage: "Os deployments não foram criados corretamente"
      
      - name: "Verificando Detalhes dos Pods"
        description: "Aprenda a verificar e depurar os pods criados pelo deployment"
        steps:
          - "Liste todos os pods gerados pelos deployments:"
          - "` + "`" + `kubectl get pods -n nginx-example` + "`" + `"
          - "Verifique os detalhes de um pod específico (substitua [pod-name] pelo nome real do pod):"
          - "` + "`" + `POD_NAME=$(kubectl get pods -n nginx-example -l app=nginx-declarative -o jsonpath='{.items[0].metadata.name}')` + "`" + `"
          - "` + "`" + `kubectl describe pod $POD_NAME -n nginx-example` + "`" + `"
          - "Verifique os logs de um pod:"
          - "` + "`" + `kubectl logs $POD_NAME -n nginx-example` + "`" + `"
          - "Verifique os eventos no namespace:"
          - "` + "`" + `kubectl get events -n nginx-example` + "`" + `"
          - "Acesse a aplicação Nginx em um pod (isso abrirá um shell):"
          - "` + "`" + `kubectl exec -it $POD_NAME -n nginx-example -- /bin/bash` + "`" + `"
          - "Dentro do container, verifique se o Nginx está respondendo:"
          - "` + "`" + `curl localhost:80` + "`" + `"
          - "Saia do shell com o comando 'exit'"
        tips:
          - type: "tip"
            title: "Filtrar pods por label"
            content: "Use -l (--selector) para filtrar pods por labels, o que é útil quando há muitos pods no namespace."
          - type: "info"
            title: "Acessando logs"
            content: "Para containers com múltiplos aplicativos (sidecar), especifique o container com -c [nome-container] no comando kubectl logs."
        validation:
          - command: "kubectl get pods -n nginx-example -l app=nginx-declarative --no-headers | wc -l"
            expectedOutput: "3"
            errorMessage: "O número de pods para o deployment declarativo não está correto"
      
      - name: "Expondo e Escalando o Deployment"
        description: "Aprenda a expor o deployment como um serviço e escalar o número de réplicas"
        steps:
          - "Crie um serviço para expor o deployment:"
          - "` + "`" + `kubectl expose deployment nginx-declarative --port=80 --type=ClusterIP -n nginx-example` + "`" + `"
          - "Verifique se o serviço foi criado:"
          - "` + "`" + `kubectl get services -n nginx-example` + "`" + `"
          - "Escale o deployment para ter 5 réplicas:"
          - "` + "`" + `kubectl scale deployment/nginx-declarative --replicas=5 -n nginx-example` + "`" + `"
          - "Verifique se o número de pods foi atualizado:"
          - "` + "`" + `kubectl get pods -n nginx-example -l app=nginx-declarative` + "`" + `"
          - "Observe o estado da escala em tempo real:"
          - "` + "`" + `kubectl rollout status deployment/nginx-declarative -n nginx-example` + "`" + `"
          - "Atualize a imagem do deployment para uma versão específica:"
          - "` + "`" + `kubectl set image deployment/nginx-declarative nginx=nginx:1.19 -n nginx-example` + "`" + `"
          - "Verifique o histórico de rollout:"
          - "` + "`" + `kubectl rollout history deployment/nginx-declarative -n nginx-example` + "`" + `"
          - "Veja os detalhes de uma revisão específica:"
          - "` + "`" + `kubectl rollout history deployment/nginx-declarative --revision=2 -n nginx-example` + "`" + `"
          - "Faça rollback para a revisão anterior, se necessário:"
          - "` + "`" + `kubectl rollout undo deployment/nginx-declarative -n nginx-example` + "`" + `"
        tips:
          - type: "info"
            title: "Tipos de serviço"
            content: "O Kubernetes oferece diferentes tipos de serviços: ClusterIP (interno), NodePort (expõe em cada nó), LoadBalancer (usa balanceador externo) e ExternalName (mapeia para um nome externo)."
          - type: "warning"
            title: "Escala e recursos"
            content: "Ao escalar um deployment, certifique-se de que o cluster tem recursos suficientes (CPU/memória) para suportar o número de réplicas solicitado."
        validation:
          - command: "kubectl get pods -n nginx-example -l app=nginx-declarative --no-headers | wc -l"
            expectedOutput: "5"
            errorMessage: "O deployment não foi escalado corretamente para 5 réplicas"
          - command: "kubectl get svc -n nginx-example -o name | grep nginx-declarative | wc -l"
            expectedOutput: "1"
            errorMessage: "O serviço para o deployment declarativo não foi criado corretamente"`

// defaultDeployment contém o YAML de deployment padrão do Girus
const defaultDeployment = `apiVersion: v1
kind: Namespace
metadata:
  name: girus
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: girus-sa
  namespace: girus
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: girus-role
  namespace: girus
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "pods/exec"]
    verbs: ["get", "list", "create", "delete", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: girus-cluster-role
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods", "pods/log", "pods/exec"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["services", "configmaps", "secrets", "serviceaccounts"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["roles", "rolebindings"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: girus-cluster-rolebinding
subjects:
  - kind: ServiceAccount
    name: girus-sa
    namespace: girus
roleRef:
  kind: ClusterRole
  name: girus-cluster-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: girus-rolebinding
  namespace: girus
subjects:
  - kind: ServiceAccount
    name: girus-sa
    namespace: girus
roleRef:
  kind: Role
  name: girus-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: girus-config
  namespace: girus
data:
  config.yaml: |
    lab:
      defaultImage: "ubuntu:latest"
      podNamePrefix: "lab"
      containerName: "linux-lab"
      command:
        - "sleep"
        - "infinity"
      resources:
        cpuRequest: "100m"
        cpuLimit: "500m"
        memoryRequest: "64Mi"
        memoryLimit: "256Mi"
      envVars:
        TERM: "xterm-256color"
        SHELL: "/bin/bash"
        privileged: false
    # Outras configurações podem ser adicionadas aqui
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: girus-backend
  namespace: girus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: girus-backend
  template:
    metadata:
      labels:
        app: girus-backend
    spec:
      serviceAccountName: girus-sa
      containers:
        - name: backend
          image: linuxtips/girus-backend:0.1
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 8080
          env:
            - name: PORT
              value: "8080"
            - name: GIN_MODE
              value: "release"
            - name: LAB_DEFAULT_IMAGE
              valueFrom:
                configMapKeyRef:
                  name: girus-config
                  key: lab.defaultImage
                  optional: true
          volumeMounts:
            - name: config-volume
              mountPath: /app/config
      volumes:
        - name: config-volume
          configMap:
            name: girus-config
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: girus-frontend
  namespace: girus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: girus-frontend
  template:
    metadata:
      labels:
        app: girus-frontend
    spec:
      containers:
        - name: frontend
          image: linuxtips/girus-frontend:0.1
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 80
          volumeMounts:
            - name: nginx-config
              mountPath: /etc/nginx/conf.d
      volumes:
        - name: nginx-config
          configMap:
            name: nginx-config
---
apiVersion: v1
kind: Service
metadata:
  name: girus-backend
  namespace: girus
spec:
  selector:
    app: girus-backend
  ports:
    - port: 8080
      targetPort: 8080
  type: ClusterIP
---
apiVersion: v1
kind: Service
metadata:
  name: girus-frontend
  namespace: girus
spec:
  selector:
    app: girus-frontend
  ports:
    - port: 80
      targetPort: 80
  type: ClusterIP
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-config
  namespace: girus
data:
  default.conf: |
    server {
        listen 80;
        server_name localhost;
        root /usr/share/nginx/html;
        index index.html;
        
        # Compressão
        gzip on;
        gzip_vary on;
        gzip_min_length 1000;
        gzip_proxied any;
        gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
        gzip_comp_level 6;
        
        # Cache para recursos estáticos
        location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
            expires 30d;
            add_header Cache-Control "public, no-transform";
        }
        
        # Redirecionar todas as requisições API para o backend
        location /api/ {
            proxy_pass http://girus-backend:8080/api/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_buffering off;
            proxy_request_buffering off;
        }
        
        # Configuração para WebSockets
        location /ws/ {
            proxy_pass http://girus-backend:8080/ws/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_read_timeout 86400;
        }
        
        # Configuração para React Router
        location / {
            try_files $uri $uri/ /index.html;
        }
    }
---
# Criar o namespace para o usuário de teste
apiVersion: v1
kind: Namespace
metadata:
  name: lab-test-user
---
# Criar a service account caso ela não exista
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
  namespace: lab-test-user
---
# Conceder permissões de administrador para o usuário de teste
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: lab-test-user-admin-access
  annotations:
    description: "Permissões de administrador para ambiente educacional"
subjects:
  - kind: ServiceAccount
    name: default
    namespace: lab-test-user
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
`

// waitForPodsReady espera até que os pods do Girus (backend e frontend) estejam prontos
func waitForPodsReady(namespace string, timeout time.Duration) error {
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
func setupPortForward(namespace string) error {
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
nohup kubectl port-forward -n NAMESPACE svc/girus-frontend 8000:80 --address 0.0.0.0> /dev/null 2>&1 &
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
			checkCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://0.0.0.0:8000")
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
		cmd := exec.Command("kubectl", "port-forward", "-n", namespace, "svc/girus-frontend", "8000:80", "--address", "0.0.0.0/0")

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
			checkCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://0.0.0.0:8000")
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
		checkCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://0.0.0.0:8000")
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
			ioutil.WriteFile(filepath.Join(pidDir, "frontend.pid"),
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

// portInUse verifica se uma porta está em uso
func portInUse(port int) bool {
	checkCmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	return checkCmd.Run() == nil
}

// openBrowser abre o navegador com a URL especificada
func openBrowser(url string) error {
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

var createCmd = &cobra.Command{
	Use:   "create [subcommand]",
	Short: "Comandos para criar recursos",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Cria o cluster Girus",
	Long: `Cria um cluster Kind com o nome "girus" e implanta todos os componentes necessários.
Por padrão, o deployment embutido no binário é utilizado.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Verificar se o Docker está instalado e funcionando
		fmt.Println("🔄 Verificando pré-requisitos...")
		dockerCmd := exec.Command("docker", "--version")
		if err := dockerCmd.Run(); err != nil {
			fmt.Println("❌ Docker não encontrado ou não está em execução")
			fmt.Println("\nO Docker é necessário para criar um cluster Kind. Instruções de instalação:")

			// Detectar o sistema operacional para instruções específicas
			if runtime.GOOS == "darwin" {
				// macOS
				fmt.Println("\n📦 Para macOS, recomendamos usar Colima (alternativa leve ao Docker Desktop):")
				fmt.Println("1. Instale o Homebrew caso não tenha:")
				fmt.Println("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
				fmt.Println("2. Instale o Colima e o Docker CLI:")
				fmt.Println("   brew install colima docker")
				fmt.Println("3. Inicie o Colima:")
				fmt.Println("   colima start")
				fmt.Println("\nAlternativamente, você pode instalar o Docker Desktop para macOS de:")
				fmt.Println("https://www.docker.com/products/docker-desktop")
			} else if runtime.GOOS == "linux" {
				// Linux
				fmt.Println("\n📦 Para Linux, use o script de instalação oficial:")
				fmt.Println("   curl -fsSL https://get.docker.com | bash")
				fmt.Println("\nApós a instalação, adicione seu usuário ao grupo docker para evitar usar sudo:")
				fmt.Println("   sudo usermod -aG docker $USER")
				fmt.Println("   newgrp docker")
				fmt.Println("\nE inicie o serviço:")
				fmt.Println("   sudo systemctl enable docker")
				fmt.Println("   sudo systemctl start docker")
			} else {
				// Windows ou outros sistemas
				fmt.Println("\n📦 Visite https://www.docker.com/products/docker-desktop para instruções de instalação para seu sistema operacional")
			}

			fmt.Println("\nApós instalar o Docker, execute novamente este comando.")
			os.Exit(1)
		}

		// Verificar se o serviço Docker está rodando
		dockerInfoCmd := exec.Command("docker", "info")
		if err := dockerInfoCmd.Run(); err != nil {
			fmt.Println("❌ O serviço Docker não está em execução")

			if runtime.GOOS == "darwin" {
				fmt.Println("\nPara macOS com Colima:")
				fmt.Println("   colima start")
				fmt.Println("\nPara Docker Desktop:")
				fmt.Println("   Inicie o aplicativo Docker Desktop")
			} else if runtime.GOOS == "linux" {
				fmt.Println("\nInicie o serviço Docker:")
				fmt.Println("   sudo systemctl start docker")
			} else {
				fmt.Println("\nInicie o Docker Desktop ou o serviço Docker apropriado para seu sistema.")
			}

			fmt.Println("\nApós iniciar o Docker, execute novamente este comando.")
			os.Exit(1)
		}

		fmt.Println("✅ Docker detectado e funcionando")

		// Verificar silenciosamente se o cluster já existe
		checkCmd := exec.Command("kind", "get", "clusters")
		output, err := checkCmd.Output()

		// Ignorar erros na checagem, apenas assumimos que não há clusters
		if err == nil {
			clusters := strings.Split(strings.TrimSpace(string(output)), "\n")

			// Verificar se o cluster "girus" já existe
			clusterExists := false
			for _, cluster := range clusters {
				if cluster == clusterName {
					clusterExists = true
					break
				}
			}

			if clusterExists {
				fmt.Printf("⚠️  Cluster Girus já existe.\n")
				fmt.Print("Deseja substituí-lo? [s/N]: ")

				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.ToLower(strings.TrimSpace(response))

				if response != "s" && response != "sim" && response != "y" && response != "yes" {
					fmt.Println("Operação cancelada.")
					return
				}

				// Excluir o cluster existente
				fmt.Printf("Excluindo cluster Girus existente...\n")

				deleteCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
				if verboseMode {
					deleteCmd.Stdout = os.Stdout
					deleteCmd.Stderr = os.Stderr
					if err := deleteCmd.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "❌ Erro ao excluir o cluster existente: %v\n", err)
						fmt.Println("   Por favor, exclua manualmente com 'kind delete cluster --name girus' e tente novamente.")
						os.Exit(1)
					}
				} else {
					// Usar barra de progresso
					bar := progressbar.NewOptions(100,
						progressbar.OptionSetDescription("Excluindo cluster existente..."),
						progressbar.OptionSetWidth(80),
						progressbar.OptionShowBytes(false),
						progressbar.OptionSetPredictTime(false),
						progressbar.OptionThrottle(65*time.Millisecond),
						progressbar.OptionSetRenderBlankState(true),
						progressbar.OptionSpinnerType(14),
						progressbar.OptionFullWidth(),
					)

					var stderr bytes.Buffer
					deleteCmd.Stderr = &stderr

					// Iniciar o comando
					err := deleteCmd.Start()
					if err != nil {
						fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar exclusão: %v\n", err)
						os.Exit(1)
					}

					// Atualizar a barra de progresso
					done := make(chan struct{})
					go func() {
						for {
							select {
							case <-done:
								return
							default:
								bar.Add(1)
								time.Sleep(100 * time.Millisecond)
							}
						}
					}()

					// Aguardar o final do comando
					err = deleteCmd.Wait()
					close(done)
					bar.Finish()

					if err != nil {
						fmt.Fprintf(os.Stderr, "❌ Erro ao excluir o cluster existente: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderr.String())
						fmt.Println("   Por favor, exclua manualmente com 'kind delete cluster --name girus' e tente novamente.")
						os.Exit(1)
					}
				}

				fmt.Println("✅ Cluster existente excluído com sucesso.")
			}
		}

		// Criar o cluster Kind
		fmt.Println("🔄 Criando cluster Girus...")

		if verboseMode {
			// Executar normalmente mostrando o output
			createClusterCmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
			createClusterCmd.Stdout = os.Stdout
			createClusterCmd.Stderr = os.Stderr

			if err := createClusterCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar o cluster Girus: %v\n", err)
				fmt.Println("   Possíveis causas:")
				fmt.Println("   • Docker não está em execução")
				fmt.Println("   • Permissões insuficientes")
				fmt.Println("   • Conflito com cluster existente")
				os.Exit(1)
			}
		} else {
			// Usando barra de progresso (padrão)
			bar := progressbar.NewOptions(100,
				progressbar.OptionSetDescription("Criando cluster..."),
				progressbar.OptionSetWidth(80),
				progressbar.OptionShowBytes(false),
				progressbar.OptionSetPredictTime(false),
				progressbar.OptionThrottle(65*time.Millisecond),
				progressbar.OptionSetRenderBlankState(true),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionFullWidth(),
			)

			// Executar comando sem mostrar saída
			createClusterCmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
			var stderr bytes.Buffer
			createClusterCmd.Stderr = &stderr

			// Iniciar o comando
			err := createClusterCmd.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar o comando: %v\n", err)
				os.Exit(1)
			}

			// Atualizar a barra de progresso enquanto o comando está em execução
			done := make(chan struct{})
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						bar.Add(1)
						time.Sleep(200 * time.Millisecond)
					}
				}
			}()

			// Aguardar o final do comando
			err = createClusterCmd.Wait()
			close(done)
			bar.Finish()

			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar o cluster Girus: %v\n", err)

				// Traduzir mensagens de erro comuns
				errMsg := stderr.String()

				if strings.Contains(errMsg, "node(s) already exist for a cluster with the name") {
					fmt.Println("   Erro: Já existe um cluster com o nome 'girus' no sistema.")
					fmt.Println("   Por favor, exclua-o primeiro com 'kind delete cluster --name girus'")
				} else if strings.Contains(errMsg, "permission denied") {
					fmt.Println("   Erro: Permissão negada. Verifique as permissões do Docker.")
				} else if strings.Contains(errMsg, "Cannot connect to the Docker daemon") {
					fmt.Println("   Erro: Não foi possível conectar ao serviço Docker.")
					fmt.Println("   Verifique se o Docker está em execução com 'systemctl status docker'")
				} else {
					fmt.Println("   Detalhes técnicos:", errMsg)
				}

				os.Exit(1)
			}
		}

		fmt.Println("✅ Cluster Girus criado com sucesso!")

		// Aplicar o manifesto de deployment do Girus
		fmt.Println("\n📦 Implantando o Girus no cluster...")

		// Verificar se existe o arquivo girus-kind-deploy.yaml
		deployYamlPath := "girus-kind-deploy.yaml"
		foundDeployFile := false

		// Verificar em diferentes locais possíveis
		possiblePaths := []string{
			deployYamlPath,                      // No diretório atual
			filepath.Join("..", deployYamlPath), // Um nível acima
			filepath.Join(os.Getenv("HOME"), "REPOS", "strigus", deployYamlPath), // Caminho comum
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				deployFile = path
				foundDeployFile = true
				break
			}
		}

		if foundDeployFile {
			fmt.Printf("🔍 Usando arquivo de deployment: %s\n", deployFile)

			// Aplicar arquivo de deployment completo (já contém o template do lab)
			if verboseMode {
				// Executar normalmente mostrando o output
				applyCmd := exec.Command("kubectl", "apply", "-f", deployFile)
				applyCmd.Stdout = os.Stdout
				applyCmd.Stderr = os.Stderr

				if err := applyCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Usar barra de progresso
				bar := progressbar.NewOptions(100,
					progressbar.OptionSetDescription("Implantando Girus..."),
					progressbar.OptionSetWidth(80),
					progressbar.OptionShowBytes(false),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
				)

				// Executar comando sem mostrar saída
				applyCmd := exec.Command("kubectl", "apply", "-f", deployFile)
				var stderr bytes.Buffer
				applyCmd.Stderr = &stderr

				// Iniciar o comando
				err := applyCmd.Start()
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar o comando: %v\n", err)
					os.Exit(1)
				}

				// Atualizar a barra de progresso enquanto o comando está em execução
				done := make(chan struct{})
				go func() {
					for {
						select {
						case <-done:
							return
						default:
							bar.Add(1)
							time.Sleep(100 * time.Millisecond)
						}
					}
				}()

				// Aguardar o final do comando
				err = applyCmd.Wait()
				close(done)
				bar.Finish()

				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					fmt.Println("   Detalhes técnicos:", stderr.String())
					os.Exit(1)
				}
			}

			fmt.Println("✅ Infraestrutura e template de laboratório aplicados com sucesso!")
		} else {
			// Usar o deployment embutido como fallback
			// fmt.Println("⚠️  Arquivo girus-kind-deploy.yaml não encontrado, usando deployment embutido.")

			// Criar um arquivo temporário para o deployment principal
			tempFile, err := os.CreateTemp("", "girus-deploy-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário: %v\n", err)
				os.Exit(1)
			}
			defer os.Remove(tempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo no arquivo temporário
			if _, err := tempFile.WriteString(defaultDeployment); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever no arquivo temporário: %v\n", err)
				os.Exit(1)
			}
			tempFile.Close()

			// Aplicar o deployment principal
			if verboseMode {
				// Executar normalmente mostrando o output
				applyCmd := exec.Command("kubectl", "apply", "-f", tempFile.Name())
				applyCmd.Stdout = os.Stdout
				applyCmd.Stderr = os.Stderr

				if err := applyCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Usar barra de progresso para o deploy (padrão)
				bar := progressbar.NewOptions(100,
					progressbar.OptionSetDescription("Implantando infraestrutura..."),
					progressbar.OptionSetWidth(80),
					progressbar.OptionShowBytes(false),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
				)

				// Executar comando sem mostrar saída
				applyCmd := exec.Command("kubectl", "apply", "-f", tempFile.Name())
				var stderr bytes.Buffer
				applyCmd.Stderr = &stderr

				// Iniciar o comando
				err := applyCmd.Start()
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar o comando: %v\n", err)
					os.Exit(1)
				}

				// Atualizar a barra de progresso enquanto o comando está em execução
				done := make(chan struct{})
				go func() {
					for {
						select {
						case <-done:
							return
						default:
							bar.Add(1)
							time.Sleep(100 * time.Millisecond)
						}
					}
				}()

				// Aguardar o final do comando
				err = applyCmd.Wait()
				close(done)
				bar.Finish()

				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o manifesto do Girus: %v\n", err)
					fmt.Println("   Detalhes técnicos:", stderr.String())
					os.Exit(1)
				}
			}

			fmt.Println("✅ Infraestrutura básica aplicada com sucesso!")

			// Agora vamos aplicar o template de laboratório que está embutido no binário
			fmt.Println("\n🔬 Aplicando templates de laboratório...")

			// Criar um arquivo temporário para o template do laboratório Linux
			labTempFile, err := os.CreateTemp("", "basic-linux-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template Linux: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
				return
			}
			defer os.Remove(labTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template Linux no arquivo temporário
			if _, err := labTempFile.WriteString(basicLinuxTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template Linux no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
				return
			}
			labTempFile.Close()

			// Criar um arquivo temporário para o template do laboratório Kubernetes
			k8sTempFile, err := os.CreateTemp("", "kubernetes-basics-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template Kubernetes: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Kubernetes.")
				return
			}
			defer os.Remove(k8sTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template Kubernetes no arquivo temporário
			if _, err := k8sTempFile.WriteString(basicKubernetesTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template Kubernetes no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Kubernetes.")
				return
			}
			k8sTempFile.Close()

			// Criar um arquivo temporário para o template do laboratório Docker
			dockerTempFile, err := os.CreateTemp("", "docker-basics-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template Docker: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Docker.")
				return
			}
			defer os.Remove(dockerTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template Docker no arquivo temporário
			if _, err := dockerTempFile.WriteString(basicDockerTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template Docker no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Docker.")
				return
			}
			dockerTempFile.Close()

			// Criar um arquivo temporário para o template de Administração de Usuários Linux
			linuxUsersTempFile, err := os.CreateTemp("", "linux-users-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Usuários Linux: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Usuários Linux.")
				return
			}
			defer os.Remove(linuxUsersTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template de Usuários Linux no arquivo temporário
			if _, err := linuxUsersTempFile.WriteString(linuxUsersTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Usuários Linux no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Usuários Linux.")
				return
			}
			linuxUsersTempFile.Close()

			// Criar um arquivo temporário para o template de Permissões de Arquivos Linux
			linuxPermsTempFile, err := os.CreateTemp("", "linux-perms-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Permissões Linux: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Permissões Linux.")
				return
			}
			defer os.Remove(linuxPermsTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template de Permissões Linux no arquivo temporário
			if _, err := linuxPermsTempFile.WriteString(linuxPermsTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Permissões Linux no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Permissões Linux.")
				return
			}
			linuxPermsTempFile.Close()

			// Criar um arquivo temporário para o template de Gerenciamento de Containers Docker
			dockerContainersTempFile, err := os.CreateTemp("", "docker-containers-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Containers Docker: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Containers Docker.")
				return
			}
			defer os.Remove(dockerContainersTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template de Containers Docker no arquivo temporário
			if _, err := dockerContainersTempFile.WriteString(dockerContainersTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Containers Docker no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Containers Docker.")
				return
			}
			dockerContainersTempFile.Close()

			// Criar um arquivo temporário para o template de Deployment Kubernetes
			k8sDeploymentTempFile, err := os.CreateTemp("", "k8s-deployment-*.yaml")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao criar arquivo temporário para o template de Deployment Kubernetes: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Deployment Kubernetes.")
				return
			}
			defer os.Remove(k8sDeploymentTempFile.Name()) // Limpar o arquivo temporário ao finalizar

			// Escrever o conteúdo do template de Deployment Kubernetes no arquivo temporário
			if _, err := k8sDeploymentTempFile.WriteString(k8sDeploymentTemplate); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Erro ao escrever template de Deployment Kubernetes no arquivo temporário: %v\n", err)
				fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de Deployment Kubernetes.")
				return
			}
			k8sDeploymentTempFile.Close()

			// Aplicar o template de laboratório Linux
			if verboseMode {
				// Executar normalmente mostrando o output
				fmt.Println("   Aplicando template de laboratório Linux...")
				applyLabCmd := exec.Command("kubectl", "apply", "-f", labTempFile.Name())
				applyLabCmd.Stdout = os.Stdout
				applyLabCmd.Stderr = os.Stderr

				if err := applyLabCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Linux: %v\n", err)
					fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Linux.")
				} else {
					fmt.Println("   ✅ Template de laboratório Linux Básico aplicado com sucesso!")
				}

				// Aplicar o template de laboratório Kubernetes
				fmt.Println("   Aplicando template de laboratório Kubernetes...")
				applyK8sCmd := exec.Command("kubectl", "apply", "-f", k8sTempFile.Name())
				applyK8sCmd.Stdout = os.Stdout
				applyK8sCmd.Stderr = os.Stderr

				if err := applyK8sCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Kubernetes: %v\n", err)
					fmt.Println("   A infraestrutura básica e o template Linux foram aplicados, mas sem o template de laboratório Kubernetes.")
				} else {
					fmt.Println("   ✅ Template de laboratório Fundamentos de Kubernetes aplicado com sucesso!")
				}

				// Aplicar o template de laboratório Docker
				fmt.Println("   Aplicando template de laboratório Docker...")
				applyDockerCmd := exec.Command("kubectl", "apply", "-f", dockerTempFile.Name())
				applyDockerCmd.Stdout = os.Stdout
				applyDockerCmd.Stderr = os.Stderr

				if err := applyDockerCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Docker: %v\n", err)
					fmt.Println("   A infraestrutura básica e os outros templates foram aplicados, mas sem o template de laboratório Docker.")
				} else {
					fmt.Println("   ✅ Template de laboratório Fundamentos de Docker aplicado com sucesso!")
				}

				// Aplicar o template de Usuários Linux
				fmt.Println("   Aplicando template de Administração de Usuários Linux...")
				applyLinuxUsersCmd := exec.Command("kubectl", "apply", "-f", linuxUsersTempFile.Name())
				applyLinuxUsersCmd.Stdout = os.Stdout
				applyLinuxUsersCmd.Stderr = os.Stderr

				if err := applyLinuxUsersCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Usuários Linux: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Administração de Usuários Linux aplicado com sucesso!")
				}

				// Aplicar o template de Permissões Linux
				fmt.Println("   Aplicando template de Permissões de Arquivos Linux...")
				applyLinuxPermsCmd := exec.Command("kubectl", "apply", "-f", linuxPermsTempFile.Name())
				applyLinuxPermsCmd.Stdout = os.Stdout
				applyLinuxPermsCmd.Stderr = os.Stderr

				if err := applyLinuxPermsCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Permissões Linux: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Permissões de Arquivos Linux aplicado com sucesso!")
				}

				// Aplicar o template de Containers Docker
				fmt.Println("   Aplicando template de Gerenciamento de Containers Docker...")
				applyDockerContainersCmd := exec.Command("kubectl", "apply", "-f", dockerContainersTempFile.Name())
				applyDockerContainersCmd.Stdout = os.Stdout
				applyDockerContainersCmd.Stderr = os.Stderr

				if err := applyDockerContainersCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Containers Docker: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Gerenciamento de Containers Docker aplicado com sucesso!")
				}

				// Aplicar o template de Deployment Kubernetes
				fmt.Println("   Aplicando template de Deployment Nginx Kubernetes...")
				applyK8sDeploymentCmd := exec.Command("kubectl", "apply", "-f", k8sDeploymentTempFile.Name())
				applyK8sDeploymentCmd.Stdout = os.Stdout
				applyK8sDeploymentCmd.Stderr = os.Stderr

				if err := applyK8sDeploymentCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Deployment Kubernetes: %v\n", err)
				} else {
					fmt.Println("   ✅ Template de Deployment Nginx Kubernetes aplicado com sucesso!")
				}
			} else {
				// Usar barra de progresso para os templates
				bar := progressbar.NewOptions(100,
					progressbar.OptionSetDescription("Aplicando templates de laboratório..."),
					progressbar.OptionSetWidth(80),
					progressbar.OptionShowBytes(false),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
				)

				// Executar comando para aplicar o template Linux
				applyLabCmd := exec.Command("kubectl", "apply", "-f", labTempFile.Name())
				var stderrLinux bytes.Buffer
				applyLabCmd.Stderr = &stderrLinux

				// Iniciar o comando
				err := applyLabCmd.Start()
				if err != nil {
					bar.Finish()
					fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar aplicação do template Linux: %v\n", err)
					fmt.Println("   A infraestrutura básica foi aplicada, mas sem os templates de laboratório.")
				} else {
					// Atualizar a barra de progresso enquanto o comando está em execução
					done := make(chan struct{})
					go func() {
						for {
							select {
							case <-done:
								return
							default:
								bar.Add(1)
								time.Sleep(50 * time.Millisecond)
							}
						}
					}()

					// Aguardar o final do comando
					err = applyLabCmd.Wait()
					close(done)

					linuxSuccess := err == nil

					// Aplicar o template de Kubernetes
					applyK8sCmd := exec.Command("kubectl", "apply", "-f", k8sTempFile.Name())
					var stderrK8s bytes.Buffer
					applyK8sCmd.Stderr = &stderrK8s

					err = applyK8sCmd.Run()
					k8sSuccess := err == nil

					// Aplicar o template de Docker
					applyDockerCmd := exec.Command("kubectl", "apply", "-f", dockerTempFile.Name())
					var stderrDocker bytes.Buffer
					applyDockerCmd.Stderr = &stderrDocker

					err = applyDockerCmd.Run()
					dockerSuccess := err == nil

					// Aplicar os novos templates
					applyLinuxUsersCmd := exec.Command("kubectl", "apply", "-f", linuxUsersTempFile.Name())
					var stderrLinuxUsers bytes.Buffer
					applyLinuxUsersCmd.Stderr = &stderrLinuxUsers

					err = applyLinuxUsersCmd.Run()
					linuxUsersSuccess := err == nil

					applyLinuxPermsCmd := exec.Command("kubectl", "apply", "-f", linuxPermsTempFile.Name())
					var stderrLinuxPerms bytes.Buffer
					applyLinuxPermsCmd.Stderr = &stderrLinuxPerms

					err = applyLinuxPermsCmd.Run()
					linuxPermsSuccess := err == nil

					applyDockerContainersCmd := exec.Command("kubectl", "apply", "-f", dockerContainersTempFile.Name())
					var stderrDockerContainers bytes.Buffer
					applyDockerContainersCmd.Stderr = &stderrDockerContainers

					err = applyDockerContainersCmd.Run()
					dockerContainersSuccess := err == nil

					applyK8sDeploymentCmd := exec.Command("kubectl", "apply", "-f", k8sDeploymentTempFile.Name())
					var stderrK8sDeployment bytes.Buffer
					applyK8sDeploymentCmd.Stderr = &stderrK8sDeployment

					err = applyK8sDeploymentCmd.Run()
					k8sDeploymentSuccess := err == nil

					bar.Finish()

					if !linuxSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Linux: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrLinux.String())
						fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Linux.")
					}

					if !k8sSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Kubernetes: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrK8s.String())
						fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Kubernetes.")
					}

					if !dockerSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de laboratório Docker: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrDocker.String())
						fmt.Println("   A infraestrutura básica foi aplicada, mas sem o template de laboratório Docker.")
					}

					if !linuxUsersSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Usuários Linux: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrLinuxUsers.String())
					}

					if !linuxPermsSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Permissões Linux: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrLinuxPerms.String())
					}

					if !dockerContainersSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Containers Docker: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrDockerContainers.String())
					}

					if !k8sDeploymentSuccess {
						fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o template de Deployment Kubernetes: %v\n", err)
						fmt.Println("   Detalhes técnicos:", stderrK8sDeployment.String())
					}

					if linuxSuccess && k8sSuccess && dockerSuccess &&
						linuxUsersSuccess && linuxPermsSuccess &&
						dockerContainersSuccess && k8sDeploymentSuccess {
						fmt.Println("✅ Todos os templates de laboratório aplicados com sucesso!")

						// Verificação de diagnóstico para confirmar que os templates estão visíveis
						fmt.Println("\n🔍 Verificando templates de laboratório instalados:")
						listLabsCmd := exec.Command("kubectl", "get", "configmap", "-n", "girus", "-l", "app=girus-lab-template", "-o", "custom-columns=NAME:.metadata.name")

						// Capturar output para apresentá-lo de forma mais organizada
						var labsOutput bytes.Buffer
						listLabsCmd.Stdout = &labsOutput
						listLabsCmd.Stderr = &labsOutput

						if err := listLabsCmd.Run(); err == nil {
							labs := strings.Split(strings.TrimSpace(labsOutput.String()), "\n")
							if len(labs) > 1 { // Primeira linha é o cabeçalho "NAME"
								fmt.Println("   Templates encontrados:")
								for i, lab := range labs {
									if i > 0 { // Pular o cabeçalho
										fmt.Printf("   ✅ %s\n", strings.TrimSpace(lab))
									}
								}
							} else {
								fmt.Println("   ⚠️ Nenhum template de laboratório encontrado!")
							}
						} else {
							fmt.Println("   ⚠️ Não foi possível verificar os templates instalados")
						}

						// Reiniciar o backend para carregar os templates
						fmt.Println("\n🔄 Reiniciando o backend para carregar os templates...")
						restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
						restartCmd.Run()

						// Aguardar o reinício completar
						fmt.Println("   Aguardando o reinício do backend completar...")
						waitCmd := exec.Command("kubectl", "rollout", "status", "deployment/girus-backend", "-n", "girus", "--timeout=60s")
						// Redirecionar saída para não exibir detalhes do rollout
						var waitOutput bytes.Buffer
						waitCmd.Stdout = &waitOutput
						waitCmd.Stderr = &waitOutput

						// Iniciar indicador de progresso simples
						spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
						spinIdx := 0
						done := make(chan struct{})
						go func() {
							for {
								select {
								case <-done:
									return
								default:
									fmt.Printf("\r   %s Aguardando... ", spinChars[spinIdx])
									spinIdx = (spinIdx + 1) % len(spinChars)
									time.Sleep(100 * time.Millisecond)
								}
							}
						}()

						// Executar e aguardar
						waitCmd.Run()
						close(done)
						fmt.Println("\r   ✅ Backend reiniciado com sucesso!            ")

						// Aguardar mais alguns segundos para o backend inicializar completamente
						fmt.Println("   Aguardando inicialização completa...")
						time.Sleep(5 * time.Second)

					} else if linuxSuccess {
						fmt.Println("✅ Template de laboratório Linux aplicado com sucesso!")
					} else if k8sSuccess {
						fmt.Println("✅ Template de laboratório Kubernetes aplicado com sucesso!")
					} else if dockerSuccess {
						fmt.Println("✅ Template de laboratório Docker aplicado com sucesso!")
					}
				}
			}
		}

		// Aguardar os pods do Girus ficarem prontos
		if err := waitForPodsReady("girus", 5*time.Minute); err != nil {
			fmt.Fprintf(os.Stderr, "Aviso: %v\n", err)
			fmt.Println("Recomenda-se verificar o estado dos pods com 'kubectl get pods -n girus'")
		} else {
			fmt.Println("Todos os componentes do Girus estão prontos e em execução!")
		}

		fmt.Println("Girus implantado com sucesso no cluster!")

		// Configurar port-forward automaticamente (a menos que --skip-port-forward tenha sido especificado)
		if !skipPortForward {
			fmt.Print("\n🔌 Configurando acesso aos serviços do Girus... ")

			if err := setupPortForward("girus"); err != nil {
				fmt.Println("⚠️")
				fmt.Printf("Não foi possível configurar o acesso automático: %v\n", err)
				fmt.Println("\nVocê pode tentar configurar manualmente com os comandos:")
				fmt.Println("kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
				fmt.Println("kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
			} else {
				fmt.Println("✅")
				fmt.Println("Acesso configurado com sucesso!")
				fmt.Println("📊 Backend: http://localhost:8080")
				fmt.Println("🖥️  Frontend: http://0.0.0.0:8000")

				// Abrir o navegador se não foi especificado para pular
				if !skipBrowser {
					fmt.Println("\n🌐 Abrindo navegador com o Girus...")
					if err := openBrowser("http://localhost:8000"); err != nil {
						fmt.Printf("⚠️  Não foi possível abrir o navegador: %v\n", err)
						fmt.Println("   Acesse manualmente: http://localhost:8000")
					}
				}
			}
		} else {
			fmt.Println("\n⏩ Port-forward ignorado conforme solicitado")
			fmt.Println("\nPara acessar o Girus posteriormente, execute:")
			fmt.Println("kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
			fmt.Println("kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
		}

		// Exibir mensagem de conclusão
		fmt.Println("\n" + strings.Repeat("─", 60))
		fmt.Println("✅ GIRUS PRONTO PARA USO!")
		fmt.Println(strings.Repeat("─", 60))

		// Exibir acesso ao navegador como próximo passo
		fmt.Println("📋 PRÓXIMOS PASSOS:")
		fmt.Println("  • Acesse o Girus no navegador:")
		fmt.Println("    http://0.0.0.0:8000")

		// Instruções para laboratórios
		fmt.Println("\n  • Para aplicar mais templates de laboratórios com o Girus:")
		fmt.Println("    girus create lab -f caminho/para/lab.yaml")

		fmt.Println("\n  • Para ver todos os laboratórios disponíveis:")
		fmt.Println("    girus list labs")

		fmt.Println(strings.Repeat("─", 60))
	},
}

var createLabCmd = &cobra.Command{
	Use:   "lab [lab-id] ou -f [arquivo]",
	Short: "Cria um novo laboratório no Girus",
	Long:  "Adiciona um novo laboratório ao Girus a partir de um arquivo de manifesto ConfigMap, ou cria um ambiente de laboratório a partir de um ID de template existente.\nOs templates de laboratório são armazenados no diretório /labs na raiz do projeto.",
	Run: func(cmd *cobra.Command, args []string) {
		// Verificar qual modo estamos
		if labFile != "" {
			// Modo de adicionar template a partir de arquivo
			addLabFromFile(labFile, verboseMode)
		} else {
			fmt.Fprintf(os.Stderr, "Erro: Você deve especificar um arquivo de laboratório com a flag -f\n")
			fmt.Println("\nExemplo:")
			fmt.Println("  girus create lab -f meulaboratorio.yaml      # Adiciona um novo template a partir do arquivo")
			fmt.Println("  girus create lab -f /home/user/REPOS/strigus/labs/basic-linux.yaml      # Adiciona um template do diretório /labs")
			os.Exit(1)
		}
	},
}

// addLabFromFile adiciona um novo template de laboratório a partir de um arquivo
func addLabFromFile(labFile string, verboseMode bool) {
	// Verificar se o arquivo existe
	if _, err := os.Stat(labFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Erro: arquivo '%s' não encontrado\n", labFile)
		os.Exit(1)
	}

	fmt.Println("🔍 Verificando ambiente Girus...")

	// Verificar se há um cluster Girus ativo
	checkCmd := exec.Command("kubectl", "get", "namespace", "girus", "--no-headers", "--ignore-not-found")
	checkOutput, err := checkCmd.Output()
	if err != nil || !strings.Contains(string(checkOutput), "girus") {
		fmt.Fprintf(os.Stderr, "❌ Nenhum cluster Girus ativo encontrado\n")
		fmt.Println("   Use 'girus create cluster' para criar um cluster ou 'girus list clusters' para ver os disponíveis.")
		os.Exit(1)
	}

	// Verificar o pod do backend (silenciosamente, só mostra mensagem em caso de erro)
	backendCmd := exec.Command("kubectl", "get", "pods", "-n", "girus", "-l", "app=girus-backend", "-o", "jsonpath={.items[0].status.phase}")
	backendOutput, err := backendCmd.Output()
	if err != nil || string(backendOutput) != "Running" {
		fmt.Fprintf(os.Stderr, "❌ O backend do Girus não está em execução\n")
		fmt.Println("   Verifique o status dos pods com 'kubectl get pods -n girus'")
		os.Exit(1)
	}

	// Ler o arquivo para verificar se é um ConfigMap válido
	content, err := os.ReadFile(labFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Erro ao ler o arquivo '%s': %v\n", labFile, err)
		os.Exit(1)
	}

	// Verificação simples se o arquivo parece ser um ConfigMap válido
	fileContent := string(content)
	if !strings.Contains(fileContent, "kind: ConfigMap") ||
		!strings.Contains(fileContent, "app: girus-lab-template") {
		fmt.Fprintf(os.Stderr, "❌ O arquivo não é um manifesto de laboratório válido\n")
		fmt.Println("   O arquivo deve ser um ConfigMap com a label 'app: girus-lab-template'")
		os.Exit(1)
	}

	// Verificar se está instalando o lab do Docker e se o Docker está disponível
	if strings.Contains(fileContent, "docker-basics") {
		fmt.Println("🐳 Detectado laboratório de Docker, verificando dependências...")

		// Verificar se o Docker está instalado
		dockerCmd := exec.Command("docker", "--version")
		dockerInstalled := dockerCmd.Run() == nil

		// Verificar se o serviço está rodando
		dockerRunning := false
		if dockerInstalled {
			infoCmd := exec.Command("docker", "info")
			dockerRunning = infoCmd.Run() == nil
		}

		if !dockerInstalled || !dockerRunning {
			fmt.Println("⚠️  Aviso: Docker não está instalado ou não está em execução")
			fmt.Println("   O laboratório de Docker será instalado, mas requer Docker para funcionar corretamente.")
			fmt.Println("   Para instalar o Docker:")

			if runtime.GOOS == "darwin" {
				fmt.Println("\n   📦 macOS (via Colima):")
				fmt.Println("      brew install colima docker")
				fmt.Println("      colima start")
			} else if runtime.GOOS == "linux" {
				fmt.Println("\n   📦 Linux:")
				fmt.Println("      curl -fsSL https://get.docker.com | bash")
				fmt.Println("      sudo usermod -aG docker $USER")
				fmt.Println("      sudo systemctl start docker")
			} else {
				fmt.Println("\n   📦 Visite: https://www.docker.com/products/docker-desktop")
			}

			fmt.Println("\n   Você deseja continuar com a instalação do template? [s/N]")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response != "s" && response != "sim" && response != "y" && response != "yes" {
				fmt.Println("Instalação cancelada.")
				os.Exit(0)
			}

			fmt.Println("Continuando com a instalação do template Docker...")
		} else {
			fmt.Println("✅ Docker detectado e funcionando")
		}
	}

	fmt.Printf("📦 Processando laboratório: %s\n", labFile)

	// Aplicar o ConfigMap no cluster usando kubectl apply
	if verboseMode {
		fmt.Println("   Aplicando ConfigMap no cluster...")
	}

	// Aplicar o ConfigMap no cluster
	if verboseMode {
		// Executar normalmente mostrando o output
		applyCmd := exec.Command("kubectl", "apply", "-f", labFile)
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
		if err := applyCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Erro ao aplicar o laboratório: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Usar barra de progresso
		bar := progressbar.NewOptions(100,
			progressbar.OptionSetDescription("   Aplicando laboratório"),
			progressbar.OptionSetWidth(80),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
		)

		// Executar comando sem mostrar saída
		applyCmd := exec.Command("kubectl", "apply", "-f", labFile)
		var stderr bytes.Buffer
		applyCmd.Stderr = &stderr

		// Iniciar o comando
		err := applyCmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Erro ao iniciar o comando: %v\n", err)
			os.Exit(1)
		}

		// Atualizar a barra de progresso enquanto o comando está em execução
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					bar.Add(1)
					time.Sleep(50 * time.Millisecond)
				}
			}
		}()

		// Aguardar o final do comando
		err = applyCmd.Wait()
		close(done)
		bar.Finish()

		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Erro ao aplicar o laboratório: %v\n", err)
			if verboseMode {
				fmt.Fprintf(os.Stderr, "   Detalhes: %s\n", stderr.String())
			}
			os.Exit(1)
		}
	}

	// Extrair o ID do lab (name) do arquivo YAML para mostrar na mensagem
	var labID string
	// Procurar pela linha 'name:' dentro do bloco lab.yaml:
	labNameCmd := exec.Command("sh", "-c", fmt.Sprintf("grep -A10 'lab.yaml:' %s | grep 'name:' | head -1", labFile))
	labNameOutput, err := labNameCmd.Output()
	if err == nil {
		nameLine := strings.TrimSpace(string(labNameOutput))
		parts := strings.SplitN(nameLine, "name:", 2)
		if len(parts) >= 2 {
			labID = strings.TrimSpace(parts[1])
		}
	}

	// Extrair também o título para exibição
	var labTitle string
	labTitleCmd := exec.Command("sh", "-c", fmt.Sprintf("grep -A10 'lab.yaml:' %s | grep 'title:' | head -1", labFile))
	labTitleOutput, err := labTitleCmd.Output()
	if err == nil {
		titleLine := strings.TrimSpace(string(labTitleOutput))
		parts := strings.SplitN(titleLine, "title:", 2)
		if len(parts) >= 2 {
			labTitle = strings.TrimSpace(parts[1])
			labTitle = strings.Trim(labTitle, "\"'")
		}
	}

	fmt.Println("\n🔄 Reiniciando backend para carregar o template...")

	// O backend apenas carrega os templates na inicialização
	if verboseMode {
		// Mostrar o output da reinicialização
		fmt.Println("   (O backend do Girus carrega os templates apenas na inicialização)")
		restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
		restartCmd.Stdout = os.Stdout
		restartCmd.Stderr = os.Stderr
		if err := restartCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Erro ao reiniciar o backend: %v\n", err)
			fmt.Println("   O template foi aplicado, mas pode ser necessário reiniciar o backend manualmente:")
			fmt.Println("   kubectl rollout restart deployment/girus-backend -n girus")
		}

		// Aguardar o reinício completar
		fmt.Println("   Aguardando o reinício do backend completar...")
		waitCmd := exec.Command("kubectl", "rollout", "status", "deployment/girus-backend", "-n", "girus", "--timeout=60s")
		// Redirecionar saída para não exibir detalhes do rollout
		var waitOutput bytes.Buffer
		waitCmd.Stdout = &waitOutput
		waitCmd.Stderr = &waitOutput

		// Iniciar indicador de progresso simples
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinIdx := 0
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					fmt.Printf("\r   %s Aguardando... ", spinChars[spinIdx])
					spinIdx = (spinIdx + 1) % len(spinChars)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()

		// Executar e aguardar
		waitCmd.Run()
		close(done)
		fmt.Println("\r   ✅ Backend reiniciado com sucesso!            ")
	} else {
		// Usar barra de progresso
		bar := progressbar.NewOptions(100,
			progressbar.OptionSetDescription("   Reiniciando backend"),
			progressbar.OptionSetWidth(80),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
		)

		// Reiniciar o deployment do backend
		restartCmd := exec.Command("kubectl", "rollout", "restart", "deployment/girus-backend", "-n", "girus")
		var stderr bytes.Buffer
		restartCmd.Stderr = &stderr

		err := restartCmd.Run()
		if err != nil {
			bar.Finish()
			fmt.Fprintf(os.Stderr, "\n⚠️  Erro ao reiniciar o backend: %v\n", err)
			if verboseMode {
				fmt.Fprintf(os.Stderr, "   Detalhes: %s\n", stderr.String())
			}
			fmt.Println("   O template foi aplicado, mas pode ser necessário reiniciar o backend manualmente:")
			fmt.Println("   kubectl rollout restart deployment/girus-backend -n girus")
		} else {
			// Aguardar o reinício completar
			waitCmd := exec.Command("kubectl", "rollout", "status", "deployment/girus-backend", "-n", "girus", "--timeout=60s")

			// Redirecionar saída para não exibir detalhes do rollout
			var waitOutput bytes.Buffer
			waitCmd.Stdout = &waitOutput
			waitCmd.Stderr = &waitOutput

			// Iniciar o comando
			err = waitCmd.Start()
			if err != nil {
				bar.Finish()
				fmt.Fprintf(os.Stderr, "\n⚠️  Erro ao verificar status do reinício: %v\n", err)
			} else {
				// Atualizar a barra de progresso enquanto o comando está em execução
				done := make(chan struct{})
				go func() {
					for {
						select {
						case <-done:
							return
						default:
							bar.Add(1)
							time.Sleep(100 * time.Millisecond)
						}
					}
				}()

				// Aguardar o final do comando
				waitCmd.Wait()
				close(done)
				fmt.Println("\r   ✅ Backend reiniciado com sucesso!            ")
			}
			bar.Finish()
		}
	}

	// Aguardar mais alguns segundos para que o backend reinicie completamente
	fmt.Println("   Aguardando inicialização completa...")
	time.Sleep(3 * time.Second)

	// Após reiniciar o backend, verificar se precisamos recriar o port-forward
	portForwardStatus := checkPortForwardNeeded()

	// Se port-forward é necessário, configurá-lo corretamente
	if portForwardStatus {
		fmt.Println("\n🔌 Reconfigurando port-forwards após reinício do backend...")

		// Usar a função setupPortForward para garantir que ambos os serviços estejam acessíveis
		err := setupPortForward("girus")
		if err != nil {
			fmt.Println("⚠️ Aviso:", err)
			fmt.Println("   Para configurar manualmente, execute:")
			fmt.Println("   kubectl port-forward -n girus svc/girus-backend 8080:8080 --address 0.0.0.0")
			fmt.Println("   kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
		} else {
			fmt.Println("✅ Port-forwards configurados com sucesso!")
			fmt.Println("   🔹 Backend: http://localhost:8080")
			fmt.Println("   🔹 Frontend: http://0.0.0.0:8000")
		}
	} else {
		// Verificar conexão com o frontend mesmo que o port-forward não seja necessário
		checkCmd := exec.Command("curl", "-s", "--max-time", "1", "-o", "/dev/null", "-w", "%{http_code}", "http://0.0.0.0:8000")
		var out bytes.Buffer
		checkCmd.Stdout = &out

		if checkCmd.Run() != nil || !strings.Contains(strings.TrimSpace(out.String()), "200") {
			fmt.Println("\n⚠️ Detectado problema na conexão com o frontend.")
			fmt.Println("   Reconfigurando port-forwards para garantir acesso...")

			// Forçar reconfiguração de port-forwards
			err := setupPortForward("girus")
			if err != nil {
				fmt.Println("   ⚠️", err)
				fmt.Println("   Configure manualmente: kubectl port-forward -n girus svc/girus-frontend 8000:80 --address 0.0.0.0")
			} else {
				fmt.Println("   ✅ Port-forwards reconfigurados com sucesso!")
			}
		}
	}

	// Desenhar uma linha separadora
	fmt.Println("\n" + strings.Repeat("─", 60))

	// Exibir informações sobre o laboratório adicionado
	fmt.Println("✅ LABORATÓRIO ADICIONADO COM SUCESSO!")

	if labTitle != "" && labID != "" {
		fmt.Printf("\n📚 Título: %s\n", labTitle)
		fmt.Printf("🏷️  ID: %s\n", labID)
	} else if labID != "" {
		fmt.Printf("\n🏷️  ID do Laboratório: %s\n", labID)
	}

	fmt.Println("\n📋 PRÓXIMOS PASSOS:")
	fmt.Println("  • Acesse o Girus no navegador para usar o novo laboratório:")
	fmt.Println("    http://localhost:8000")

	fmt.Println("\n  • Para ver todos os laboratórios disponíveis via CLI:")
	fmt.Println("    girus list labs")

	fmt.Println("\n  • Para verificar detalhes do template adicionado:")
	if labID != "" {
		fmt.Printf("    kubectl describe configmap -n girus | grep -A20 %s\n", labID)
	} else {
		fmt.Println("    kubectl get configmaps -n girus -l app=girus-lab-template")
		fmt.Println("    kubectl describe configmap <nome-do-configmap> -n girus")
	}

	// Linha final
	fmt.Println(strings.Repeat("─", 60))
}

// checkPortForwardNeeded verifica se o port-forward para o backend precisa ser reconfigurado
func checkPortForwardNeeded() bool {
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
			frontendCheckCmd := exec.Command("curl", "-s", "--max-time", "2", "-o", "/dev/null", "-w", "%{http_code}", "http://0.0.0.0:8000")
			var out bytes.Buffer
			frontendCheckCmd.Stdout = &out
			if frontendCheckCmd.Run() != nil {
				frontendNeeded = true
			} else {
				statusCode := strings.TrimSpace(out.String())
				frontendNeeded = !(statusCode == "200" || statusCode == "301" || statusCode == "302")
			}
		}
	}

	// Se qualquer um dos serviços precisar de port-forward, retorne true
	return backendNeeded || frontendNeeded
}

func init() {
	createCmd.AddCommand(createClusterCmd)
	createCmd.AddCommand(createLabCmd)

	// Flags para createClusterCmd
	createClusterCmd.Flags().StringVarP(&deployFile, "file", "f", "", "Arquivo YAML para deployment do Girus (opcional)")
	createClusterCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Modo detalhado com output completo em vez da barra de progresso")
	createClusterCmd.Flags().BoolVarP(&skipPortForward, "skip-port-forward", "", false, "Não perguntar sobre configurar port-forwarding")
	createClusterCmd.Flags().BoolVarP(&skipBrowser, "skip-browser", "", false, "Não abrir o navegador automaticamente")

	// Flags para createLabCmd
	createLabCmd.Flags().StringVarP(&labFile, "file", "f", "", "Arquivo de manifesto do laboratório (ConfigMap)")
	createLabCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Modo detalhado com output completo em vez da barra de progresso")

	// definir o nome do cluster como "girus" sempre
	clusterName = "girus"
}
