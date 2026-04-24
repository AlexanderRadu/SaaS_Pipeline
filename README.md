# SaaS Pipeline

A microservices-based SaaS platform built in Go.

## Architecture

The project consists of several microservices, which communicate with each other:

- **api-gateway**: Entry point that routes incoming requests to the respective microservices.
- **auth**: Handles user authentication and authorization.
- **catalog**: Manages the catalog of services/products.
- **deploy**: Handles deployment workflows.
- **status**: Tracks and provides status updates via RabbitMQ.
- **migrate**: Database migration runner.

### Tech Stack

- **Language**: Go 1.26
- **Database**: PostgreSQL 
- **Message Broker**: RabbitMQ
- **Reverse Proxy**: Nginx
- **API Definition**: OpenAPI (Code generation for handlers and models)
- **Containerization**: Docker, Docker Compose

## Repository Structure

```
SaaS_Pipeline/
├── api/             # OpenAPI specifications and generated config files
├── cmd/             # Entry points for microservices
├── configs/         # Nginx and other configuration files
├── DataBase/        # SQL migration files
├── deployments/     # Dockerfiles and docker-compose configurations
├── internal/        # Internal logic for microservices (handlers, repositories, models)
└── pkg/             # Shared libraries (logger, config, postgres, rabbitmq, migrator)
```

## Running the Project

The project is fully containerized and can be started using Docker Compose.

1. Ensure you have Docker and Docker Compose installed.
2. Run the application:

```bash
docker-compose -f deployments/docker-compose.yml up -d --build
```

This will run all the necessary services, the databases, and message broker.

## API Documentation

The API endpoints are defined via OpenAPI specification. Find the `openapi.yaml` file located in the `api/` directory.

## Development

The project uses `oapi-codegen` to generate server code from OpenAPI files. Configuration files for the generator are present in `api/gen`.

Database migrations are located in `DataBase/migrations/` and run automatically on startup via the `migrate` service.

