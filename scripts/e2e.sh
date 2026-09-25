#!/usr/bin/env bash
#
# Teste end-to-end do girus-cli.
#
# Sobe um cluster Kind de verdade, implanta o Girus, valida que o backend e o
# frontend ficaram de pe e que os templates de laboratorio foram carregados,
# e no final remove o cluster.
#
# Existe porque a suite unitaria nao cobre nada disso: hoje so
# internal/templates tem testes, e todo o internal/k8s (~630 linhas que falam
# com o cluster) nunca e executado pelo CI. `go test ./...` passa mesmo que
# `girus create cluster` esteja completamente quebrado.
#
# Uso:
#   make e2e                 # ciclo completo, remove o cluster no final
#   E2E_KEEP=1 make e2e      # mantem o cluster de pe para inspecao
#   E2E_FORCE=1 make e2e     # remove um cluster girus preexistente antes
#
set -euo pipefail

CLUSTER_NAME="girus"
NAMESPACE="girus"
BIN="./dist/girus"
KEEP="${E2E_KEEP:-0}"
FORCE="${E2E_FORCE:-0}"

# Cores apenas quando a saida for um terminal.
if [ -t 1 ]; then
  VERM=$'\033[31m'; VERD=$'\033[32m'; AMAR=$'\033[33m'; CIAN=$'\033[36m'; NEG=$'\033[0m'
else
  VERM=''; VERD=''; AMAR=''; CIAN=''; NEG=''
fi

ETAPA=0
TOTAL=7
FALHAS=0

info()  { printf '%s\n' "${CIAN}$*${NEG}"; }
ok()    { printf '  %s %s\n' "${VERD}OK:${NEG}" "$*"; }
aviso() { printf '  %s %s\n' "${AMAR}AVISO:${NEG}" "$*"; }
erro()  { printf '  %s %s\n' "${VERM}ERRO:${NEG}" "$*"; FALHAS=$((FALHAS + 1)); }

etapa() {
  ETAPA=$((ETAPA + 1))
  printf '\n%s\n' "${CIAN}[${ETAPA}/${TOTAL}] $*${NEG}"
}

# Remove o cluster ao sair, inclusive em caso de falha, a menos que E2E_KEEP=1.
limpar() {
  local codigo=$?
  if [ "$KEEP" = "1" ]; then
    printf '\n%s\n' "${AMAR}E2E_KEEP=1: cluster '${CLUSTER_NAME}' mantido de pe.${NEG}"
    printf 'Para remover: kind delete cluster --name %s\n' "$CLUSTER_NAME"
    return $codigo
  fi
  if kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
    printf '\n%s\n' "${CIAN}Removendo o cluster...${NEG}"
    "$BIN" delete cluster --force >/dev/null 2>&1 || \
      kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
  fi
  return $codigo
}
trap limpar EXIT

# ---------------------------------------------------------------------------
etapa "Verificando pre-requisitos"

for cmd in docker kind kubectl make go; do
  command -v "$cmd" >/dev/null || { erro "$cmd nao encontrado no PATH"; exit 1; }
done
ok "docker, kind, kubectl, make e go presentes"

docker info >/dev/null 2>&1 || {
  erro "o daemon do Docker nao esta respondendo"
  echo "      Se o Docker esta rodando, verifique se seu usuario pertence ao grupo 'docker'."
  exit 1
}
ok "daemon do Docker respondendo"

if kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
  if [ "$FORCE" = "1" ]; then
    aviso "cluster '${CLUSTER_NAME}' ja existe, removendo (E2E_FORCE=1)"
    kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1
  else
    erro "ja existe um cluster '${CLUSTER_NAME}'"
    echo "      Use E2E_FORCE=1 make e2e para remove-lo, ou remova manualmente:"
    echo "      kind delete cluster --name ${CLUSTER_NAME}"
    exit 1
  fi
fi
ok "nenhum cluster '${CLUSTER_NAME}' preexistente"

# ---------------------------------------------------------------------------
etapa "Compilando o binario"

# Obrigatorio usar 'make build', nao 'go build'.
#
# O make injeta a versao via ldflags. Sem isso common.Version fica vazio,
# IsNewerVersion() interpreta como 0.0.0, e 'create cluster' conclui que ha
# atualizacao disponivel e abre um prompt. Com stdin fechado o ReadString
# devolve "", que cmd/create.go:78 trata como SIM -- o teste tentaria
# atualizar o proprio binario e sairia com os.Exit(0), passando sem testar nada.
make build >/dev/null
[ -x "$BIN" ] || { erro "binario nao encontrado em ${BIN}"; exit 1; }

VERSAO=$("$BIN" version | head -1)
ok "${VERSAO}"

case "$VERSAO" in
  *dev*) aviso "versao 'dev': o prompt de auto-atualizacao pode disparar" ;;
esac

# ---------------------------------------------------------------------------
etapa "Criando o cluster e implantando o Girus"

INICIO=$(date +%s)
if ! "$BIN" create cluster --skip-port-forward --skip-browser </dev/null; then
  erro "'girus create cluster' falhou"
  exit 1
fi
ok "cluster criado em $(( $(date +%s) - INICIO ))s"

# ---------------------------------------------------------------------------
etapa "Aguardando os deployments ficarem disponiveis"

for dep in girus-backend girus-frontend; do
  if kubectl wait --for=condition=available --timeout=300s \
       "deployment/${dep}" -n "$NAMESPACE" >/dev/null 2>&1; then
    ok "deployment/${dep} disponivel"
  else
    erro "deployment/${dep} nao ficou disponivel em 300s"
    kubectl get pods -n "$NAMESPACE" -o wide || true
    kubectl logs -n "$NAMESPACE" "deployment/${dep}" --tail=30 || true
  fi
done

# ---------------------------------------------------------------------------
etapa "Validando os templates de laboratorio"

# Os manifests embutidos sao aplicados como ConfigMaps rotulados
# 'app: girus-lab-template'. Se a contagem for zero, o embed ou o apply quebrou.
CMS=$(kubectl get configmaps -n "$NAMESPACE" \
        -l app=girus-lab-template --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "${CMS:-0}" -gt 0 ]; then
  ok "${CMS} ConfigMaps de laboratorio aplicados"
else
  erro "nenhum ConfigMap com label 'app=girus-lab-template' encontrado"
fi

# O backend precisa conseguir desserializar os templates, nao so recebe-los.
# Divergencia de schema falha em silencio: o ConfigMap existe, o backend
# descarta o lab, e nada aparece na UI (ver issues #209 e #210).
#
# A resposta e {"templates":[{"name":...,"title":...,"description":...}]}.
# Contar ocorrencias de '"name"' nao funciona: a chave tambem aparece dentro
# do texto das descricoes e infla o total.
RESPOSTA=$(kubectl exec -n "$NAMESPACE" deploy/girus-backend -- \
             wget -q -O- http://localhost:8080/api/v1/templates 2>/dev/null || true)

if command -v python3 >/dev/null 2>&1; then
  CARREGADOS=$(printf '%s' "$RESPOSTA" | python3 -c \
    'import sys,json; print(len(json.load(sys.stdin).get("templates",[])))' 2>/dev/null || echo 0)
else
  # Sem python3: "duration" e um campo folha, nao aparece em prosa.
  CARREGADOS=$(printf '%s' "$RESPOSTA" | grep -o '"duration"' | wc -l | tr -d ' ')
fi
if [ "${CARREGADOS:-0}" -gt 0 ]; then
  ok "backend carregou ${CARREGADOS} templates"
  if [ "${CARREGADOS:-0}" -lt "${CMS:-0}" ]; then
    aviso "${CMS} ConfigMaps aplicados, mas so ${CARREGADOS} carregados"
    echo "         Provavel divergencia de schema. Ver: kubectl logs -n ${NAMESPACE} deploy/girus-backend"
  fi
else
  erro "o backend nao carregou nenhum template"
fi

# ---------------------------------------------------------------------------
etapa "Exercitando os comandos do CLI"

if "$BIN" list labs </dev/null >/dev/null 2>&1; then
  ok "'girus list labs' respondeu"
else
  erro "'girus list labs' falhou"
fi

if "$BIN" status </dev/null >/dev/null 2>&1; then
  ok "'girus status' respondeu"
else
  erro "'girus status' falhou"
fi

# 'girus stop' seguido de 'girus start' fica de fora de proposito: start nao
# restabelece o port-forward (issues #202 e #208), entao o passo falharia
# sempre. Incluir como teste de regressao quando a correcao entrar.

# ---------------------------------------------------------------------------
etapa "Resultado"

if [ "$FALHAS" -eq 0 ]; then
  printf '\n%s\n' "${VERD}E2E PASSOU${NEG} -- ${CMS} laboratorios aplicados, ${CARREGADOS} carregados pelo backend"
  exit 0
fi

printf '\n%s\n' "${VERM}E2E FALHOU${NEG} -- ${FALHAS} verificacao(oes) com erro"
echo "Para investigar, rode de novo com E2E_KEEP=1 e inspecione o cluster."
exit 1
