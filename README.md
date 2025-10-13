![GIRUS](girus-logo.png)

**Escolha seu idioma / Elija su idioma:** [Português](README.md) | [Español](README.es.md)

# GIRUS: Plataforma de Laboratórios Interativos

Versão 0.3.0 Codename: "Maracatu" - Maio de 2025

## Visão Geral

GIRUS é uma plataforma open-source de laboratórios interativos que permite a criação, gerenciamento e execução de ambientes de aprendizado prático para tecnologias como Linux, Docker, Kubernetes, Terraform e outras ferramentas essenciais para profissionais de DevOps, SRE, Dev e Platform Engineering.

Desenvolvida pela LINUXtips, a plataforma GIRUS se diferencia por ser executada localmente na máquina do usuário, eliminando a necessidade de infraestrutura na nuvem ou configurações complexas. Através de um CLI intuitivo, os usuários podem criar rapidamente ambientes isolados e seguros onde podem praticar e aperfeiçoar suas habilidades técnicas.

## Principais Diferenciais

- **Execução Local**: Diferentemente de outras plataformas como Katacoda ou Instruqt que funcionam como SaaS, o GIRUS é executado diretamente na máquina do usuário através de containers Docker e Kubernetes, e o melhor, é que o projeto é open source e gratuito.
- **Ambientes Isolados**: Cada laboratório é executado em um ambiente isolado no Kubernetes, garantindo segurança e evitando conflitos com o sistema host
- **Interface Intuitiva**: Terminal interativo com tarefas guiadas e validação automática de progresso
- **Fácil Instalação**: CLI simples que gerencia todo o ciclo de vida da plataforma (criação, execução e exclusão)
- **Atualização Simplificada**: Comando `update` integrado que verifica, baixa e instala novas versões automaticamente
- **Laboratórios Personalizáveis**: Sistema de templates baseado em ConfigMaps do Kubernetes que facilita a criação de novos laboratórios
- **Open Source**: Projeto totalmente aberto para contribuições da comunidade
- **Multilíngue**: Além do português, o GIRUS agora oferece suporte oficial ao espanhol. O sistema de templates permite adicionar facilmente novos idiomas.

## Gerenciamento de Repositórios e Laboratórios

O GIRUS implementa um sistema robusto de gerenciamento de repositórios e laboratórios, similar ao Helm para Kubernetes. Este sistema permite:

### Instalação
```bash
curl -sSL girus.linuxtips.io | bash
```

Você precisa ter o Docker instalado em seu computador para poder instalar o Girus.

### Atualização da CLI

- **Verificar e Atualizar para a Última Versão**:
  ```bash
  girus update
  ```
  Este comando verifica se há uma versão mais recente do GIRUS CLI disponível, baixa e instala a atualização, oferecendo a opção de recriar o cluster após a atualização para garantir compatibilidade.

### Repositórios

- **Adicionar Repositórios**: 
  ```bash
  girus repo add linuxtips https://github.com/linuxtips/labs/raw/main
  ```

- **Listar Repositórios**:
  ```bash
  girus repo list
  ```

- **Remover Repositórios**:
  ```bash
  girus repo remove linuxtips
  ```

- **Atualizar Repositórios**:
  ```bash
  girus repo update linuxtips https://github.com/linuxtips/labs/raw/main
  ```

### Suporte a Repositórios Locais (file://)

O GIRUS agora suporta repositórios locais usando o prefixo `file://`. Isso é útil para testar laboratórios ou desenvolver repositórios sem precisar publicar em um servidor remoto.

#### Exemplo de uso:

```bash
# Adicionando um repositório local
./girus repo add meu-local file:///caminho/absoluto/para/seu-repo

# Exemplo prático:
./girus repo add test-repo file:///home/jeferson/REPOS/teste/girus-cli/test-repo
```

> **Nota:** O caminho após `file://` deve ser absoluto e apontar para o diretório onde está o `index.yaml` do repositório.

Você pode listar, buscar e instalar laboratórios normalmente a partir de repositórios locais, assim como faria com repositórios remotos.

### Laboratórios

- **Listar Laboratórios Disponíveis**:
  ```bash
  girus lab list
  ```

- **Instalar Laboratório**:
  ```bash
  girus lab install linuxtips linux-basics
  ```

- **Buscar Laboratórios**:
  ```bash
  girus lab search docker
  ```

### Estrutura de Repositórios

Os repositórios seguem uma estrutura padronizada:

```
repositorio/
├── index.yaml           # Índice do repositório
└── labs/               # Diretório contendo os laboratórios
    ├── lab1/
    │   ├── lab.yaml    # Definição do laboratório
    │   └── assets/     # Recursos do laboratório (opcional)
    └── lab2/
        ├── lab.yaml
        └── assets/
```

### Formato dos Arquivos

#### index.yaml
```yaml
apiVersion: v1
generated: "2024-03-20T10:00:00Z"
entries:
  lab-name:
    - name: lab-name
      version: "1.0.0"
      description: "Descrição do laboratório"
      keywords:
        - keyword1
        - keyword2
      maintainers:
        - "Nome <email@exemplo.com>"
      url: "https://github.com/seu-repo/raw/main/labs/lab-name/lab.yaml"
      created: "2024-03-20T10:00:00Z"
      digest: "sha256:hash-do-arquivo"
```

#### lab.yaml
```yaml
apiVersion: girus.linuxtips.io/v1
kind: Lab
metadata:
  name: lab-name
  version: "1.0.0"
  description: "Descrição do laboratório"
  author: "Nome do Autor"
  created: "2024-03-20T10:00:00Z"
spec:
  environment:
    image: ubuntu:22.04
    resources:
      cpu: "1"
      memory: "1Gi"
    volumes:
      - name: workspace
        mountPath: /workspace
        size: "1Gi"

  tasks:
    - name: "Nome da Tarefa"
      description: "Descrição da tarefa"
      steps:
        - description: "Descrição do passo"
          command: "comando"
          expectedOutput: "saída esperada"
          hint: "Dica para o usuário"

  validation:
    - name: "Nome da Validação"
      description: "Descrição da validação"
      checks:
        - command: "comando"
          expectedOutput: "saída esperada"
          errorMessage: "Mensagem de erro"
```

## Arquitetura

O projeto GIRUS é composto por quatro componentes principais:

1. **GIRUS CLI**: Ferramenta de linha de comando que gerencia todo o ciclo de vida da plataforma
2. **Backend**: API Golang que orquestra os laboratórios através da API do Kubernetes
3. **Frontend**: Interface web React que fornece acesso ao terminal interativo e às tarefas
4. **Templates de Laboratórios**: Definições YAML para os diferentes laboratórios disponíveis

### Diagrama de Fluxo de Arquitetura

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  GIRUS CLI  │────▶│ Kind Cluster │────▶│ Kubernetes   │
└─────────────┘     └──────────────┘     └──────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Terminal   │◀───▶│   Frontend   │◀───▶│   Backend    │
│ Interativo  │     │    (React)   │     │     (Go)     │
└─────────────┘     └──────────────┘     └──────────────┘
                                               │
                                               ▼
                                         ┌──────────────┐
                                         │  Templates   │
                                         │     Labs     │
                                         └──────────────┘
```

## Componentes Detalhados

### GIRUS CLI

GIRUS (GIRUS Is Really Useful System) é uma ferramenta CLI desenvolvida pela LINUXtips para criar e gerenciar ambientes de laboratório práticos.

## Instalação

### Usando o script de instalação

```bash
curl -sSL girus.linuxtips.io | bash
```

### Usando o Makefile

Clone o repositório e execute `make <comando>`.

Aqui estão os comandos disponíveis:

### Compilação e Instalação

* **`make build`** (ou simplesmente `make`): Compila o binário `girus` para o seu sistema operacional atual e o coloca no diretório `dist/`. Este é o comando padrão se você executar `make` sem argumentos.
* **`make install`**: Compila o binário (se ainda não estiver compilado) e o move para `/usr/local/bin/girus`, tornando-o acessível globalmente no seu sistema. Requer permissões de superusuário (`sudo`).
* **`make clean`**: Remove o diretório `dist/` e todos os arquivos de build gerados.
* **`make release`**: Compila o binário `girus` para múltiplas plataformas (Linux, macOS, Windows - amd64 e arm64) e os coloca no diretório `dist/`.

### Versionamento

O GIRUS CLI utiliza um sistema de versionamento dinâmico baseado em git tags. O processo de build detecta automaticamente a versão com base nos seguintes critérios:

* Se existir uma tag git (ex: `v0.3.0`), essa versão será utilizada removendo o prefixo `v` (resultado: `0.3.0`)
* Se não existirem tags, será utilizada a versão padrão `0.3.0`
* Para builds locais, você pode compilar com uma versão específica através do seguinte comando:

```bash
go build -o girus -ldflags="-X 'github.com/badtuxx/girus-cli/internal/common.Version=0.3.0'" ./main.go
```

Para verificar a versão atual do binário, execute:

```bash
./girus version
```

Os workflows CI/CD do projeto também utilizam este mecanismo de versionamento dinâmico para as builds do Docker e artefatos de release, garantindo consistência em todo o processo de build.

### Gerenciamento de Dependências (Go Modules)

* **`make check-updates`**: Verifica se há atualizações disponíveis para as dependências Go do projeto.
* **`make upgrade-all`**: Atualiza todas as dependências Go para suas versões mais recentes e executa `go mod tidy`.
* **`make upgrade MODULE=<nome/do/modulo>`**: Atualiza uma dependência Go específica para a versão mais recente. Substitua `<nome/do/modulo>` pelo caminho do módulo (ex: `make upgrade MODULE=github.com/spf13/cobra`).
* **`make tidy`**: Executa `go mod tidy` para remover dependências não utilizadas e limpar os arquivos `go.mod` e `go.sum`.
* **`make deps`**: Exibe o gráfico de dependências do projeto.

### Primeiros Passos
## Criando seu Primeiro Cluster
Após instalar o GIRUS, o primeiro passo é criar um cluster Kubernetes local usando Kind:

  ```bash
  girus create cluster
  ```
## Este comando irá:

- **Verificar se o Docker está rodando**
- **Instalar o Kind (se não estiver presente)**
- **Criar um cluster Kubernetes local**
- **Instalar os componentes do GIRUS (backend, frontend, etc.)**
- **Configurar os serviços necessários**

> **Nota:** O processo pode levar alguns minutos na primeira execução, pois precisa baixar as imagens Docker necessárias.

- **Verificando o Status do Cluster**:
Para verificar se o cluster foi criado com sucesso:
  ```bash
  # Verificar clusters Kind disponíveis
  kind get clusters

  # Verificar pods do GIRUS
  kubectl get pods -n girus

  # Verificar serviços do GIRUS
  kubectl get services -n girus
  ```

**Gerenciando o Cluster**:
Para verificar o status:
  ```bash
  # Listar clusters disponíveis
  kind get clusters

  # Verificar se o cluster está saudável
  kubectl cluster-info
  ```

 **Deletar o Cluster**:

  ```bash
  # Remover o cluster quando não precisar mais
  kind delete cluster --name girus
  ```
**Recriar o Cluster**:

  ```bash
  # Se precisar recriar o cluster
  kind delete cluster --name girus
  girus create cluster
  ```

## Repositório de Labs

Este repositório contém uma coleção de labs práticos para diferentes tecnologias, organizados nas seguintes categorias:

### AWS Labs
- AWS LocalStack com Terraform
- AWS S3 Storage
- AWS DynamoDB NoSQL
- AWS Lambda Serverless

### Terraform Labs
- Fundamentos do Terraform
- Terraform com AWS
- Provisioners e Módulos no Terraform

### Kubernetes Labs
- Fundamentos do Kubernetes
- Deployment no Kubernetes
- Exploração de Recursos
- Serviços e Redes
- ConfigMaps e Secrets
- CronJobs

### Docker Labs
- Fundamentos do Docker
- Gerenciamento de Containers
- Fundamentos de Redes
- Volumes
- Docker Compose

### Linux Labs
- Comandos Básicos
- Gerenciamento de Usuários
- Permissões de Arquivos
- Processamento de Texto
- Gerenciamento de Processos
- Shell Script
- Monitoramento de Sistema

## Usando os Labs

### Adicionar o Repositório

```bash
# Adicionar o repositório oficial
girus repo add girus-cli https://raw.githubusercontent.com/badtuxx/girus-cli/main/index.yaml

# Ou adicionar localmente para desenvolvimento
girus repo add girus-cli file:///caminho/para/girus-cli
```

### Listar Labs Disponíveis

```bash
girus lab list
```

### Iniciar um Lab

```bash
girus lab start <nome-do-lab>
```

Por exemplo:
```bash
girus lab start aws_localstack_terraform
```

## Contribuindo com Labs

Para contribuir com novos labs, siga estas etapas:

1. Crie um novo diretório em `labs/<nome-do-lab>`
2. Adicione um arquivo `lab.yaml` com a estrutura do lab
3. Atualize o `index.yaml` com as informações do novo lab
4. Envie um Pull Request

### Estrutura do Lab

```yaml
name: nome-do-lab
title: "Título do Lab"
description: "Descrição detalhada do lab"
duration: 45m
image: "ubuntu:20.04"
tasks:
  - name: "Nome da Tarefa"
    description: "Descrição da tarefa"
    steps:
      - "Passo 1: Faça isso"
      - "Passo 2: Execute aquilo"
    validation:
      - command: "comando para verificar"
        expectedOutput: "saída esperada"
        errorMessage: "Mensagem de erro personalizada"
```
## Integração com Sistemas de Aprendizado

O GIRUS foi criado pela LINUXtips para melhorar a experiência de aprendizado dos alunos, mas não é restrito aos alunos da LINUXtips, você pode usar o GIRUS em qualquer lugar que você precise de um ambiente prático para aprender ou testar novas tecnologias, por exemplo:

1. **Complemento para Cursos Online**:
   - Ambiente prático para reforçar conteúdo teórico
   - Validação automatizada de exercícios
   - Experiência hands-on sem necessidade de infraestrutura adicional

2. **Treinamentos Corporativos**:
   - Ambientes padronizados para todos os participantes
   - Fácil distribuição de exercícios práticos
   - Redução de custos com infraestrutura

3. **Preparação para Certificações**:
   - Simulação de ambientes similares aos exames
   - Tarefas baseadas em syllabus oficial
   - Feedback imediato sobre o progresso

## Contribuição e Comunidade

O GIRUS é um projeto open-source que depende da contribuição da comunidade para crescer. Existem várias formas de contribuir:

### Desenvolvimento
- Correção de bugs e melhorias no código-fonte
- Implementação de novos recursos
- Otimização de performance
- Testes e garantia de qualidade

### Criação de Conteúdo
- Desenvolvimento de novos templates de laboratórios
- Tradução do conteúdo para outros idiomas
- Elaboração de tutoriais e documentação

### Divulgação
- Compartilhamento do projeto nas redes sociais
- Apresentações em eventos e conferências
- Artigos sobre o uso e benefícios da plataforma

### Processo de Contribuição
1. Fork o repositório em [GitHub](https://github.com/badtuxx/girus)
2. Crie uma branch para sua feature (`git checkout -b feature/nova-funcionalidade`)
3. Faça suas alterações e commit (`git commit -m 'Adiciona nova funcionalidade'`)
4. Push para a branch (`git push origin feature/nova-funcionalidade`)
5. Abra um Pull Request

## Roadmap de Desenvolvimento

O projeto GIRUS tem um plano de desenvolvimento contínuo com os seguintes objetivos:

### Curto Prazo (3-6 meses)
- Expansão da biblioteca de laboratórios com o Girus Hub
- Melhorias na interface do usuário
- Suporte a múltiplos idiomas
- Integração com sistemas de badges/conquistas

### Médio Prazo (6-12 meses)
- Implementação de recursos colaborativos
- Sistema de competições e desafios
- Métricas avançadas de progresso
- Suporte a plugins de extensão

## Suporte e Contato

* **GitHub Issues**: [github.com/badtuxx/girus-cli/issues](https://github.com/badtuxx/girus-cli/issues)
* **GitHub Discussions**: [github.com/badtuxx/girus-cli/discussions](https://github.com/badtuxx/girus-cli/discussions)
* **Discord da Comunidade**: [discord.gg/linuxtips](https://discord.gg/linuxtips)

## Licença

Este projeto é distribuído sob a licença GPL-3.0. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## Agradecimentos

O GIRUS é possível graças à contribuição de muitas pessoas e projetos:

- **Equipe LINUXtips**: Pelo desenvolvimento e manutenção do projeto
- **Contribuidores**: Desenvolvedores, criadores de conteúdo e tradutores
- **Projetos Open Source**: Go, React, Kubernetes, Kind, Docker e muitos outros
- **Comunidade**: Todos os usuários e apoiadores que acreditam no projeto

---

## FAQ - Perguntas Frequentes

**Q: O GIRUS funciona offline?**  
A: Sim, após a instalação inicial e download das imagens, o GIRUS pode funcionar completamente offline.

**Q: Quanto consome de recursos da minha máquina?**  
A: O GIRUS é otimizado para ser leve. Um cluster básico consome aproximadamente 1-2GB de RAM e requer cerca de 5GB de espaço em disco.

**Q: Posso criar laboratórios personalizados para minha equipe/empresa?**  
A: Absolutamente! O sistema de templates é flexível e permite a criação de laboratórios específicos para suas necessidades.

**Q: Como faço para atualizar o GIRUS para a versão mais recente?**  
A: Execute o comando `girus update`. O comando verificará se há uma versão mais recente disponível e, se houver, executará a atualização automaticamente. Após a atualização, você terá a opção de recriar o cluster para garantir a compatibilidade com as novas funcionalidades.

**Q: O GIRUS funciona em ambientes corporativos com restrições de rede?**  
A: Sim, após o download inicial das imagens, o GIRUS opera localmente sem necessidade de conexão externa.

**Q: Posso contribuir com novos laboratórios para o projeto?**  
A: Definitivamente! Contribuições são bem-vindas e valorizadas. Consulte a seção ["Contribuição e Comunidade"](#contribui%C3%A7%C3%A3o-e-comunidade) para detalhes.
