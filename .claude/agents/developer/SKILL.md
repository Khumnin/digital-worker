---
name: fullstack-developer
description: |
  Full-stack development expertise with Next.js frontend and Go backend.
  Use when: building web applications, developing Go APIs, creating Next.js frontends,
  setting up databases, deploying full-stack apps, or when user mentions Next.js,
  Go, Golang, Gin, Echo, REST API, PostgreSQL, or full-stack development.
---

# Full-Stack Developer (Next.js + Go)

Expert in building production-grade full-stack applications with **Next.js** on the frontend and **Go** on the backend.

---

## Technology Stack

### Frontend — Next.js (App Router)
- **Next.js 14+** — App Router, Server Components, Server Actions, Route Handlers
- **TypeScript** — strict mode, no `any`
- **Tailwind CSS** — utility-first styling
- **shadcn/ui** — accessible component library
- **State** — React Query (server state), Zustand (client state), react-hook-form (forms)
- **Validation** — Zod (shared schemas between frontend and backend contracts)

### Backend — Go
- **Framework** — Gin (default) or Echo for HTTP routing
- **ORM / Query Builder** — GORM (rapid dev) or sqlc (type-safe SQL)
- **Auth** — JWT (`golang-jwt/jwt`), OAuth2 (`golang.org/x/oauth2`)
- **Validation** — `go-playground/validator`
- **Config** — `joho/godotenv` + typed config structs
- **Migrations** — `golang-migrate/migrate`
- **Testing** — `testing` stdlib + `testify`

### Database
- **PostgreSQL** — primary relational database
- **Redis** — caching, sessions, rate limiting
- **Docker Compose** — local dev database setup

### DevOps
- **Docker** — multi-stage builds for Go (minimal final image)
- **GitHub Actions** — CI/CD (lint, test, build, deploy)
- **Vercel** — Next.js frontend deployment
- **Fly.io / Railway** — Go backend deployment

---

## Project Scaffold

### Monorepo Structure
```
my-app/
├── frontend/                    # Next.js app
│   ├── src/
│   │   ├── app/                 # App Router pages & layouts
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx
│   │   │   └── api/             # Route Handlers (proxy/BFF only)
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/ui base components
│   │   │   └── features/        # Feature-specific components
│   │   ├── lib/
│   │   │   ├── api.ts           # API client (fetch wrapper)
│   │   │   └── utils.ts
│   │   ├── hooks/               # Custom React hooks
│   │   └── types/               # Shared TypeScript types
│   ├── .env.local
│   ├── next.config.ts
│   ├── tailwind.config.ts
│   └── package.json
│
├── backend/                     # Go API
│   ├── cmd/
│   │   └── api/
│   │       └── main.go          # Entry point
│   ├── internal/
│   │   ├── config/              # App configuration
│   │   ├── handler/             # HTTP handlers (controllers)
│   │   ├── service/             # Business logic
│   │   ├── repository/          # Data access layer
│   │   ├── model/               # Domain models / DTOs
│   │   ├── middleware/          # Auth, logging, CORS, rate limit
│   │   └── router/              # Route registration
│   ├── migrations/              # SQL migration files
│   ├── .env
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
└── docker-compose.yml           # PostgreSQL + Redis for local dev
```

---

## Code Patterns

### Go — Entry Point
```go
// backend/cmd/api/main.go
package main

import (
    "log"
    "my-app/internal/config"
    "my-app/internal/router"
)

func main() {
    cfg := config.Load()

    r := router.New(cfg)

    log.Printf("Server running on :%s", cfg.Port)
    if err := r.Run(":" + cfg.Port); err != nil {
        log.Fatal(err)
    }
}
```

### Go — Config
```go
// backend/internal/config/config.go
package config

import (
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    Port        string
    DatabaseURL string
    RedisURL    string
    JWTSecret   string
    Env         string
}

func Load() *Config {
    _ = godotenv.Load()
    return &Config{
        Port:        getEnv("PORT", "8080"),
        DatabaseURL: mustGetEnv("DATABASE_URL"),
        RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
        JWTSecret:   mustGetEnv("JWT_SECRET"),
        Env:         getEnv("ENV", "development"),
    }
}

func mustGetEnv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        panic("missing required env var: " + key)
    }
    return v
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### Go — Handler (Gin)
```go
// backend/internal/handler/user_handler.go
package handler

import (
    "net/http"
    "my-app/internal/model"
    "my-app/internal/service"
    "github.com/gin-gonic/gin"
)

type UserHandler struct {
    userService service.UserService
}

func NewUserHandler(us service.UserService) *UserHandler {
    return &UserHandler{userService: us}
}

func (h *UserHandler) Create(c *gin.Context) {
    var req model.CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    user, err := h.userService.Create(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
        return
    }

    c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) GetByID(c *gin.Context) {
    id := c.Param("id")

    user, err := h.userService.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    c.JSON(http.StatusOK, user)
}
```

### Go — Service Layer
```go
// backend/internal/service/user_service.go
package service

import (
    "context"
    "my-app/internal/model"
    "my-app/internal/repository"
    "golang.org/x/crypto/bcrypt"
)

type UserService interface {
    Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error)
    GetByID(ctx context.Context, id string) (*model.User, error)
}

type userService struct {
    repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
    return &userService{repo: repo}
}

func (s *userService) Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }

    return s.repo.Create(ctx, model.User{
        Email:        req.Email,
        Name:         req.Name,
        PasswordHash: string(hash),
    })
}

func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
    return s.repo.FindByID(ctx, id)
}
```

### Go — Repository Layer
```go
// backend/internal/repository/user_repository.go
package repository

import (
    "context"
    "my-app/internal/model"
    "gorm.io/gorm"
)

type UserRepository interface {
    Create(ctx context.Context, user model.User) (*model.User, error)
    FindByID(ctx context.Context, id string) (*model.User, error)
    FindByEmail(ctx context.Context, email string) (*model.User, error)
}

type userRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user model.User) (*model.User, error) {
    result := r.db.WithContext(ctx).Create(&user)
    return &user, result.Error
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
    var user model.User
    result := r.db.WithContext(ctx).First(&user, "id = ?", id)
    return &user, result.Error
}
```

### Go — Models
```go
// backend/internal/model/user.go
package model

import "time"

type User struct {
    ID           string    `json:"id"    gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Email        string    `json:"email" gorm:"uniqueIndex;not null"`
    Name         string    `json:"name"  gorm:"not null"`
    PasswordHash string    `json:"-"     gorm:"not null"`  // never expose
    CreatedAt    time.Time `json:"createdAt"`
    UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateUserRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Name     string `json:"name"     binding:"required,min=1,max=100"`
    Password string `json:"password" binding:"required,min=8"`
}
```

### Go — Router
```go
// backend/internal/router/router.go
package router

import (
    "my-app/internal/config"
    "my-app/internal/handler"
    "my-app/internal/middleware"
    "github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(middleware.Logger())
    r.Use(middleware.CORS())

    api := r.Group("/api/v1")
    {
        users := api.Group("/users")
        {
            uh := handler.NewUserHandler( /* inject deps */ )
            users.POST("", uh.Create)
            users.GET("/:id", uh.GetByID)
        }

        auth := api.Group("/auth")
        auth.Use(middleware.RequireAuth(cfg.JWTSecret))
        {
            // protected routes
        }
    }

    return r
}
```

### Go — JWT Middleware
```go
// backend/internal/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func RequireAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
            return []byte(secret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        claims := token.Claims.(jwt.MapClaims)
        c.Set("userID", claims["sub"])
        c.Next()
    }
}
```

### Go — Dockerfile (multi-stage)
```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# Final stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

---

### Next.js — API Client
```typescript
// frontend/src/lib/api.ts
const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(error.error || `HTTP ${res.status}`)
  }

  return res.json()
}

export const api = {
  get:    <T>(path: string, init?: RequestInit) => request<T>(path, { method: 'GET', ...init }),
  post:   <T>(path: string, body: unknown)      => request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put:    <T>(path: string, body: unknown)      => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: <T>(path: string)                     => request<T>(path, { method: 'DELETE' }),
}
```

### Next.js — Data Fetching with React Query
```typescript
// frontend/src/hooks/useUser.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'

export function useUser(id: string) {
  return useQuery({
    queryKey: ['users', id],
    queryFn: () => api.get<User>(`/users/${id}`),
    enabled: !!id,
  })
}

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateUserInput) => api.post<User>('/users', data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}
```

---

## Setup Guide

### docker-compose.yml (local dev)
```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: myapp
      POSTGRES_PASSWORD: secret
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pg_data:
```

### Backend Setup
```bash
cd backend
cp .env.example .env
# Edit .env with your values

go mod tidy
go run ./cmd/api          # run server

# Migrations
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate -path migrations -database $DATABASE_URL up
```

### Frontend Setup
```bash
cd frontend
npm install
cp .env.example .env.local
# Set NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1

npm run dev
```

---

## Coding Standards

| Concern | Convention |
|---------|-----------|
| Go packages | lowercase, single word (`handler`, not `handlers`) |
| Go files | snake_case (`user_handler.go`) |
| Go exported | PascalCase (`UserService`) |
| Go unexported | camelCase (`userService`) |
| Next.js components | PascalCase (`UserCard.tsx`) |
| Next.js files | kebab-case (`user-card.tsx`) |
| TS types/interfaces | PascalCase (`UserProfile`) |
| TS functions/vars | camelCase (`getUserById`) |
| Constants | UPPER_SNAKE (`MAX_RETRIES`) |
| Error handling (Go) | always check and return errors — never ignore |
| No `any` (TS) | always type explicitly |

## Go-Specific Rules
- Use interfaces for all service and repository layers (enables testing)
- Inject dependencies via constructors — no global state
- Context must flow through all function calls
- Return `error` as the last return value — always
- Never log and return an error — do one or the other
- Prefer `errors.Is` / `errors.As` over string comparison
