![GIRUS](girus-logo.png)

**Choose your language / Escolha seu idioma / Elija su idioma:** [Português](README.md) | [English](README.en.md) | [Español](README.es.md)

# GIRUS: Interactive Labs Platform

Version 0.5.0 Codename: "Maracatu" - May 2025

## Overview

GIRUS is an open-source interactive labs platform that enables the creation, management, and execution of hands-on learning environments for technologies such as Linux, Docker, Kubernetes, Terraform, and other essential tools for DevOps, SRE, Dev, and Platform Engineering professionals.

Developed by LINUXtips, the GIRUS platform stands out by running locally on the user's machine, eliminating the need for cloud infrastructure or complex configurations. Through an intuitive CLI, users can quickly create isolated and secure environments where they can practice and improve their technical skills.

## Key Features

- **Local Execution**: Unlike other platforms like Katacoda or Instruqt that operate as SaaS, GIRUS runs directly on the user's machine through Docker and Kubernetes containers, and best of all, the project is open source and free.
- **Isolated Environments**: Each lab runs in an isolated environment on Kubernetes, ensuring security and avoiding conflicts with the host system
- **Intuitive Interface**: Interactive terminal with guided tasks and automatic progress validation
- **Easy Installation**: Simple CLI that manages the entire platform lifecycle (creation, execution, and deletion)
- **Simplified Updates**: Built-in `update` command that checks, downloads, and installs new versions automatically
- **Customizable Labs**: Template system based on Kubernetes ConfigMaps that facilitates the creation of new labs
- **Open Source**: Project completely open to community contributions
- **Multilingual**: In addition to Portuguese, GIRUS now offers official Spanish support. The template system allows easy addition of new languages.

## Repository and Labs Management

GIRUS implements a robust repository and labs management system, similar to Helm for Kubernetes. This system allows:

### Installation
```bash
curl -sSL girus.linuxtips.io | bash
```

You need to have Docker installed on your computer to install Girus.

### CLI Update

- **Check and Update to the Latest Version**:
  ```bash
  girus update
  ```
  This command checks if a newer version of GIRUS CLI is available, downloads and installs the update, offering the option to recreate the cluster after the update to ensure compatibility.

### Repositories

- **Add Repositories**: 
  ```bash
  girus repo add linuxtips https://github.com/linuxtips/labs/raw/main
  ```

- **List Repositories**:
  ```bash
  girus repo list
  ```

- **Remove Repositories**:
  ```bash
  girus repo remove linuxtips
  ```

- **Update Repositories**:
  ```bash
  girus repo update linuxtips https://github.com/linuxtips/labs/raw/main
  ```

### Local Repository Support (file://)

GIRUS now supports local repositories using the `file://` prefix. This is useful for testing labs or developing repositories without needing to publish to a remote server.

#### Example usage:

```bash
# Adding a local repository
./girus repo add my-local file:///absolute/path/to/your-repo

# Practical example:
./girus repo add test-repo file:///home/jeferson/REPOS/teste/girus-cli/test-repo
```

> **Note:** The path after `file://` must be absolute and point to the directory where the repository's `index.yaml` is located.

You can list, search, and install labs normally from local repositories, just as you would with remote repositories.

### Labs

- **List Available Labs**:
  ```bash
  girus lab list
  ```

- **Install Lab**:
  ```bash
  girus lab install linuxtips linux-basics
  ```

- **Search Labs**:
  ```bash
  girus lab search docker
  ```

### Repository Structure

Repositories follow a standardized structure:

```
repository/
├── index.yaml           # Repository index
└── labs/               # Directory containing the labs
    ├── lab1/
    │   ├── lab.yaml    # Lab definition
    │   └── assets/     # Lab resources (optional)
    └── lab2/
        ├── lab.yaml
        └── assets/
```

### File Formats

#### index.yaml
```yaml
apiVersion: v1
generated: "2024-03-20T10:00:00Z"
entries:
  lab-name:
    - name: lab-name
      version: "1.0.0"
      description: "Lab description"
      keywords:
        - keyword1
        - keyword2
      maintainers:
        - "Name <email@example.com>"
      url: "https://github.com/your-repo/raw/main/labs/lab-name/lab.yaml"
      created: "2024-03-20T10:00:00Z"
      digest: "sha256:file-hash"
```

#### lab.yaml
```yaml
apiVersion: girus.linuxtips.io/v1
kind: Lab
metadata:
  name: lab-name
  version: "1.0.0"
  description: "Lab description"
  author: "Author Name"
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
    - name: "Task Name"
      description: "Task description"
      steps:
        - description: "Step description"
          command: "command"
          expectedOutput: "expected output"
          hint: "User hint"

  validation:
    - name: "Validation Name"
      description: "Validation description"
      checks:
        - command: "command"
          expectedOutput: "expected output"
          errorMessage: "Error message"
```

## Architecture

The GIRUS project consists of four main components:

1. **GIRUS CLI**: Command-line tool that manages the entire platform lifecycle
2. **Backend**: Golang API that orchestrates the labs through the Kubernetes API
3. **Frontend**: React web interface that provides access to the interactive terminal and tasks
4. **Lab Templates**: YAML definitions for the different available labs

### Architecture Flow Diagram

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  GIRUS CLI  │────▶│ Kind Cluster │────▶│ Kubernetes   │
└─────────────┘     └──────────────┘     └──────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Terminal   │◀───▶│   Frontend   │◀───▶│   Backend    │
│ Interactive │     │    (React)   │     │     (Go)     │
└─────────────┘     └──────────────┘     └──────────────┘
                                               │
                                               ▼
                                         ┌──────────────┐
                                         │  Templates   │
                                         │     Labs     │
                                         └──────────────┘
```

## Detailed Components

### GIRUS CLI

GIRUS (GIRUS Is Really Useful System) is a CLI tool developed by LINUXtips to create and manage practical lab environments.

## Installation

### Using the installation script

```bash
curl -sSL girus.linuxtips.io | bash
```

### Using the Makefile

Clone the repository and run `make <command>`.

Here are the available commands:

### Build and Installation

* **`make build`** (or simply `make`): Compiles the `girus` binary for your current operating system and places it in the `dist/` directory. This is the default command if you run `make` without arguments.
* **`make install`**: Compiles the binary (if not already compiled) and moves it to `/usr/local/bin/girus`, making it globally accessible on your system. Requires superuser permissions (`sudo`).
* **`make clean`**: Removes the `dist/` directory and all generated build files.
* **`make release`**: Compiles the `girus` binary for multiple platforms (Linux, macOS, Windows - amd64 and arm64) and places them in the `dist/` directory.

### Versioning

The GIRUS CLI uses a dynamic versioning system based on git tags. The build process automatically detects the version based on the following criteria:

* If a git tag exists (e.g., `v0.3.0`), that version will be used after removing the `v` prefix (result: `0.3.0`)
* If no tags exist, the default version `0.3.0` will be used
* For local builds, you can compile with a specific version using the following command:

```bash
go build -o girus -ldflags="-X 'github.com/badtuxx/girus-cli/internal/common.Version=0.5.0'" ./main.go
```

To check the current binary version, run:

```bash
./girus version
```

The project's CI/CD workflows also use this dynamic versioning mechanism for Docker builds and release artifacts, ensuring consistency throughout the build process.

### Dependency Management (Go Modules)

* **`make check-updates`**: Checks if there are updates available for the project's Go dependencies.
* **`make upgrade-all`**: Updates all Go dependencies to their latest versions and runs `go mod tidy`.
* **`make upgrade MODULE=<module/name>`**: Updates a specific Go dependency to the latest version. Replace `<module/name>` with the module path (e.g., `make upgrade MODULE=github.com/spf13/cobra`).
* **`make tidy`**: Runs `go mod tidy` to remove unused dependencies and clean up the `go.mod` and `go.sum` files.
* **`make deps`**: Displays the project's dependency graph.

### Getting Started
## Creating Your First Cluster
After installing GIRUS, the first step is to create a local Kubernetes cluster using Kind:

  ```bash
  girus create cluster
  ```
## This command will:

- **Check if Docker is running**
- **Install Kind (if not present)**
- **Create a local Kubernetes cluster**
- **Install GIRUS components (backend, frontend, etc.)**
- **Configure necessary services**

> **Note:** The process may take a few minutes on the first run, as it needs to download the necessary Docker images.

- **Checking Cluster Status**:
To check if the cluster was created successfully:
  ```bash
  # Check available Kind clusters
  kind get clusters

  # Check GIRUS pods
  kubectl get pods -n girus

  # Check GIRUS services
  kubectl get services -n girus
  ```

**Managing the Cluster**:
To check status:
  ```bash
  # List available clusters
  kind get clusters

  # Check if the cluster is healthy
  kubectl cluster-info
  ```

 **Delete the Cluster**:

  ```bash
  # Remove the cluster when no longer needed
  kind delete cluster --name girus
  ```
**Recreate the Cluster**:

  ```bash
  # If you need to recreate the cluster
  kind delete cluster --name girus
  girus create cluster
  ```

## Labs Repository

This repository contains a collection of hands-on labs for different technologies, organized in the following categories:

### AWS Labs
- AWS LocalStack with Terraform
- AWS S3 Storage
- AWS DynamoDB NoSQL
- AWS Lambda Serverless

### Terraform Labs
- Terraform Fundamentals
- Terraform with AWS
- Provisioners and Modules in Terraform

### Kubernetes Labs
- Kubernetes Fundamentals
- Deployment on Kubernetes
- Resource Exploration
- Services and Networking
- ConfigMaps and Secrets
- CronJobs

### Docker Labs
- Docker Fundamentals
- Container Management
- Networking Fundamentals
- Volumes
- Docker Compose

### Linux Labs
- Basic Commands
- User Management
- File Permissions
- Text Processing
- Process Management
- Shell Scripting
- System Monitoring

## Using the Labs

### Add the Repository

```bash
# Add the official repository
girus repo add girus-cli https://raw.githubusercontent.com/badtuxx/girus-cli/main/index.yaml

# Or add locally for development
girus repo add girus-cli file:///path/to/girus-cli
```

### List Available Labs

```bash
girus lab list
```

### Start a Lab

```bash
girus lab start <lab-name>
```

For example:
```bash
girus lab start aws_localstack_terraform
```

## Contributing Labs

To contribute new labs, follow these steps:

1. Create a new directory in `labs/<lab-name>`
2. Add a `lab.yaml` file with the lab structure
3. Update the `index.yaml` with the new lab information
4. Submit a Pull Request

### Lab Structure

```yaml
name: lab-name
title: "Lab Title"
description: "Detailed lab description"
duration: 45m
image: "ubuntu:20.04"
tasks:
  - name: "Task Name"
    description: "Task description"
    steps:
      - "Step 1: Do this"
      - "Step 2: Execute that"
    validation:
      - command: "command to verify"
        expectedOutput: "expected output"
        errorMessage: "Custom error message"
```

## Support and Contact

* **GitHub Issues**: [github.com/badtuxx/girus-cli/issues](https://github.com/badtuxx/girus-cli/issues)
* **GitHub Discussions**: [github.com/badtuxx/girus-cli/discussions](https://github.com/badtuxx/girus-cli/discussions)
* **Community Discord**: [discord.gg/linuxtips](https://discord.gg/linuxtips)

## License

This project is distributed under the GPL-3.0 license. See the [LICENSE](LICENSE) file for more details.

## Acknowledgments

GIRUS is made possible thanks to the contribution of many people and projects:

- **LINUXtips Team**: For the development and maintenance of the project
- **Contributors**: Developers, content creators, and translators
- **Open Source Projects**: Go, React, Kubernetes, Kind, Docker, and many others
- **Community**: All users and supporters who believe in the project

---

## FAQ - Frequently Asked Questions

**Q: Does GIRUS work offline?**  
A: Yes, after initial installation and image downloads, GIRUS can work completely offline.

**Q: How much resources does it consume on my machine?**  
A: GIRUS is optimized to be lightweight. A basic cluster consumes approximately 1-2GB of RAM and requires about 5GB of disk space.

**Q: Can I create custom labs for my team/company?**  
A: Absolutely! The template system is flexible and allows the creation of specific labs for your needs.

**Q: How do I update GIRUS to the latest version?**  
A: Run the command `girus update`. The command will check if a newer version is available and, if so, will execute the update automatically. After the update, you will have the option to recreate the cluster to ensure compatibility with new features.

**Q: Does GIRUS work in corporate environments with network restrictions?**  
A: Yes, after the initial image downloads, GIRUS operates locally without the need for external connections.

**Q: Can I contribute new labs to the project?**  
A: Definitely! Contributions are welcome and valued. See the ["Contributing Labs"](#contributing-labs) section for details.
