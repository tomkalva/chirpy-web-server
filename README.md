# Chirpy

A lightweight social feed API server, inspired by early Twitter,
built to practice designing and securing a RESTful HTTP backend in Go.

## Why this project?

Chirpy demonstrates a full authentication and content-management flow
from scratch, without relying on a framework:

- User registration and login with hashed passwords
- JWT-based access tokens and long-lived refresh tokens
- CRUD operations on user-generated content ("chirps"), with ownership checks
- Query-parameter-based filtering and sorting
- Webhook handling for third-party payment upgrades (Polka)
- Admin-only endpoints gated by environment/platform checks

It's a good reference for anyone learning how to structure a Go web
server with a real Postgres database, JWT auth, and clean separation
of concerns between handlers, business logic, and data access.

## Tech stack

- Go (standard library `net/http`, no framework)
- PostgreSQL
- [sqlc](https://sqlc.dev/) for generated, type-safe SQL queries
- JWT for access tokens

## Getting started

### Prerequisites

- Go 1.2x+
- PostgreSQL running locally (or accessible via connection string)

### Setup

1. Clone the repo:
   ```sh
   git clone https://github.com/tomkalva/chirpy-web-server.git
   cd chirpy-web-server
   ```

2. Create a .env file in the project root:
```sh
DB_URL=postgres://user:password@localhost:5432/chirpy?sslmode=disable
PLATFORM=dev
JWT_SECRET=your-secret-here
POLKA_KEY=your-polka-key-here
```

3. Run database migrations using goose:
```sh
goose postgres "$DB_URL" up
```

4. Start the server:
```sh
go run .
```

The server runs on http://localhost:8080 by default.

## API Overview

| Method | Endpoint              | Description                             |
|--------|------------------------|------------------------------------------|
| POST   | /api/users             | Create a new user                       |
| PUT    | /api/users             | Update email/password (auth required)   |
| POST   | /api/login             | Log in, receive access + refresh token  |
| POST   | /api/refresh           | Get a new access token                  |
| POST   | /api/revoke            | Revoke a refresh token                  |
| POST   | /api/chirps            | Create a chirp (auth required)          |
| GET    | /api/chirps            | List chirps (supports `sort`, `author_id`) |
| GET    | /api/chirps/{chirpID}  | Get a single chirp                      |
| DELETE | /api/chirps/{chirpID}  | Delete your own chirp                   |
| POST   | /api/polka/webhooks    | Webhook for upgrade events              |
| POST   | /admin/reset           | Reset users and admin metrics           |
| GET    | /admin/metrics         | Prints admin metrics                    |
| GET    | /api/healthz           | Get server status                       |
