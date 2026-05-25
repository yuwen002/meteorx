# MeteorX API

A multi-tenant REST API built with Go, Chi router, GORM, and Redis.

## Project Structure

```
meteorx-api/
├── cmd/
│   └── server/
│       └── main.go              # Program entry point
│
├── internal/
│   ├── bootstrap/              # Startup initialization (core)
│   │   ├── app.go              # App initialization
│   │   ├── config.go          # Config loading
│   │   ├── database.go        # DB initialization
│   │   ├── router.go          # Route registration (Chi)
│   │   └── middleware.go      # Global middleware
│   │
│   ├── modules/               # ⭐ Business modules (core)
│   │   ├── auth/
│   │   ├── user/
│   │   ├── tenant/
│   │   ├── rbac/
│   │   └── audit/
│   │
│   ├── middleware/
│   ├── common/
│   ├── database/
│   ├── cache/
│   └── config/
│
├── pkg/                       # ⭐ Reusable base library
│   ├── logger/
│   ├── jwt/
│   ├── crypto/
│   └── pagination/
│
├── deployments/
│   ├── docker/
│   └── nginx/
│
├── scripts/
├── docs/
├── .env
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Features

- Multi-tenant architecture
- Authentication & Authorization (JWT)
- Role-Based Access Control (RBAC)
- Audit logging
- RESTful API with Chi router
- PostgreSQL with GORM
- Redis caching
- Docker support

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker (optional)

### Installation

1. Clone the repository
2. Copy `.env` and configure environment variables
3. Run migrations
4. Start the server

```bash
# Install dependencies
go mod download

# Run the server
go run cmd/server/main.go
```

### Docker

```bash
# Build and run with docker-compose
docker-compose up -d
```

## API Endpoints

- `/api/v1/auth` - Authentication
- `/api/v1/users` - User management
- `/api/v1/tenants` - Tenant management
- `/api/v1/rbac` - Roles & Permissions
- `/api/v1/audit` - Audit logs

## Configuration

Configuration is loaded from `internal/config/config.yaml` and environment variables.

## License

MIT
