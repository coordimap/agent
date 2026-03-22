# Cleye (Coordimap Agent)

## Project Overview

Cleye is a robust infrastructure data crawler agent written in Go. It is designed to gather inventory, configuration, and network flow data from various sources (cloud providers, databases, Kubernetes) and ship it to the Coordimap platform (or any compatible collector).

Key features include:
*   **Multi-Source Crawling:** Supports AWS, GCP, Kubernetes, PostgreSQL, MariaDB/MySQL, MongoDB, and Flow Logs.
*   **eBPF Integration:** Capable of capturing network flows using eBPF (requires `bpftool`, `clang`, `llvm`).
*   **Modular Architecture:** Uses a factory pattern to easily plug in new integrations.

## Architecture

The project follows the **Standard Go Project Layout**:

*   **`cmd/`**: Entry points.
    *   `agent/`: The main agent binary.
    *   `collector/`: A collector component (likely for testing or standalone usage).
*   **`internal/`**: Private application and library code.
    *   `config/`: Configuration loading logic (`yaml`, `file`).
    *   `integrations/`: Core crawling logic for each supported data source.
    *   `cloud/`: Shared cloud provider logic and eBPF flow generation.
*   **`pkg/`**: Library code that could potentially be shared.
    *   `models/`: Shared data structures.
    *   `utils/`: Generic utility functions.
*   **`build/package/`**: Packaging artifacts.
    *   `agent/`: Contains `nfpm.yaml`, systemd service files, and install scripts.
*   **`configs/`**: Example configuration files.

## Building and Running

### Prerequisites
*   Go 1.23+
*   Docker (for containerized builds)
*   `nfpm` (for creating system packages)
*   **eBPF Requirements:** `clang`, `llvm`, `bpftool`, `libbpf-dev`, kernel headers (if building eBPF probes).

### Build Commands

**1. Generate eBPF Artifacts (Required for eBPF support):**
```bash
go generate ./internal/cloud/flows
```

**2. Build the Agent:**
```bash
go build -o agent cmd/agent/main.go
```

**3. Run Tests:**
```bash
go test ./...
```

**4. Build Docker Image:**
```bash
docker build -t cleye-agent .
```

**5. Create System Packages (deb/rpm/apk):**
Navigate to the build directory and use `nfpm`:
```bash
# (Assuming binaries are built and placed correctly)
nfpm pkg --packager deb --target .
```
*Note: The CI pipeline handles this automatically.*

## Configuration

The agent is configured via a YAML file (default `config.yaml`).
*   **Example:** `configs/agent.example.yaml` (contains detailed comments).
*   **Environment Variables:** Can be used in the config file (e.g., `${ENV_VAR}`).
*   **Flags:**
    *   `--config`: Path to config file.
    *   `--endpoint`: URL of the collector.
    *   `--debug`: Enable debug logging.

## Development Conventions

*   **Imports:** Internal packages are imported as `cleye/internal/...`.
*   **Logging:** Uses `zerolog` for structured logging.
*   **Dependency Management:** `go.mod` handles Go dependencies.
*   **CI/CD:** Azure Pipelines (`azure-pipelines.yaml`) manages testing, Docker builds, and GitHub releases.

## Key Files

*   `cmd/agent/main.go`: The main entry point. Initializes config, starts crawlers, and handles data shipping.
*   `internal/integrations/integrations.go`: The factory that initializes specific crawlers based on the config.
*   `internal/config/file_config.go`: Logic for reading and parsing the YAML configuration.
*   `build/package/agent/nfpm.yaml`: Configuration for generating `.deb`, `.rpm`, and `.tar.gz` releases.
*   `Dockerfile`: Multi-stage build file for creating the agent container.
