# NOME DO BINÁRIO
BIN=girus
INSTALL_DIR=/usr/local/bin
# DIRETÓRIO DE BUILD
BD=dist
# DIRETÓRIO ATUAL
CDR=.
CONFIG_PATH=manifest/config.yaml
# Variáveis para versionamento
VERSION    := 0.5.0
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILT_BY   := $(shell whoami)
COMMITID   := $(shell git rev-parse --short HEAD)
GO_VERSION := $(shell go version | cut -d' ' -f3)
GO_OS 	   := $(shell go env GOOS)
GO_ARCH    := $(shell go env GOARCH)

LDFLAGS := -X github.com/badtuxx/girus-cli/internal/common.Version=$(VERSION)
LDFLAGS += -X github.com/badtuxx/girus-cli/internal/common.BuildDate=$(DATE)
LDFLAGS += -X github.com/badtuxx/girus-cli/internal/common.BuildUser=$(BUILT_BY)
LDFLAGS += -X github.com/badtuxx/girus-cli/internal/common.CommitID=$(COMMITID)
LDFLAGS += -X github.com/badtuxx/girus-cli/internal/common.GoVersion=$(GO_VERSION)
LDFLAGS += -X github.com/badtuxx/girus-cli/internal/common.GoOS=$(GO_OS)
LDFLAGS += -X github.com/badtuxx/girus-cli/internal/common.GoArch=$(GO_ARCH)

# PLATAFORMAS SUPORTADAS
PLATAFORMAS = \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

# Alvo padrão: construir o binário
all: build

# Constrói o binário principal
build:
	go build -v -ldflags="$(LDFLAGS)" -o $(BD)/$(BIN) $(CDR)/main.go

# Executa os tests
tests:
	go test -v ./...

# Instala o binário no sistema
install: build
	sudo mv $(BD)/$(BIN) $(INSTALL_DIR)/$(BIN)

# Limpa o diretório de build
clean:
	rm -rf $(BD)

# Cria os binários para múltiplas plataformas (release)
release:
	@echo "Construindo para múltiplas plataformas..."
	@mkdir -p $(BD)
	@for platform in $(PLATAFORMAS); do \
		OS=$$(echo $$platform | cut -d/ -f1); \
		ARCH=$$(echo $$platform | cut -d/ -f2); \
		OUT=$(BD)/$(BIN)-$$OS-$$ARCH; \
		if [ "$$OS" = "windows" ]; then \
			OUT=$$OUT.exe; \
		fi; \
		echo "-> $$OS/$$ARCH (Saída: $$OUT)"; \
		GOOS=$$OS GOARCH=$$ARCH go build -ldflags "$(LDFLAGS)" -v -o $$OUT $(CDR)/main.go || exit 1; \
	done

# Executa o binário localmente usando o arquivo de configuração
run-local: build
	CONFIG_FILE=$(CONFIG_PATH) ./$(BD)/$(BIN)

# Verifica se há atualizações de dependências disponíveis
check-updates:
	@echo "Verificando atualizações disponíveis..."
	go list -u -m -json all | grep '"Path"\|"Version"\|"Update"'

# Atualiza todas as dependências do projeto
upgrade-all:
	@echo "Atualizando todas as dependências..."
	go get -u ./...
	go mod tidy
	@echo "Todas as dependências foram atualizadas."

# Atualiza uma dependência específica (requer MODULE=nome/do/modulo)
upgrade:
ifndef MODULE
	$(error Você deve fornecer o nome do módulo com MODULE=exemplo.com/lib)
endif
	@echo "Atualizando $(MODULE)..."
	go get -u $(MODULE)
	go mod tidy
	@echo "$(MODULE) atualizado."

# Limpa dependências não utilizadas (go mod tidy)
tidy:
	@echo "Limpando dependências não utilizadas..."
	go mod tidy
	@echo "go.mod e go.sum estão limpos."

# Exibe o gráfico de dependências
deps:
	@echo "Exibindo gráfico de dependências..."
	go mod graph

# Declara alvos que não representam arquivos
.PHONY: all build install clean release run-local check-updates upgrade-all upgrade tidy deps
