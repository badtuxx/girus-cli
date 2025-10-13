![GIRUS](girus-logo.png)

**Elige tu idioma / Escolha seu idioma:** [Portugués](README.md) | [Español](README.es.md)

# GIRUS: Plataforma de Laboratorios Interactivos

Versión 0.3.0 Codename: "Maracatu" - Mayo de 2025

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

* **`make build`** (o simplemente `make`): Compila el binario `girus` para tu sistema operativo actual y lo coloca en el directorio `dist/`.
* **`make install`**: Compila el binario (si aún no está compilado) y lo mueve a `/usr/local/bin/girus`, requiriendo permisos de superusuario (`sudo`).
* **`make clean`**: Elimina el directorio `dist/` y todos los archivos generados de build.
* **`make release`**: Compila el binario `girus` para múltiples plataformas (Linux, macOS, Windows - amd64 y arm64) y los coloca en `dist/`.

### Versionamiento

GIRUS CLI utiliza versionamiento dinámico basado en etiquetas git. Puedes verificar la versión actual ejecutando:

```bash
./girus version
```

## Contribuyendo con Labs

1. Crea un nuevo directorio en `labs/<nombre-del-lab>`.
2. Agrega un archivo `lab.yaml` con la estructura del lab.
3. Actualiza `index.yaml` con la información del nuevo lab.
4. Envía un Pull Request.

## Contribución y comunidad

GIRUS es un proyecto de código abierto que depende de las contribuciones de la comunidad para crecer. Hay varias maneras de contribuir:

### Desarrollo
- Corrección de bugs y mejoras del código fuente
- Implementación de nuevas funcionalidades
- Optimización del rendimiento
- Pruebas y garantía de calidad

### Creación de contenido
- Desarrollo de nuevos templates de laboratorio
- Traducción de contenidos a otros idiomas
- Elaboración de tutoriales y documentación

### Divulgación
- Compartiendo el proyecto en las redes sociales
- Presentaciones en eventos y congresos
- Artículos sobre el uso y beneficios de la plataforma

### Proceso de contribución
1. Fork el repositorio en [GitHub](https://github.com/badtuxx/girus)
2. Crea una branch para tu feature (`git checkout -b feature/nueva-funcionalidad`)
3. Realice sus cambios y commit (`git commit -m 'Añade nueva funcionalidad'`)
4. Push a la branch (`git push origin feature/nueva-funcionalidad`)
5. Crear un Pull Request

## Soporte y Contacto

* **GitHub Issues**: [github.com/badtuxx/girus-cli/issues](https://github.com/badtuxx/girus-cli/issues)
* **GitHub Discussions**: [github.com/badtuxx/girus-cli/discussions](https://github.com/badtuxx/girus-cli/discussions)
* **Discord de la Comunidad**: [discord.gg/linuxtips](https://discord.gg/linuxtips)

## Licencia

Este proyecto se distribuye bajo la licencia GPL-3.0. Consulta el archivo [LICENSE](LICENSE) para más detalles.
