# OpenVDO

A high-performance video streaming backend built with Go, Gin, PostgreSQL, and Redis.

## Tech Stack

- **Backend**: Go 1.21+ with Gin framework
- **Database**: PostgreSQL with golang-migrate
- **Cache**: Redis
- **Hot Reload**: Air
- **Containerization**: Docker & Docker Compose

## Project Structure

```
.
├── cmd/
│   └── server/          # Application entry point
│       └── main.go
├── internal/            # Private application logic
│   ├── config/         # Configuration management
│   ├── database/       # Database connections
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # Gin middleware
│   ├── models/         # Data models
│   ├── routes/         # Route definitions
│   ├── services/       # Business logic
│   └── utils/          # Internal utilities
├── migrations/          # Database migration files
├── pkg/                # Public/reusable packages
│   ├── logger/         # Logging utilities
│   └── response/       # Standardized API responses
├── env/                # Environment-specific configs
├── docs/               # Documentation
└── scripts/            # Build and deployment scripts
```

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd openvdo
   ```

2. **Set up the project**
   ```bash
   make setup
   ```

3. **Configure environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Start the services**

   **Option 1: Using Docker (Recommended for development)**
   ```bash
   make docker-up
   make migrate-up
   make dev
   ```

   **Option 2: Local setup**
   ```bash
   # Start PostgreSQL and Redis locally
   make migrate-up
   make dev
   ```

### Development

- **Run with hot reload**:
  ```bash
  make dev
  ```

- **Run tests**:
  ```bash
  make test
  ```

- **Run tests with coverage**:
  ```bash
  make test-coverage
  ```

- **Code formatting**:
  ```bash
  make fmt
  ```

- **Linting**:
  ```bash
  make lint
  ```

### Database Migrations

- **Run migrations**:
  ```bash
  make migrate-up
  ```

- **Rollback migrations**:
  ```bash
  make migrate-down
  ```

- **Create new migration**:
  ```bash
  make migration-new
  ```

### Building & Deployment

- **Build the application**:
  ```bash
  make build
  ```

- **Run production build**:
  ```bash
  ./bin/openvdo
  ```

- **Docker deployment**:
  ```bash
  make docker-full
  ```

## API Documentation

### Interactive Swagger UI

Start the application and visit:
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **JSON Schema**: `http://localhost:8080/swagger/doc.json`

### API Endpoints

#### Health Check

```http
GET /health
```

#### Users API

```http
# Create user
POST /api/v1/users
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "first_name": "John",
  "last_name": "Doe"
}

# Get all users
GET /api/v1/users

# Get user by ID
GET /api/v1/users/{id}

# Update user
PUT /api/v1/users/{id}
Content-Type: application/json

{
  "first_name": "Jane",
  "last_name": "Smith"
}

# Delete user
DELETE /api/v1/users/{id}
```

### Generating Documentation

To regenerate Swagger documentation after adding new endpoints:

```bash
make swagger
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `GIN_MODE` | Gin mode (debug/release) | `debug` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | `openvdo` |
| `DB_SSLMODE` | Database SSL mode | `disable` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `REDIS_DB` | Redis database number | `0` |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## License

This project is licensed under the MIT License.