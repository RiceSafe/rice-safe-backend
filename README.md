# RiceSafe Backend API

![CI](https://github.com/RiceSafe/rice-safe-backend/actions/workflows/test.yml/badge.svg)

RiceSafe Backend is a robust Go-based server-side application powering the RiceSafe ecosystem. It handles user authentication, disease diagnosis, outbreak tracking, community engagement, and real-time data management.

## Features

- **Multi-Platform Authentication:** Secure registration and login via JWT. Supports Email/Password, Google OAuth (Web, iOS, Android), and LINE Login.
- **AI-Powered Diagnosis:** Seamless integration with our AI Service to analyze rice images and symptoms for accurate disease detection.
- **Real-Time Weather:** Integrated with OpenWeatherMap API to provide hyper-local weather data on the user dashboard.
- **Outbreak Monitoring:** Dynamic tracking of disease outbreaks with proximity-based alerts and interactive maps.
- **Intelligent Notifications:** System-wide notification engine for critical alerts and community updates.
- **Community Ecosystem:** Full-featured social platform for knowledge sharing, including posts, likes, and moderated comments.
- **Cloud Storage:** Secure and scalable image management using Google Cloud Storage (GCS).
- **Automated Communication:** Transactional email delivery (e.g., password resets) powered by the Resend API.
- **API Documentation:** Interactive Swagger UI for developers to explore and test endpoints.

## Tech Stack

- **Language:** Go (Golang) 1.26+
- **Web Framework:** Fiber v2
- **Database:** PostgreSQL (with `pgx`)
- **Migrations:** Golang-Migrate
- **Authentication:** JWT, Bcrypt, Google GSI, LINE Login
- **Documentation:** Swaggo (Swagger/OpenAPI)
- **Infrastructure:**
  - **Storage:** Google Cloud Storage (GCS)
  - **Email:** Resend API
  - **Weather:** OpenWeatherMap API
  - **AI:** Custom AI Service Integration
- **DevOps:** Docker, Docker Compose, Makefile, Air (Hot Reload)

## Architecture

The project follows a **Modular Monolith** pattern, prioritizing clean separation of concerns and maintainability.

### Project Structure

```bash
rice-safe-backend/
├── cmd/
│   └── api/                # Application entry point (main.go)
├── internal/               # Core business logic
│   ├── auth/               # IAM, JWT, OAuth (Google/LINE), and User Management
│   ├── community/          # Social features (Posts, Comments, Likes)
│   ├── config/             # Environment configuration (Fiber/Env)
│   ├── dashboard/          # Weather & Statistics integration
│   ├── diagnosis/          # AI Diagnosis & History tracking
│   ├── disease/            # Disease Knowledge Base
│   ├── notification/       # Alerting & Message system
│   ├── outbreak/           # Geospatial disease tracking
│   ├── platform/           # Infrastructure Adapters (Database, Storage, Email, AI Client)
│   ├── server/             # Fiber app initialization & Dependency Injection
│   └── testutil/           # Test helpers, Database setup, and Mocks
├── pkg/                    # Shared utility packages
├── migrations/             # SQL migration files (PostgreSQL)
├── tests/                  # API Integration & End-to-End Test Suite
├── docs/                   # Auto-generated Swagger/OpenAPI documentation
├── Makefile                # Automation scripts (database, swagger, dev environment)
├── docker-compose.yml      # Infrastructure orchestration (DB)
├── Dockerfile              # Local development container config
└── Dockerfile.prod         # Multi-stage production build config
```

### Dependency Injection
We use a centralized `SetupApp` pattern in `internal/server/app.go` to wire up dependencies. This ensures that the application is easily testable and the entry point (`main.go`) remains clean.

## Getting Started

### Prerequisites
- Go 1.26 or higher
- Docker & Docker Compose
- `make` utility

### Development
1. **Setup Environment**: Copy `.env.example` to `.env` and fill in your credentials.
2. **Start Infrastructure**:
   ```bash
   make docker-up
   ```
3. **Run Migrations**:
   ```bash
   make migrate-up
   ```
4. **Start Application**:
   ```bash
   make dev
   ```

### Shortcuts (Makefile)
- `make swagger`: Regenerate API documentation.
- `make test`: Run the integration test suite.
- `make docker-down`: Stop all services.

## API Documentation

Once the server is running, explore the API at:
**[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

## Security & Health
- All sensitive routes are protected by JWT middleware.
- **Health Check**: Available at `/api/health`.
