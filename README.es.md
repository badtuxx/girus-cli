![GIRUS](girus-logo.png)

**Elige tu idioma / Escolha seu idioma:** [Portugués](README.md) | [Español](README.es.md)

# GIRUS: Plataforma de Laboratorios Interactivos

Versión 0.5.0 Codename: "Maracatu" - Mayo de 2025

## Visión General

GIRUS es una plataforma open-source de laboratorios interactivos que permite la creación, gestión y ejecución de entornos de aprendizaje práctico para tecnologías como Linux, Docker, Kubernetes, Terraform y otras herramientas esenciales para profesionales de DevOps, SRE, Desarrollo y Platform Engineering.

Desarrollada por LINUXtips, GIRUS se diferencia por ejecutarse localmente en la máquina del usuario, eliminando la necesidad de infraestructura en la nube o configuraciones complejas. A través de una CLI intuitiva, los usuarios pueden crear rápidamente entornos aislados y seguros donde practicar y perfeccionar sus habilidades técnicas.

## Principales Características

- **Ejecución Local**: A diferencia de plataformas como Katacoda o Instruqt que funcionan como SaaS, GIRUS se ejecuta directamente en la máquina del usuario mediante contenedores Docker y Kubernetes. Lo mejor de todo: el proyecto es open source y gratuito.
- **Entornos Aislados**: Cada laboratorio se ejecuta en un entorno aislado en Kubernetes, garantizando seguridad y evitando conflictos con el sistema host.
- **Interfaz Intuitiva**: Terminal interactivo con tareas guiadas y validación automática del progreso.
- **Instalación Fácil**: CLI simple que gestiona todo el ciclo de vida de la plataforma (creación, ejecución y eliminación).
- **Actualización Sencilla**: Comando `update` integrado que verifica, descarga e instala nuevas versiones automáticamente.
- **Laboratorios Personalizables**: Sistema de plantillas basado en ConfigMaps de Kubernetes que facilita la creación de nuevos laboratorios.
- **Open Source**: Proyecto completamente abierto a contribuciones de la comunidad.
- **Multilingüe**: Además del portugués, GIRUS ahora ofrece soporte oficial para español. El sistema de plantillas permite agregar fácilmente otros idiomas.

## Gestión de Repositorios y Laboratorios

GIRUS implementa un sistema robusto de gestión de repositorios y laboratorios, similar a Helm para Kubernetes. Este sistema permite:

### Actualizar la CLI

- **Verificar y Actualizar a la Última Versión**:
  ```bash
  girus update
  ```
  Este comando comprueba si hay una versión más reciente del GIRUS CLI disponible, la descarga e instala, ofreciendo la opción de recrear el cluster tras la actualización para garantizar compatibilidad.

### Repositorios

- **Agregar Repositorios**:
  ```bash
  girus repo add linuxtips https://github.com/linuxtips/labs/raw/main
  ```
- **Listar Repositorios**:
  ```bash
  girus repo list
  ```
- **Eliminar Repositorios**:
  ```bash
  girus repo remove linuxtips
  ```
- **Actualizar Repositorios**:
  ```bash
  girus repo update linuxtips https://github.com/linuxtips/labs/raw/main
  ```

### Soporte para Repositorios Locales (file://)

GIRUS también admite repositorios locales usando el prefijo `file://`. Esto es útil para probar laboratorios o desarrollar repositorios sin necesidad de publicarlos en un servidor remoto.

#### Ejemplo de uso:

```bash
# Agregando un repositorio local
./girus repo add mi-local file:///ruta/absoluta/a/tu-repo
```

## Laboratorios

- **Listar Laboratorios Disponibles**:
  ```bash
  girus lab list
  ```
- **Instalar Laboratorio**:
  ```bash
  girus lab install linuxtips linux-basics
  ```
- **Buscar Laboratorios**:
  ```bash
  girus lab search docker
  ```

## Instalación

### Usando el script de instalación

```bash
curl -sSL girus.linuxtips.io | bash
```

### Usando el Makefile

Clona el repositorio y ejecuta `make <comando>`.

### Compilación y Instalación

> **Requisito:** Go **1.26** o superior. La versión mínima está declarada en `go.mod` y proviene de las dependencias de `k8s.io` (0.37.0), que exigen Go 1.26. Los workflows de CI y el `Dockerfile` usan esa misma versión.

* **`make build`** (o simplemente `make`): Compila el binario `girus` para tu sistema operativo actual y lo coloca en el directorio `dist/`.
* **`make install`**: Compila el binario (si aún no está compilado) y lo mueve a `/usr/local/bin/girus`, requiriendo permisos de superusuario (`sudo`).
* **`make clean`**: Elimina el directorio `dist/` y todos los archivos generados de build.
* **`make release`**: Compila el binario `girus` para múltiples plataformas (Linux, macOS, Windows - amd64 y arm64) y los coloca en `dist/`.

### Versionamiento

GIRUS CLI utiliza versionamiento dinámico basado en etiquetas git. Puedes verificar la versión actual ejecutando:

```bash
./girus version
```

> **Importante:** usa siempre `make build` en lugar de un `go build` sin flags. El `make` inyecta la versión mediante `ldflags`; sin eso el binario reporta `dev`, y `girus create cluster` concluye que hay una actualización disponible y abre un prompt de auto-actualización antes de crear el cluster.

### Pruebas

El proyecto tiene dos capas de pruebas, con propósitos distintos.

#### Pruebas unitarias

```bash
go test ./...                                              # suite completa
go test -run TestListAndGetManifests ./internal/templates  # una prueba específica
```

Son rápidas y no dependen de Docker. Hoy cubren solo la validación de los manifiestos embebidos (`internal/templates`) — todo el código que habla con el cluster (`internal/k8s`) no es ejercitado por ellas. En la práctica, `go test ./...` pasa incluso si `girus create cluster` está roto.

#### Prueba end-to-end

```bash
make e2e
```

Levanta un cluster Kind real, despliega GIRUS, valida el resultado y elimina el cluster al final. Es la única prueba que ejercita el camino completo de la CLI.

**Prerrequisitos:** Docker en ejecución, `kind`, `kubectl`, `make` y `go`. El script los verifica todos antes de empezar y aborta con un mensaje claro si falta alguno.

Lo que valida, en siete etapas:

1. Prerrequisitos instalados y daemon de Docker respondiendo
2. Compilación del binario con la versión correcta
3. `girus create cluster` termina sin error
4. Los deployments `girus-backend` y `girus-frontend` quedan disponibles
5. ConfigMaps de laboratorio aplicados **y** efectivamente cargados por el backend
6. `girus list labs` y `girus status` responden
7. Resultado consolidado

La etapa 5 merece atención: compara cuántos ConfigMaps se aplicaron con cuántas plantillas logró deserializar el backend. Una divergencia de esquema falla en silencio — el ConfigMap existe en el cluster, el backend descarta el laboratorio, y nada aparece en la interfaz. La prueba convierte eso en un aviso explícito.

**Variables de entorno:**

| Variable | Efecto |
| --- | --- |
| `E2E_KEEP=1` | Mantiene el cluster levantado al final, para inspección |
| `E2E_FORCE=1` | Elimina un cluster `girus` preexistente antes de empezar |

Por seguridad, la prueba **se niega** a ejecutarse si ya existe un cluster llamado `girus` (para no destruir un entorno en uso) y elimina el cluster que creó incluso si falla.

Salida esperada de un ciclo limpio:

```
[5/7] Validando os templates de laboratorio
  OK: 26 ConfigMaps de laboratorio aplicados
  OK: backend carregou 26 templates
[6/7] Exercitando os comandos do CLI
  OK: 'girus list labs' respondeu
  OK: 'girus status' respondeu

E2E PASSOU -- 26 laboratorios aplicados, 26 carregados pelo backend
```

El ciclo completo tarda unos 2 minutos con las imágenes en caché, y entre 5 y 10 minutos en la primera ejecución.

> **Recomendado antes de publicar una release.** La suite unitaria no cubre `internal/k8s`, así que `make e2e` es lo que realmente comprueba que el binario a publicar crea un cluster funcional.

### Gestión de Dependencias (Go Modules)

* **`make check-updates`**: Verifica si hay actualizaciones disponibles para las dependencias Go del proyecto.
* **`make upgrade-all`**: Actualiza todas las dependencias Go a sus versiones más recientes y ejecuta `go mod tidy`.
* **`make upgrade MODULE=<nombre/del/modulo>`**: Actualiza una dependencia Go específica (ej.: `make upgrade MODULE=github.com/spf13/cobra`).
* **`make tidy`**: Ejecuta `go mod tidy` para limpiar `go.mod` y `go.sum`.
* **`make deps`**: Muestra el grafo de dependencias del proyecto.

## Contribuyendo con Labs

1. Crea un nuevo directorio en `labs/<nombre-del-lab>`.
2. Agrega un archivo `lab.yaml` con la estructura del lab.
3. Actualiza `index.yaml` con la información del nuevo lab.
4. Envía un Pull Request.

## Soporte y Contacto

* **GitHub Issues**: [github.com/badtuxx/girus-cli/issues](https://github.com/badtuxx/girus-cli/issues)
* **GitHub Discussions**: [github.com/badtuxx/girus-cli/discussions](https://github.com/badtuxx/girus-cli/discussions)
* **Discord de la Comunidad**: [discord.gg/linuxtips](https://discord.gg/linuxtips)

## Licencia

Este proyecto se distribuye bajo la licencia GPL-3.0. Consulta el archivo [LICENSE](LICENSE) para más detalles.
