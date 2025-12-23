# Go API Starter

A production-ready Go API starter template with built-in authentication, database management, caching, and email support.

## Features

- **Web Framework**: Echo v4 for high-performance HTTP routing
- **Database**: PostgreSQL with SQLC for type-safe queries
- **Caching**: Redis for session and data caching
- **Authentication**: JWT and session-based auth with OTP support
- **Email**: SMTP integration with templated emails
- **File Storage**: S3-compatible blob storage
- **Monitoring**: Prometheus metrics and pprof profiling
- **Logging**: Structured JSON logging with slog
- **Configuration**: Environment-based 12-factor app config
- **Database Migrations**: Version-controlled schema migrations

## Quick Start

### Prerequisites

- Go 1.25.2 or later
- PostgreSQL 12+
- Redis
- Docker & Docker Compose (optional, for containerized setup)

### Setup

1. **Clone the repository**

   ```bash
   git clone <repo-url>
   cd go-api-starter
   ```

2. **Configure environment**

   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Run migrations**

   ```bash
   task db:migrate
   ```

4. **Start development server**

   ```bash
   task run
   ```

The server will start at `http://localhost:8080`

## Project Structure

```
.
├── cmd/                 # Application entry points
│   ├── app/            # Main API server
│   └── devcerts/       # Dev certificate generation
├── handler/            # HTTP request handlers
├── database/           # Database layer (SQLC + migrations)
├── deps/               # External dependencies (config, email, storage, etc.)
├── assets/             # Static files and templates
├── util/               # Utility functions
├── deploy/             # Docker & deployment configs
└── docs/               # Documentation
```

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture overview.

## Configuration

All configuration is managed via environment variables:

### Required

- `APP_ENV` - Environment (development, staging, production)
- `HTTP_HOST` - Server host
- `HTTP_PORT` - Server port
- `POSTGRES_URL` - PostgreSQL connection URL
- `REDIS_URL` - Redis connection URL
- `SESSION_SECRET` - 64-character session secret
- `ALLOWED_ORIGINS` - CORS allowed origins (comma-separated)
- `TMP_DIR` - Temporary directory path

### Optional

- `DEBUG` - Enable debug logging (default: false)

## Development

### Available Commands

```bash
# Run server
task run

# Run tests
task test

# Database migrations
task db:migrate
task db:migrate:down

# Generate code
task generate

# Build Docker image
task build:docker
```

See `Taskfile.yaml` for all available commands.

## API Endpoints

The API includes routes for:

- Authentication (login, signup, logout)
- User account management
- Session management
- OTP verification
- Subscription handling

See handler files for detailed endpoint documentation.

## Database

Database schema is managed via migrations in `database/migrations/`. Queries are defined in `database/queries/` and compiled to type-safe Go code by SQLC.

### Running Migrations

```bash
task db:migrate       # Apply all migrations
task db:migrate:down  # Rollback one migration
```

## Security

- All secrets loaded from environment variables
- Session tokens for CSRF protection
- Secure password hashing with Argon2
- Input validation on all endpoints
- HTTPS support with development certificate generation
- No hardcoded credentials or sensitive data

## Monitoring

The API exposes:

- **Metrics**: Prometheus metrics at `/metrics`
- **Profiling**: pprof profiles at `/debug/pprof/`

## Deployment

### Docker

```bash
docker build -t go-api-starter -f deploy/docker/Dockerfile .
docker run -p 8080:8080 go-api-starter
```

### Docker Compose (Development)

```bash
docker-compose -f deploy/compose/compose.dev.yaml up
```

## Code Style

- Follow Go standard conventions
- Use `fmt.Errorf` for error wrapping with context
- Keep functions small and focused
- Write table-driven tests for complex logic
- Match existing project style

## License

MIT License - see LICENSE file for details

## Support

For issues, questions, or suggestions, please open an issue on GitHub.
