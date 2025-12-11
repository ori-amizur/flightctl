# Development Guide

## Prerequisites

- **Go**: 1.24.0 or later
- **Make**: Build automation
- **Docker/Podman**: Container runtime
- **PostgreSQL**: Database (for local development)
- **Redis**: Cache/queue (for local development)

## Project Setup

### Clone Repository

```bash
git clone <repository-url>
cd flightctl
```

### Install Dependencies

```bash
go mod download
```

### Build Tools

```bash
make build
```

## Development Workflow

### Local Development

**Run API Server:**
```bash
make run-api
# or
go run cmd/flightctl-api/main.go
```

**Run Agent:**
```bash
make run-agent
# or
go run cmd/flightctl-agent/main.go
```

**Run Worker:**
```bash
make run-worker
# or
go run cmd/flightctl-worker/main.go
```

### Build Commands

**Build all binaries:**
```bash
make build
```

**Build specific component:**
```bash
make build-api
make build-agent
make build-worker
```

**Build containers:**
```bash
make build-containers
```

## Testing

### Run Tests

**All tests:**
```bash
make test
```

**Unit tests:**
```bash
go test ./...
```

**Integration tests:**
```bash
make test-integration
```

**E2E tests:**
```bash
make test-e2e
```

### Test Patterns

Test files follow Go conventions:
- `*_test.go` - Test files
- `test/` directory - Test suites
- `test/integration/` - Integration tests
- `test/e2e/` - End-to-end tests

## Code Structure

### Adding New Features

1. **API Changes**: `api/v1beta1/` or `api/grpc/v1/`
2. **Business Logic**: `internal/service/`
3. **Data Models**: `internal/store/model/`
4. **API Handlers**: `internal/api_server/` or `internal/api/server/`

### Code Organization

- **cmd/**: Application entry points
- **internal/**: Private application code
- **pkg/**: Public reusable packages
- **api/**: API definitions
- **test/**: Test suites

## Configuration

**Config Files:**
- `config.yaml` - Main configuration
- Environment variables for overrides
- Viper for configuration management

**Development Config:**
- Local database connection
- Development authentication settings
- Debug logging

## Database

### Migrations

**Run migrations:**
```bash
make migrate
# or
go run cmd/flightctl-db-migrate/main.go
```

### Database Setup

1. Install PostgreSQL
2. Create database
3. Run migrations
4. Configure connection in `config.yaml`

## Observability

**Metrics**: Prometheus endpoints
**Tracing**: OpenTelemetry integration
**Logging**: Structured logging with Logrus

## Development Tools

**Linting:**
```bash
make lint
```

**Formatting:**
```bash
make fmt
```

**Code Generation:**
```bash
make generate
```

## Container Development

**Build containers:**
```bash
make build-containers
```

**Run with Podman:**
```bash
make run-podman
```

## Documentation

**User Docs**: `docs/user/`
**Developer Docs**: `docs/developer/`
**API Docs**: Generated from code

## Common Tasks

### Adding a New Service

1. Create entry point in `cmd/`
2. Add service logic in `internal/service/`
3. Add API handlers if needed
4. Add tests
5. Update deployment configs

### Adding a New API Endpoint

1. Define in `api/v1beta1/` or `api/grpc/v1/`
2. Add handler in `internal/api_server/` or `internal/api/server/`
3. Add service method in `internal/service/`
4. Add tests
5. Update API documentation

## Notes

For more detailed information:
- See `Makefile` for all available commands
- Check `docs/developer/README.md` for developer documentation
- Review existing code for patterns and conventions

