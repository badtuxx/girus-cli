#!/usr/bin/env bash
set -e

# ASCII Art Banner para o Girus
cat << "EOF"
   ██████╗ ██╗██████╗ ██╗   ██╗███████╗
  ██╔════╝ ██║██╔══██╗██║   ██║██╔════╝
  ██║  ███╗██║██████╔╝██║   ██║███████╗
  ██║   ██║██║██╔══██╗██║   ██║╚════██║
  ╚██████╔╝██║██║  ██║╚██████╔╝███████║
   ╚═════╝ ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝
EOF

# Perguntar idioma antes de qualquer saída relevante
read -p "Escolha o idioma / Elija el idioma (pt/es) [pt]: " CLI_LANG
CLI_LANG=${CLI_LANG:-pt}
mkdir -p "$HOME/.girus"
echo "language: $CLI_LANG" > "$HOME/.girus/config.yaml"

# Função simples para traduzir mensagens
t() {
    local pt="$1"
    local es="$2"
    if [ "$CLI_LANG" = "es" ]; then
        eval echo "\"$es\""
    else
        eval echo "\"$pt\""
    fi
}

echo -e "\n$(t 'Script de Instalação - Versão 0.5.0 - Codename: Maracatu' 'Script de Instalación - Versión 0.5.0 - Codename: Maracatu')\n"

# Verificar se o terminal é interativo
IS_INTERACTIVE=0
if [ -t 0 ]; then
    IS_INTERACTIVE=1
fi

# Forçar modo interativo para o script completo
IS_INTERACTIVE=1

# Função para pedir confirmação ao usuário (interativo) ou mostrar ação padrão (não-interativo)
ask_user() {
    local prompt="$1"
    local default="$2"
    local variable_name="$3"
    
    # Modo sempre interativo - perguntar ao usuário
    echo -n "$prompt: "
    read response

    # Se resposta for vazia, usar o padrão
    response=${response:-$default}
    
    # Exportar a resposta para a variável solicitada
    eval "$variable_name=\"$response\""
}

# Verificar se o script está sendo executado como root (sudo)
if [ "$(id -u)" -eq 0 ]; then
    echo "$(t '❌ ERRO: Este script não deve ser executado como root ou com sudo.' '❌ ERROR: Este script no debe ejecutarse como root o con sudo.')"
    echo "$(t '   Por favor, execute sem sudo. O script solicitará elevação quando necessário.' '   Por favor, ejecútelo sin sudo. El script solicitará elevación cuando sea necesario.')"
    exit 1
fi

# Configuração de variáveis e ambiente
set -e

# Detectar o sistema operacional
case "$(uname -s)" in
    Linux*) OS="linux" ;;
    Darwin*) OS="darwin" ;;
    CYGWIN*|MINGW*|MSYS*) OS="windows" ;;
    *) OS="unknown" ;;
esac

# Verificar distribuição
if [ "$OS" == "linux" ]; then
    DISTRO=""
	if [ -r /etc/os-release ]; then
		DISTRO="$(. /etc/os-release && echo "$ID")"
	fi
fi

# Detectar a arquitetura
ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64) ARCH="amd64" ;;
    amd64) ARCH="amd64" ;;
    arm64) ARCH="arm64" ;;
    aarch64) ARCH="arm64" ;;
    *) ARCH="unknown" ;;
esac

echo "$(t 'Sistema operacional detectado: $OS' 'Sistema operativo detectado: $OS')"
echo "$(t 'Arquitetura detectada: $ARCH' 'Arquitectura detectada: $ARCH')"

# Verificar se o sistema operacional é suportado
if [ "$OS" == "unknown" ]; then
    echo "$(t '❌ Sistema operacional não suportado: $(uname -s)' '❌ Sistema operativo no soportado: $(uname -s)')"
    exit 1
fi

# Verificar se a arquitetura é suportada
if [ "$ARCH" == "unknown" ]; then
    echo "$(t '❌ Arquitetura não suportada: $ARCH_RAW' '❌ Arquitectura no soportada: $ARCH_RAW')"
    exit 1
fi

# Configurações e variáveis
GIRUS_VERSION="v0.5.0"

# Definir URL com base no sistema operacional e arquitetura
if [ "$OS" == "windows" ]; then
    BINARY_URL="https://github.com/badtuxx/girus-cli/releases/download/$GIRUS_VERSION/girus-cli-$OS-$ARCH.exe"
else
    BINARY_URL="https://github.com/badtuxx/girus-cli/releases/download/$GIRUS_VERSION/girus-cli-$OS-$ARCH"
fi

echo "$(t 'URL de download: $BINARY_URL' 'URL de descarga: $BINARY_URL')"
ORIGINAL_DIR=$(pwd)
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Configurações e variáveis
GIRUS_CODENAME="Maracatu"
KIND_VERSION="0.27.0"
DOWNLOAD_TOOL="none"

# Função para verificar se o comando curl ou wget está disponível
check_download_tool() {
    if command -v curl &> /dev/null; then
        echo "curl"
    elif command -v wget &> /dev/null; then
        echo "wget"
    else
        echo "none"
    fi
}

# Função para instalar Docker
install_docker() {
    echo "$(t 'Instalando Docker...' 'Instalando Docker...')"

    if [[ "$OS" == "linux" && "$DISTRO" != "rocky" ]]; then
        # Linux (script de conveniência do Docker)
        echo "$(t 'Baixando o script de instalação do Docker...' 'Descargando el script de instalación de Docker...')"
        curl -fsSL https://get.docker.com -o get-docker.sh
        echo "$(t 'Executando o script de instalação (será solicitada senha de administrador)...' 'Ejecutando el script de instalación (se solicitará contraseña de administrador)...')"
        sudo sh get-docker.sh

        # Adicionar usuário atual ao grupo docker
        echo "$(t 'Adicionando usuário atual ao grupo docker...' 'Añadiendo el usuario actual al grupo docker...')"
        sudo usermod -aG docker $USER

        # Iniciar o serviço
        echo "$(t 'Iniciando o serviço Docker...' 'Iniciando el servicio Docker...')"
        sudo systemctl enable --now docker

        # Limpar arquivo de instalação
        rm get-docker.sh

    elif [[ "$OS" == "linux" && "$DISTRO" == "rocky" ]]; then
        # instalando docker no rocky linux (padrão podman)
        echo "$(t 'Instalando o docker (será solicitada senha de administrador)...' 'Instalando Docker (se solicitará contraseña de administrador)...')"
        echo "$(t 'Adicionando repositório do docker...' 'Añadiendo repositorio de docker...')"
        sudo dnf config-manager --add-repo https://download.docker.com/linux/rhel/docker-ce.repo
        sudo dnf -yq install docker-ce docker-ce-cli containerd.io docker-compose-plugin

        # Adicionar usuário atual ao grupo docker
        echo "$(t 'Adicionando usuário atual ao grupo docker...' 'Añadiendo el usuario actual al grupo docker...')"
        sudo usermod -aG docker $USER

        # Iniciar o serviço
        echo "$(t 'Iniciando o serviço Docker...' 'Iniciando el servicio Docker...')"
        sudo systemctl enable --now docker

    elif [ "$OS" == "darwin" ]; then
        # MacOS
        echo "$(t 'No macOS, o Docker Desktop precisa ser instalado manualmente.' 'En macOS, Docker Desktop debe instalarse manualmente.')"
        echo "$(t 'Por favor, baixe e instale o Docker Desktop para Mac:' 'Por favor, descarga e instala Docker Desktop para Mac:')"
        echo "https://docs.docker.com/desktop/mac/install/"
        echo "$(t 'Após a instalação, reinicie seu terminal e execute este script novamente.' 'Después de la instalación, reinicia tu terminal y ejecuta este script nuevamente.')"
        exit 1

    elif [ "$OS" == "windows" ]; then
        # Windows
        echo "$(t 'No Windows, o Docker Desktop precisa ser instalado manualmente.' 'En Windows, Docker Desktop debe instalarse manualmente.')"
        echo "$(t 'Por favor, baixe e instale o Docker Desktop para Windows:' 'Por favor, descarga e instala Docker Desktop para Windows:')"
        echo "https://docs.docker.com/desktop/windows/install/"
        echo "$(t 'Após a instalação, reabra o terminal e execute este script novamente.' 'Después de la instalación, vuelve a abrir la terminal y ejecuta este script nuevamente.')"
        exit 1
    fi

    # Verificar a instalação
    if ! command -v docker &> /dev/null; then
        echo "$(t '❌ Falha ao instalar o Docker.' '❌ Error al instalar Docker.')"
        echo "$(t 'Por favor, instale manualmente seguindo as instruções em https://docs.docker.com/engine/install/' 'Por favor, instálalo manualmente siguiendo las instrucciones en https://docs.docker.com/engine/install/')"
        exit 1
    fi

    echo "$(t 'Docker instalado com sucesso!' 'Docker instalado con éxito!')"
    echo "$(t 'NOTA: Pode ser necessário reiniciar seu sistema ou fazer logout/login para que as permissões de grupo sejam aplicadas.' 'NOTA: Puede ser necesario reiniciar tu sistema o cerrar sesión/iniciar sesión para aplicar los permisos del grupo.')"
}

# Função para instalar Kind
install_kind() {
    echo "Instalando Kind..."

    if [ "$OS" == "linux" ] || [ "$OS" == "darwin" ]; then
        # Linux/Mac
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-$(uname)-${ARCH}
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind

    elif [ "$OS" == "windows" ]; then
        # Windows
        echo "$(t 'Instalação automática do Kind não suportada no Windows.' 'La instalación automática de Kind no es compatible en Windows.')"
        echo "$(t 'Por favor, baixe e instale Kind manualmente:' 'Por favor, descarga e instala Kind manualmente:')"
        echo "https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        echo "$(t 'Após a instalação, reabra o terminal e execute este script novamente.' 'Después de la instalación, vuelve a abrir la terminal y ejecuta este script nuevamente.')"
        exit 1
    fi

    # Verificar a instalação
    if ! command -v kind &> /dev/null; then
        echo "$(t 'Falha ao instalar o Kind. Por favor, instale manualmente.' 'Error al instalar Kind. Por favor, instálalo manualmente.')"
        exit 1
    fi

    echo "$(t 'Kind instalado com sucesso!' 'Kind instalado con éxito!')"
}

# Função para instalar Kubectl
install_kubectl() {
    echo "$(t 'Instalando Kubectl...' 'Instalando Kubectl...')"

    if [ "$OS" == "linux" ]; then
        # Linux
        curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${ARCH}/kubectl"
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/

    elif [ "$OS" == "darwin" ]; then
        # MacOS
        if command -v brew &> /dev/null; then
            brew install kubectl
        else
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/${ARCH}/kubectl"
            chmod +x kubectl
            sudo mv kubectl /usr/local/bin/
        fi

    elif [ "$OS" == "windows" ]; then
        # Windows
        echo "$(t 'Instalação automática do Kubectl não suportada no Windows.' 'La instalación automática de Kubectl no es compatible en Windows.')"
        echo "$(t 'Por favor, baixe e instale Kubectl manualmente:' 'Por favor, descarga e instala Kubectl manualmente:')"
        echo "https://kubernetes.io/docs/tasks/tools/install-kubectl/"
        echo "$(t 'Após a instalação, reabra o terminal e execute este script novamente.' 'Después de la instalación, vuelve a abrir la terminal y ejecuta este script nuevamente.')"
        exit 1
    fi

    # Verificar a instalação
    if ! command -v kubectl &> /dev/null; then
        echo "$(t 'Falha ao instalar o Kubectl. Por favor, instale manualmente.' 'Error al instalar Kubectl. Por favor, instálalo manualmente.')"
        exit 1
    fi

    echo "$(t 'Kubectl instalado com sucesso!' 'Kubectl instalado con éxito!')"
}

# Função para verificar se o Docker está em execução
check_docker_running() {
    if docker info &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# Função para verificar a versão do GLIBC
check_glibc_version() {
    # Skip GLIBC check on non-Linux systems
    if [ "$OS" != "linux" ]; then
        return 0
    fi

    if command -v ldd &> /dev/null; then
        GLIBC_VERSION=$(ldd --version | head -n 1 | grep -oP '\d+\.\d+' | head -n 1)
        if [ -z "$GLIBC_VERSION" ]; then
            echo "$(t '❌ Não foi possível detectar a versão do GLIBC.' '❌ No fue posible detectar la versión de GLIBC.')"
            return 1
        fi

        # Converter versão para número para comparação
        GLIBC_VERSION_NUM=$(echo $GLIBC_VERSION | awk -F. '{printf "%d.%02d", $1, $2}')
        MIN_GLIBC_VERSION_NUM=2.17

        if (( $(echo "$GLIBC_VERSION_NUM >= $MIN_GLIBC_VERSION_NUM" | bc -l) )); then
            echo "$(t '✅ GLIBC versão $GLIBC_VERSION detectada (mínimo requerido: 2.17)' '✅ GLIBC versión $GLIBC_VERSION detectada (mínimo requerido: 2.17)')"
            return 0
        else
            echo "$(t '❌ GLIBC versão $GLIBC_VERSION detectada (mínimo requerido: 2.17)' '❌ GLIBC versión $GLIBC_VERSION detectada (mínimo requerido: 2.17)')"
            echo "$(t 'Por favor, atualize o GLIBC para uma versão mais recente.' 'Por favor, actualiza GLIBC a una versión más reciente.')"
            return 1
        fi
    else
        echo "$(t '❌ Comando ldd não encontrado. Não foi possível verificar a versão do GLIBC.' '❌ Comando ldd no encontrado. No fue posible verificar la versión de GLIBC.')"
        return 1
    fi
}

# Verificar se o Girus CLI está no PATH
check_girus_in_path() {
    if command -v girus &> /dev/null; then
        # Se o Girus estiver instalado, verificar a versão do GLIBC
        if ! check_glibc_version; then
            echo "⚠️ Problema de compatibilidade detectado com o GLIBC."
            echo " Por favor, siga as instruções acima para resolver o problema."
            exit 1
        fi
        return 0
    else
        return 1
    fi
}

# Função para verificar instalações anteriores do Girus CLI
check_previous_install() {
    local previous_install_found=false
    local install_locations=(
        "/usr/local/bin/girus"
        "/usr/bin/girus"
        "$HOME/.local/bin/girus"
        "./girus"
    )

    # Verificar instalações anteriores
    for location in "${install_locations[@]}"; do
        if [ -f "$location" ]; then
            echo "$(t '⚠️ Instalação anterior encontrada em: $location' '⚠️ Instalación anterior encontrada en: $location')"
            previous_install_found=true
        fi
    done

    # Se uma instalação anterior foi encontrada, perguntar sobre limpeza
    if [ "$previous_install_found" = true ]; then
        ask_user "$(t 'Deseja remover a(s) instalação(ões) anterior(es)? (S/n): ' '¿Desea remover la(s) instalación(es) anterior(es)? (S/n): ')" "S" "CLEAN_INSTALL"

        if [[ "$CLEAN_INSTALL" =~ ^[Ss]$ ]]; then
            echo "$(t '🧹 Removendo instalações anteriores...' '🧹 Eliminando instalaciones anteriores...')"

            for location in "${install_locations[@]}"; do
                if [ -f "$location" ]; then
                    echo "$(t 'Removendo $location' 'Eliminando $location')"
                    if [[ "$location" == "/usr/local/bin/girus" || "$location" == "/usr/bin/girus" ]]; then
                        sudo rm -f "$location"
                    else
                        rm -f "$location"
                    fi
                fi
            done

            echo "$(t '✅ Limpeza concluída.' '✅ Limpieza completada.')"
        else
            echo "$(t 'Continuando com a instalação sem remover versões anteriores.' 'Continuando la instalación sin eliminar versiones anteriores.')"
        fi
    else
        echo "$(t '✅ Nenhuma instalação anterior do Girus CLI encontrada.' '✅ No se encontró una instalación previa de Girus CLI.')"
    fi
}

# Função para baixar e instalar o binário
download_and_install() {
    echo "$(t '📥 Baixando o Girus CLI versão $GIRUS_VERSION para $OS-$ARCH...' '📥 Descargando Girus CLI versión $GIRUS_VERSION para $OS-$ARCH...')"
    cd "$TEMP_DIR"

    # Verificar qual ferramenta de download está disponível
    DOWNLOAD_TOOL=$(check_download_tool)

    if [ "$DOWNLOAD_TOOL" == "curl" ]; then
        echo "$(t 'Usando curl para download de: $BINARY_URL' 'Usando curl para descargar de: $BINARY_URL')"
        echo "$(t 'Executando: curl -L --progress-bar \"$BINARY_URL\" -o girus' 'Ejecutando: curl -L --progress-bar \"$BINARY_URL\" -o girus')"
        if ! curl -L --progress-bar "$BINARY_URL" -o girus; then
            echo "$(t '❌ Erro no curl. Tentando com opções de debug...' '❌ Error en curl. Probando con opciones de depuración...')"
            curl -L -v "$BINARY_URL" -o girus
        fi
    else
        echo "$(t 'Usando wget para download de: $BINARY_URL' 'Usando wget para descargar de: $BINARY_URL')"
        echo "$(t 'Executando: wget --show-progress -q \"$BINARY_URL\" -O girus' 'Ejecutando: wget --show-progress -q \"$BINARY_URL\" -O girus')"
        if ! wget --show-progress -q "$BINARY_URL" -O girus; then
            echo "$(t '❌ Erro no wget. Tentando com opções de debug...' '❌ Error en wget. Probando con opciones de depuración...')"
            wget -v "$BINARY_URL" -O girus
        fi
    fi

    # Verificar se o download foi bem-sucedido
    if [ ! -f girus ] || [ ! -s girus ]; then
        echo "$(t '❌ Erro: Falha ao baixar o Girus CLI.' '❌ Error: Fallo al descargar Girus CLI.')"
        echo "$(t 'URL: $BINARY_URL' 'URL: $BINARY_URL')"
        echo "$(t 'Verifique sua conexão com a internet e se a versão $GIRUS_VERSION está disponível.' 'Verifica tu conexión a internet y si la versión $GIRUS_VERSION está disponible.')"
        exit 1
    fi

    # Tornar o binário executável
    chmod +x girus

    # Perguntar se o usuário deseja instalar no PATH
    echo "$(t '🔧 Girus CLI baixado com sucesso.' '🔧 Girus CLI descargado con éxito.')"
    ask_user "$(t 'Deseja instalar o Girus CLI em /usr/local/bin? (S/n): ' '¿Desea instalar el Girus CLI en /usr/local/bin? (S/n): ')" "S" "INSTALL_GLOBALLY"

    if [[ "$INSTALL_GLOBALLY" =~ ^[Ss]$ ]]; then
        echo "$(t '📋 Instalando o Girus CLI em /usr/local/bin/girus...' '📋 Instalando Girus CLI en /usr/local/bin/girus...')"
        sudo mv girus /usr/local/bin/
        echo "$(t '✅ Girus CLI instalado com sucesso em /usr/local/bin/girus' '✅ Girus CLI instalado con éxito en /usr/local/bin/girus')"
        echo "$(t ' Você pode executá-lo de qualquer lugar com o comando '\''girus'\''' ' Puede ejecutarlo desde cualquier lugar con el comando '\''girus'\''')"
    else
        # Copiar para o diretório original
        cp girus "$ORIGINAL_DIR/"
        echo "$(t '✅ Girus CLI copiado para o diretório atual: $(realpath "$ORIGINAL_DIR/girus")' '✅ Girus CLI copiado al directorio actual: $(realpath "$ORIGINAL_DIR/girus")')"
        echo "$(t ' Você pode executá-lo com: '\''./girus'\''' ' Puede ejecutarlo con: '\''./girus'\''')"
    fi
}

# Verificar se todas as dependências estão instaladas
verify_all_dependencies() {
    local all_deps_ok=true

    # Verificar Docker
    if command -v docker &> /dev/null && check_docker_running; then
        echo "$(t '✅ Docker está instalado e em execução.' '✅ Docker está instalado y en ejecución.')"
    else
        echo "$(t '❌ Docker não está instalado, não está em execução ou logout/login pendente.' '❌ Docker no está instalado, no está en ejecución o requiere cerrar sesión/iniciar sesión.')"
        all_deps_ok=false
    fi

    # Verificar Kind
    if command -v kind &> /dev/null; then
        echo "$(t '✅ Kind está instalado.' '✅ Kind está instalado.')"
    else
        echo "$(t '❌ Kind não está instalado.' '❌ Kind no está instalado.')"
        all_deps_ok=false
    fi

    # Verificar Kubectl
    if command -v kubectl &> /dev/null; then
        echo "$(t '✅ Kubectl está instalado.' '✅ Kubectl está instalado.')"
    else
        echo "$(t '❌ Kubectl não está instalado.' '❌ Kubectl no está instalado.')"
        all_deps_ok=false
    fi

    # Verificar Girus CLI e GLIBC
    if check_girus_in_path; then
        echo "$(t '✅ Girus CLI está instalado e disponível no PATH.' '✅ Girus CLI está instalado y disponible en el PATH.')"
    else
        echo "$(t '⚠️ Girus CLI não está disponível no PATH.' '⚠️ Girus CLI no está disponible en el PATH.')"
        all_deps_ok=false
    fi

    return $( [ "$all_deps_ok" = true ] && echo 0 || echo 1 )
}

# Iniciar mensagem principal
echo "$(t '=== Iniciando instalação do Girus CLI ===' '=== Iniciando instalación del Girus CLI ===')"

# Verificar e limpar instalações anteriores
check_previous_install

# Verificando pacotes necessarios para instalação
echo "$(t '=== Verificando pacotes necessários ===' '=== Verificando paquetes requeridos ===')"


CHECK=$(check_download_tool)
if [ "$CHECK" == "none" ]; then
    echo "$(t '❌ Erro: curl ou wget não encontrados. Por favor, instale um deles e tente novamente.' '❌ Error: curl o wget no encontrados. Instala uno de ellos y ejecuta este script nuevamente.')"
    exit 1
else
    echo "$(t '✅ $CHECK encontrado.' '✅ $CHECK encontrado.')"
fi

# Checa se bc está instalado para comparação numérica

if command -v bc &> /dev/null; then
    echo "$(t '✅ Pacote bc encontrado.' '✅ Paquete bc encontrado')"
else
    echo "$(t 'Comando bc não encontrado. Instalando:' 'Comando bc no encontrado. Instalando')"
    if [ "$OS" == "darwin" ]; then
        if command -v brew &> /dev/null; then
            brew install bc
        else
            echo "$(t '❌ Não foi possível instalar o pacote bc. Instale e execute novamente.' '❌ No se pudo instalar el paquete bc. Instálelo ejecuta este script nuevamente.')"
            exit 1
        fi
    elif [ "$OS" == "linux" ]; then
        case "$DISTRO" in
        "debian" | "ubuntu") sudo apt-get install -qq bc >/dev/null;;
        "rhel" | "fedora" | "rocky") sudo yum install bc -yq;;
        "cachyos") sudo pacman -S --noconfirm bc ;;
        *) echo "$(t '❌ Não foi possível instalar o pacote bc. Instale e execute novamente.' '❌ No se pudo instalar el paquete bc. Instálelo ejecuta este script nuevamente.')" && exit 1 ;;
        esac
    fi
fi   

# ETAPA 1: Verificar pré-requisitos - Docker
echo "$(t '=== ETAPA 1: Verificando Docker ===' '=== ETAPA 1: Verificando Docker ===')"
if ! command -v docker &> /dev/null; then
    echo "$(t 'Docker não está instalado.' 'Docker no está instalado.')"
    ask_user "$(t 'Deseja instalar Docker automaticamente? (Linux apenas) (S/n): ' '¿Desea instalar Docker automáticamente? (solo Linux) (S/n): ')" "S" "INSTALL_DOCKER"

    if [[ "$INSTALL_DOCKER" =~ ^[Ss]$ ]]; then
        install_docker
    else
        echo "$(t '⚠️ Aviso: Docker é necessário para criar clusters Kind e executar o Girus.' '⚠️ Aviso: Docker es necesario para crear clústeres Kind y ejecutar Girus.')"
        echo "$(t 'Por favor, instale o Docker adequado para seu sistema operacional:' 'Por favor, instala Docker adecuado para tu sistema operativo:')"
        echo " - Linux: https://docs.docker.com/engine/install/"
        echo " - macOS: https://docs.docker.com/desktop/install/mac-install/"
        echo " - Windows: https://docs.docker.com/desktop/install/windows-install/"
        exit 1
    fi
else
    # Verificar se o Docker está em execução
    if ! docker info &> /dev/null; then
        echo "$(t '⚠️ Aviso: Docker está instalado, mas não está em execução.' '⚠️ Aviso: Docker está instalado, pero no se está ejecutando.')"
        ask_user "$(t 'Deseja tentar iniciar o Docker? (S/n): ' '¿Desea intentar iniciar Docker? (S/n): ')" "S" "START_DOCKER"

        if [[ "$START_DOCKER" =~ ^[Ss]$ ]]; then
            echo "$(t 'Tentando iniciar o Docker...' 'Intentando iniciar Docker...')"
            if [ "$OS" == "linux" ]; then
                sudo systemctl start docker
                # Verificar novamente
                if ! docker info &> /dev/null; then
                    echo "$(t '❌ Falha ao iniciar o Docker. Por favor, inicie manualmente com '\''sudo systemctl start docker'\''' '❌ Error al iniciar Docker. Por favor, inícialo manualmente con '\''sudo systemctl start docker'\''')"
                    exit 1
                fi
            else
                echo "$(t 'No macOS/Windows, inicie o Docker Desktop manualmente e execute este script novamente.' 'En macOS/Windows, inicia Docker Desktop manualmente y ejecuta este script nuevamente.')"
                exit 1
            fi
        else
            echo "$(t '❌ Erro: Docker precisa estar em execução para usar o Girus. Por favor, inicie-o e tente novamente.' '❌ Error: Docker debe estar en ejecución para usar Girus. Por favor, inícialo e inténtalo nuevamente.')"
            exit 1
        fi
    fi
    echo "✅ Docker está instalado e em execução."
fi

# ETAPA 2: Verificar pré-requisitos - Kind
echo "=== ETAPA 2: Verificando Kind ==="
if ! command -v kind &> /dev/null; then
    echo "$(t 'Kind não está instalado.' 'Kind no está instalado.')"
    ask_user "$(t 'Deseja instalar Kind automaticamente? (S/n): ' '¿Desea instalar Kind automáticamente? (S/n): ')" "S" "INSTALL_KIND"

    if [[ "$INSTALL_KIND" =~ ^[Ss]$ ]]; then
        install_kind
    else
        echo "$(t '⚠️ Aviso: Kind é necessário para criar clusters Kubernetes e executar o Girus.' '⚠️ Aviso: Kind es necesario para crear clústeres Kubernetes y ejecutar Girus.')"
        echo "$(t 'Você pode instalá-lo manualmente seguindo as instruções em: https://kind.sigs.k8s.io/docs/user/quick-start/#installation' 'Puedes instalarlo manualmente siguiendo las instrucciones en: https://kind.sigs.k8s.io/docs/user/quick-start/#installation')"
        exit 1
    fi
else
    echo "$(t '✅ Kind já está instalado.' '✅ Kind ya está instalado.')"
fi

# ETAPA 3: Verificar pré-requisitos - Kubectl
echo "=== ETAPA 3: Verificando Kubectl ==="
if ! command -v kubectl &> /dev/null; then
    echo "$(t 'Kubectl não está instalado.' 'Kubectl no está instalado.')"
    ask_user "$(t 'Deseja instalar Kubectl automaticamente? (S/n): ' '¿Desea instalar Kubectl automáticamente? (S/n): ')" "S" "INSTALL_KUBECTL"

    if [[ "$INSTALL_KUBECTL" =~ ^[Ss]$ ]]; then
        install_kubectl
    else
        echo "$(t '⚠️ Aviso: Kubectl é necessário para interagir com o cluster Kubernetes.' '⚠️ Aviso: Kubectl es necesario para interactuar con el clúster de Kubernetes.')"
        echo "$(t 'Você pode instalá-lo manualmente seguindo as instruções em: https://kubernetes.io/docs/tasks/tools/install-kubectl/' 'Puedes instalarlo manualmente siguiendo las instrucciones en: https://kubernetes.io/docs/tasks/tools/install-kubectl/')"
        exit 1
    fi
else
    echo "$(t '✅ Kubectl já está instalado.' '✅ Kubectl ya está instalado.')"
fi

# ETAPA 4: Baixar e instalar o Girus CLI
echo "$(t '=== ETAPA 4: Instalando Girus CLI ===' '=== ETAPA 4: Instalando Girus CLI ===')"
download_and_install

# Voltar para o diretório original
cd "$ORIGINAL_DIR"

# Mensagem final de conclusão
echo ""
echo "$(t '===== INSTALAÇÃO CONCLUÍDA =====' '===== INSTALACIÓN COMPLETADA =====')"
echo ""

# Verificar todas as dependências
verify_all_dependencies
echo ""

# Exibir instruções para próximos passos
cat << EOF
$(t '📝 PRÓXIMOS PASSOS:' '📝 PRÓXIMOS PASOS:')

1. $(t 'Para criar um novo cluster Kubernetes e instalar o Girus:' 'Para crear un nuevo clúster de Kubernetes e instalar Girus:')
   $ girus create cluster

2. $(t 'Após a criação do cluster, acesse o Girus no navegador:' 'Después de crear el clúster, accede a Girus en el navegador:')
   http://localhost:8000

3. $(t 'No navegador, inicie o laboratório Linux de boas-vindas para conhecer' 'En el navegador, inicia el laboratorio de bienvenida de Linux para conocer')
   $(t '   a plataforma e começar sua experiência com o Girus!' '   la plataforma y comenzar tu experiencia con Girus!')

$(t 'Obrigado por instalar o Girus CLI!' '¡Gracias por instalar Girus CLI!')
EOF

exit 0 