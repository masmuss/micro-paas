# Micro PaaS

A simple Platform as a Service (PaaS) built with Go that allows you to manage Docker containers and route traffic via subdomains automatically.

## Features

- **Instance Management**: Create, list, and delete application instances.
- **Automatic Proxying**: Requests to `subdomain.yourdomain.com` are automatically proxied to the corresponding Docker container.
- **Environment Variables**: Configure your application using custom environment variables.
- **Custom Internal Port**: Support for applications running on any port inside the container (defaults to 80).
- **Health Monitoring**: Periodically syncs container status from Docker to the database.
- **Self-Healing**: Automatically detects if a container is down and updates its status.

## Prerequisites

- Go 1.22+
- Docker (daemon must be running)
- SQLite (for the database)

## Getting Started

### 1. Configuration

Copy `config.example.yml` to `config.yml` (if provided) or ensure you have a `config.yml` with the following content:

```yaml
port: 8080
docker_socket: /var/run/docker.sock
db_dsn: micro-paas.db
```

### 2. Run the Application

```bash
go run cmd/main.go
```

The management API will be available at `http://localhost:8080/api`.

## API Documentation

### Create Instance

**Endpoint:** `POST /api/instances`

**Request Body:**

```json
{
  "name": "my-web-app",
  "image": "nginx:latest",
  "subdomain": "webapp",
  "port": 80,
  "env": {
    "APP_COLOR": "blue",
    "DEBUG": "true"
  }
}
```

### List Instances

**Endpoint:** `GET /api/instances`

### Delete Instance

**Endpoint:** `DELETE /api/instances/{id}`

## How it Works

1.  **Creation**: When you create an instance, Micro PaaS pulls the Docker image, creates a container with the specified name and environment variables, and starts it.
2.  **Routing**: The `InstanceProxy` middleware intercepts incoming requests, extracts the subdomain, looks up the corresponding container ID and port in the database, and proxies the request using `httputil.ReverseProxy`.
3.  **Sync**: A background `HealthChecker` service keeps the database in sync with the actual Docker container states.
4.  **Self-Healing**: If a container is detected as down during a health check, its status is updated in the database, and it can be restarted or removed via the API.
