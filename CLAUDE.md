# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- Build the API service: `go build -o towercore ./cmd/api`
- Run the API service: `./towercore` or `go run ./cmd/api`
- Build all services: `go build ./cmd/...`
- Start development environment: `docker-compose up -d`
- Stop development environment: `docker-compose down`

### Testing
- Run all tests: `go test ./...`
- Run tests with coverage: `go test ./... -cover`
- Run tests for a specific package: `go test ./internal/core/services/tower`
- Run a single test function: `go test -run TestFunctionName ./internal/core/services/tower`
- Run tests with verbose output: `go test -v ./...`

### Code Quality
- Format code: `go fmt ./...`
- Vet code for issues: `go vet ./...`
- Check for lint issues: `golangci-lint run` (if installed)
- Generate mocks: `mockery` (used for generating interface mocks in tests)

### Database Management
- Apply migrations: `goose up` (requires goose tool)
- Rollback migrations: `goose down`
- Check migration status: `goose status`
- Create new migration: `goose create migration_name sql`

## Architecture Overview

### Hexagonal Architecture
Antosc follows a hexagonal (ports and adapters) architecture with clear separation of concerns:

1. **Core Domain** (`internal/core/domain/`) - Business entities and value objects
2. **Core Interfaces** (`internal/core/interfaces/`) - Repository and service contracts
3. **Core Services** (`internal/core/services/`) - Business logic implementations
4. **API Layer** (`internal/api/`) - HTTP handlers, routes, and middleware
5. **Infrastructure** (`internal/infrastructure/`) - Database, logging, caching, configuration
6. **Adapters** (`internal/adapters/`) - Vendor-specific protocol implementations (SNMP, etc.)
7. **Scheduler** (`internal/scheduler/`) - Polling mechanisms for device monitoring
8. **Observability** (`internal/observability/`) - Metrics collection and monitoring

### Key Directories
- `cmd/` - Application entry points (api, scheduler, comap utilities)
- `internal/core/` - Domain entities, interfaces, and business services
- `internal/api/` - HTTP API implementation with handlers and middleware
- `internal/adapters/` - Protocol-specific implementations for device communication
- `internal/scheduler/` - Background processes for polling equipment
- `internal/infrastructure/` - Technical concerns (database, logger, security, cache)
- `pkg/` - Shared libraries used across the application
- `migrations/` - SQL database schema migrations
- `observability/` - Prometheus and Grafana configuration

### Data Flow
1. Scheduler processes poll configured devices via vendor adapters
2. Adapters translate protocol-specific data to normalized metrics
3. Metrics are ingested through service layer and stored in PostgreSQL
4. API layer exposes endpoints for querying data and managing configuration
5. Observability stack collects metrics from the application and database

### Communication Patterns
- Repository pattern for data access
- Dependency injection for service configuration
- Middleware chains for cross-cutting concerns (auth, rate limiting, logging)
- Event-driven architecture for alerts and notifications
- Standardized error handling via `pkg/apierror`

## Development Guidelines

Based on existing contribution guidelines:

### Branch Strategy
- Create feature branches from main: `feature/*`, `fix/*`, `docs/*`
- Implement small, focused changes
- Update related documentation
- Open PRs with context, impact, and validation instructions

### Go Code Standards
- Prioritize simplicity and clarity
- Business logic belongs in `internal/core`
- HTTP handlers should contain no business logic
- Write unit tests for services and domain validators

### Commit Message Convention
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `refactor:` for code refactoring
- `test:` for test additions/changes
- `chore:` for maintenance tasks

### Definition of Done
- Code implemented and reviewed
- Relevant local tests executed
- Documentation updated
- No secrets or sensitive data versioned

## Important Notes

- The system currently provides only API endpoints - no frontend components are present
- Vendor-specific SNMP profiles require maintenance as equipment evolves
- Environment configuration is managed through `.env` file and Docker Compose
- Database migrations should be backward compatible and tested thoroughly
- Cache invalidation is implemented in services but requires careful attention