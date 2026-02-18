# RiceSafe Backend API

RiceSafe Backend is the server-side application powering the RiceSafe mobile app. It handles user authentication, disease diagnosis, outbreak alert, community features, and data management using PostgreSQL.

## Features

- **User Authentication:** Secure registration, login, and profile management using JWT and Bcrypt.
- **AI-Powered Diagnosis:** Integrates with the AI Service to process rice leaf images, description and return predictions.
- **Disease Library:** Manages a comprehensive database of rice diseases, symptoms, and treatments.
- **Outbreak alert Map:** Tracks diagnosed disease locations and calculates distances to warn nearby farmers.
- **Community Feed:** A social platform for users to create posts, like, and comment to share knowledge.
- **Image Storage:** Handles secure image uploads to Google Cloud Storage (GCS).
- **API Documentation:** Interactive Swagger UI for testing and exploring API endpoints.

## Tech Stack

- **Language:** Go (Golang)
- **Framework:** Fiber (v2) - Express-inspired web framework
- **Database:** PostgreSQL (with `pgx` driver)
- **Migration:** Golang-Migrate
- **Architecture:** Modular Monolith (Clean Architecture)
- **Containerization:** Docker & Docker Compose
- **Documentation:** Swaggo (Swagger/OpenAPI)
- **Storage:** Google Cloud Storage (GCS)

## Architecture

The project follows a **Modular Monolith** architecture where each domain (feature) is self-contained with its own layers of concern.

### Layers
Each module (e.g., `auth`, `diagnosis`, `community`) typically contains:
1.  **Handler (`handler.go`):** HTTP layer, handles requests/responses and validation.
2.  **Service (`service.go`):** Business logic, calls repositories or external services (AI/GCS).
3.  **Repository (`repository.go`):** Data access layer, executes SQL queries.
4.  **Models (`models.go`):** Go structs defining the data structures.

## Project Structure

```
rice-safe-backend/
├── cmd/
│   └── api/                # Main entry point (`main.go`)
├── internal/               # Application modules
│   ├── auth/               # User authentication & Profile
│   ├── community/          # Posts, Comments, Likes
│   ├── config/             # Configuration loader
│   ├── diagnosis/          # Diagnosis logic & History
│   ├── disease/            # Disease Library CRUD
│   ├── outbreak/           # Outbreak tracking & Maps
│   └── platform/           # Shared infra (Database, GCS, AI Client)
├── migrations/             # SQL migration files
├── docs/                   # Generated Swagger docs (docs.go, swagger.json)
├── pkg/                    # Public shared packages (if any)
├── tmp/                    # Temporary build artifacts (Air)
├── .air.toml               # Air configuration for live reload
├── .env.example            # Environment variables example
├── .gitignore              # Git ignore rules
├── docker-compose.yml      # Docker services (DB, AI, Backend)
├── Dockerfile              # Docker build instructions for Backend
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── Makefile                # Command shortcuts
├── service-account.json    # Google Cloud Service Account (for GCS)
└── README.md               # This file
```

## API Documentation

Once the server is running, you can access the interactive API documentation at:

**[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

Use this UI to:
-   Visualize all available endpoints.
-   Authorize with your JWT token (`Bearer <token>`).
-   Test API requests directly.
