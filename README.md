# Micro PaaS

A simple Platform as a Service (PaaS) built with Go that allows you to manage Docker containers and route traffic via subdomains automatically.

## Features

- **Instance Management**: Create, list, detail, and delete application instances.
- **Automatic Proxying**: Requests to `subdomain.localhost` are automatically proxied to the corresponding Docker container.
- **Dynamic Database Support**: Supports SQLite, PostgreSQL, and MySQL via Bun ORM.
- **Environment Variables**: Configure your application using custom environment variables during creation.
- **Custom Internal Port**: Support for applications running on any port inside the container (defaults to 80).
- **Container Logs**: Stream application logs directly via API for debugging.
- **Health Monitoring & Self-Healing**: Periodically syncs container status and automatically restarts containers if they go down unexpectedly.

## Prerequisites

- Go 1.22+
- Docker (daemon must be running)
- Database (SQLite file, PostgreSQL, or MySQL)

## Getting Started

### 1. Configuration

Ensure you have a `config.yml` in the root directory:

```yaml
server_port: "8080"
docker_socket: "/var/run/docker.sock"
db_driver: "sqlite" # options: sqlite, postgres, mysql
db_dsn: "./micro-paas.db"
```

### 2. Run the Application

```bash
go run cmd/main.go
```

The management API will be available at `http://localhost:8080/api`.

## API Documentation

### Create Instance

`POST /api/instances`

```json
{
  "name": "my-web-app",
  "image": "nginx:latest",
  "subdomain": "webapp",
  "port": 80,
  "env": {
    "APP_COLOR": "blue"
  }
}
```

### List Instances

`GET /api/instances`

### Get Instance Detail

`GET /api/instances/{id}`

### Get Instance Logs

`GET /api/instances/{id}/logs`

### Delete Instance

`DELETE /api/instances/{id}`

## How it Works

1.  **Creation**: Micro PaaS pulls the Docker image, creates a container with custom environment variables, and records the assigned subdomain and port in the database.
2.  **Routing**: The `InstanceProxy` middleware intercepts requests, extracts the subdomain from the `Host` header, and reverse-proxies the traffic to the container's internal IP and port.
3.  **Sync & Self-Healing**: The `HealthChecker` background service monitors container states. If a container is found "Stopped" while its database status is "Running", it will attempt an automatic restart to ensure high availability.

## Testing Subdomains Locally

You can test the reverse proxy without custom DNS using `curl`:

```bash
curl -H "Host: webapp.localhost" http://localhost:8080
```

## License

MIT
