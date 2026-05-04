# Micro PaaS 🚀

A simple Platform as a Service (PaaS) built with Go that allows you to manage Docker containers and route traffic via subdomains automatically.

## Features

- **Web Dashboard**: Clean UI built with HTMX and Pico.css.
- **Instance Management**: Create, start, stop, and delete application instances.
- **Automatic Proxying**: Dynamic routing to containers via subdomains (e.g., `myapp.micro-paas.local`).
- **Real-time Logs**: View container logs directly from the dashboard.
- **Self-Healing**: Automatically restarts containers if they crash unexpectedly.
- **Docker Integration**: Directly manages containers via Docker Socket.

## Prerequisites

- **OrbStack** (recommended for macOS) or Docker Desktop.
- **Go 1.26+** (for local development).
- **Task** (optional, for running automation commands).

## Setup & Configuration

This project uses a unified `.env` file for configuration. Create a `.env` file in the root directory:

```env
SERVER_PORT=8080
DB_DRIVER=sqlite
DB_DSN=./micro-paas.db
DOCKER_SOCKET=/var/run/docker.sock
MAIN_DOMAIN=micro-paas.local
```

## Running the Application

### Using Docker (Recommended)

The easiest way to run Micro PaaS is using Docker Compose. It's pre-configured for **OrbStack** with automatic domain support.

```bash
# Build and start the container
task docker:up

# View logs
task docker:logs

# Stop the container
task docker:down
```

After starting, access the dashboard at:

- **OrbStack**: [http://micro-paas.local](http://micro-paas.local)
- **Direct**: [http://localhost:8080](http://localhost:8080)

### Local Development (Hot Reload)

To run the application locally with hot reload using `air`:

```bash
task server
```

## How it Works

1.  **Dashboard**: You create an instance by providing an image (e.g., `nginx:latest`) and a subdomain.
2.  **Deployment**: Micro PaaS pulls the image and creates a container.
3.  **Dynamic Routing**: The `InstanceProxy` middleware detects the subdomain from the request and routes traffic to the correct container.
4.  **OrbStack Support**: Uses labels to automatically map the dashboard to `micro-paas.local`.

## Local DNS Testing

If you are using OrbStack, subdomains like `any-app.micro-paas.local` will work automatically. If you are not using OrbStack, you can test via `curl`:

```bash
curl -H "Host: myapp.micro-paas.local" http://localhost:8080
```

## Available Task Commands

- `task build`: Build the server binary.
- `task docker:build`: Build the Docker image.
- `task docker:up`: Run the application in Docker.
- `task docker:dev`: Run development mode with `air` inside Docker.
- `task test`: Run project tests.
- `task tidy`: Clean up go modules.

## License

MIT
