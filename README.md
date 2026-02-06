# dctl

Docker Compose compatible CLI for [Apple container](https://github.com/apple/container).

`dctl` translates `docker compose` commands into Apple `container` CLI invocations, letting you use familiar Compose workflows with the container runtime.

## Requirements

- macOS 15+
- [container](https://github.com/apple/container) CLI installed and running (`container system start`)
- Go 1.23+ (to build from source)

## Install

```bash
git clone https://github.com/raviatluri/dctl.git
cd dctl
make build
# Binary is at bin/dctl

# Optionally install to PATH:
make install
```

## Usage

`dctl` works with standard `compose.yaml` / `docker-compose.yml` files.

```bash
# Start all services in detached mode
dctl compose up -d

# Start and build images first
dctl compose up -d --build

# View running services
dctl compose ps

# Follow logs
dctl compose logs -f
dctl compose logs -f web      # specific service

# Execute a command in a running service
dctl compose exec web bash

# Run a one-off command
dctl compose run --rm web npm test

# Stop services
dctl compose stop

# Stop and remove containers, networks
dctl compose down

# Stop and remove everything including volumes
dctl compose down -v

# Restart services
dctl compose restart

# Build service images
dctl compose build

# Pull service images
dctl compose pull

# Validate compose file
dctl compose config

# Remove stopped containers
dctl compose rm

# Force stop services
dctl compose kill
```

### Global Flags

```
-f, --file         Compose configuration file(s) (can be specified multiple times)
-p, --project-name Project name (defaults to directory name)
--project-directory Alternate working directory
--profile          Activate a profile
--env-file         Alternate environment file
--debug            Enable debug output
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DCTL_CONTAINER_BIN` | Path to the `container` binary (auto-detected if not set) |
| `DCTL_DEBUG` | Enable debug output |

## Compose File Support

`dctl` parses standard Compose files (`compose.yaml`, `compose.yml`, `docker-compose.yml`, `docker-compose.yaml`) and supports:

### Services
- `image`, `build` (context, dockerfile, args, target, labels)
- `command`, `entrypoint`
- `environment`, `env_file`
- `ports`, `volumes`, `tmpfs`
- `networks`, `dns`, `dns_search`
- `depends_on` (with `service_started`, `service_healthy`, `service_completed_successfully` conditions)
- `working_dir`, `user`, `hostname`
- `labels`, `platform`
- `tty`, `stdin_open`, `read_only`
- `cpus`, `mem_limit`
- `restart`, `stop_signal`, `stop_grace_period`
- `container_name`, `pull_policy`
- `healthcheck`

### Top-Level
- `name` (project name)
- `services` (required)
- `networks` (create/external)
- `volumes` (create/external)

### Features
- Environment variable interpolation: `${VAR}`, `${VAR:-default}`, `${VAR-default}`
- Multiple compose files via `-f` (merged in order)
- Dependency ordering via `depends_on` (topological sort with cycle detection)
- Rollback on failure during `up` (stops already-started services)
- Project state tracking in `~/.dctl/projects/`

## How It Works

`dctl` is a translation layer, not a reimplementation. Each compose command orchestrates one or more `container` CLI calls:

| dctl compose | container CLI |
|---|---|
| `up` | `network create` + `volume create` + `run --detach` (per service, in dependency order) |
| `down` | `stop` + `delete` (per container) + `network delete` + `volume delete` |
| `ps` | `list --format json` (filtered by project) |
| `logs` | `logs` (per service) |
| `exec` | `exec` |
| `run` | `run` (with service config + overrides) |
| `build` | `build` (per service with build config) |
| `pull` | `image pull` (per service) |
| `stop` | `stop` (per service) |
| `restart` | `stop` + `start` (per service) |
| `rm` | `delete` (per service) |
| `kill` | `kill` (per service) |
| `config` | Parse and print resolved YAML |

## Limitations

These Docker Compose features are not supported by the container runtime:

- `privileged`, `cap_add`, `cap_drop` (VM-based isolation, not namespace-based)
- `network_mode: host`
- `extra_hosts` / `add-host`
- `devices`, `gpus`
- `logging` drivers
- `deploy` (replicas, resources, placement)
- `healthcheck` (parsed but not actively polled during `up`)
- `profiles` (parsed but not filtered)
- `secrets`, `configs`
- `watch` mode

## Project Structure

```
dctl/
├── main.go                  # Entry point
├── cmd/
│   ├── app.go              # Root CLI command
│   └── compose.go          # All compose commands and flag translation
├── pkg/
│   ├── runner/
│   │   └── runner.go       # Executes container CLI commands
│   └── compose/
│       ├── types.go        # Compose file structs
│       ├── parser.go       # YAML parsing with env interpolation
│       ├── graph.go        # Dependency graph (topological sort)
│       └── project.go      # Project state management
├── go.mod
├── go.sum
└── Makefile
```

## License

Apache License 2.0
