# Solution Architecture Document
# Authentication System — Pilot Project

**Version:** 1.0
**Date:** 2026-02-27
**Status:** Approved for Development
**Author:** Solution Architect Agent
**Input Documents:** PRD v1.1, Project Management Plan v1.0
**Intended Consumers:** Developer Agent, Tester Agent, DevOps Engineer

---

## Revision History

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-02-27 | Initial architecture document — all sections complete |

---

## Table of Contents

1. System Context (C4 Level 1)
2. Container Diagram (C4 Level 2)
3. Component Diagram (C4 Level 3) — Go Auth API Internals
4. Tech Stack Decision Matrix
5. Database Schema (ERD)
6. API Design
7. Go Project Structure
8. Schema-per-Tenant Implementation Design
9. Security Architecture
10. Non-Functional Architecture
11. Architecture Decision Records (ADRs)

---

## 1. System Context (C4 Level 1)

### Diagram

```mermaid
C4Context
  title System Context — Authentication System

  Person(endUser, "End User (Maya)", "Registers, logs in, resets password, uses social login via a consuming application.")
  Person(tenantAdmin, "Tenant Admin (Carlos)", "Manages users, roles, and audit logs within their tenant via API.")
  Person(developer, "Developer / API Consumer (Priya)", "Integrates services using OAuth 2.0; registers OAuth clients; calls token introspection.")
  Person(superAdmin, "Platform Super-Admin (Jordan)", "Provisions tenants, manages platform configuration, views cross-tenant audit data.")

  System(authSystem, "Authentication System", "Multi-tenant authentication and authorization platform. Provides user registration, login, OAuth 2.0 authorization server, RBAC, and audit logging via REST API.")

  System_Ext(googleOAuth, "Google OAuth / OIDC", "External identity provider for social login. Provides OIDC discovery and token endpoints.")
  System_Ext(emailProvider, "Email Provider (Transactional)", "Sends verification emails, password reset emails, and user invitation emails.")
  System_Ext(secretsManager, "Secrets Manager (HashiCorp Vault)", "Stores and rotates signing keys, database credentials, client secrets, and API keys. Never environment variables.")
  System_Ext(resourceServer, "Downstream Resource Servers", "APIs protected by tokens issued from this system. Validate JWTs locally using JWKS endpoint or call /oauth/introspect.")
  System_Ext(coldStorage, "Cold Storage (S3-compatible)", "Long-term archive for audit logs older than 1 year.")

  Rel(endUser, authSystem, "Registers, logs in, resets password, refreshes session, links social account", "HTTPS/REST")
  Rel(tenantAdmin, authSystem, "Invites/disables/deletes users, assigns roles, reads audit log", "HTTPS/REST")
  Rel(developer, authSystem, "Registers OAuth clients, initiates authorization code flow, introspects tokens, fetches JWKS", "HTTPS/REST")
  Rel(superAdmin, authSystem, "Provisions tenants, views platform-level audit data, manages signing keys", "HTTPS/REST")

  Rel(authSystem, googleOAuth, "Fetches OIDC discovery metadata, exchanges authorization code for ID token, verifies token", "HTTPS/OIDC")
  Rel(authSystem, emailProvider, "Sends transactional emails (verification, reset, invitation)", "HTTPS/SMTP API")
  Rel(authSystem, secretsManager, "Reads signing keys, DB credentials, client secrets at runtime", "HTTPS/TLS (mTLS preferred)")
  Rel(authSystem, coldStorage, "Archives audit log records older than 1 year", "HTTPS/S3 API")

  Rel(resourceServer, authSystem, "Fetches JWKS public keys, calls /oauth/introspect for opaque token validation", "HTTPS/REST")
```

### Relationship Descriptions

| Relationship | Description |
|---|---|
| End User → Auth System | Non-technical users interact through a consuming application's UI. The consuming application calls Auth System REST endpoints on the user's behalf. |
| Tenant Admin → Auth System | Carlos calls admin REST endpoints (invite user, disable user, assign role, read audit log) scoped to his tenant. Requires a JWT with `admin` role. |
| Developer → Auth System | Priya's services integrate via OAuth 2.0. She registers clients, initiates authorization code flows, calls token introspection, and fetches JWKS for local JWT validation. |
| Super-Admin → Auth System | Jordan calls super-admin endpoints to provision tenants, rotate signing keys, and inspect cross-tenant audit data. Requires a special super-admin JWT with elevated privileges. |
| Auth System → Google OAuth | The system fetches Google's OIDC discovery document at startup, redirects users to Google for authentication, and exchanges codes for ID tokens. Per-tenant Google credentials are stored in Secrets Manager. |
| Auth System → Email Provider | Asynchronous calls to send transactional emails. Email sends must not block the request path. |
| Auth System → Secrets Manager | At startup and on rotation events, the system fetches signing keys and credentials from Vault. No secrets are stored in environment variables or config files. |
| Auth System → Cold Storage | A scheduled job exports audit log records older than 1 year from PostgreSQL to S3-compatible cold storage and tombstones or deletes the source rows. |
| Resource Server → Auth System | Downstream APIs validate JWTs by fetching public keys from `/.well-known/jwks.json`. For opaque tokens or active checks, they call `POST /oauth/introspect`. |

---

## 2. Container Diagram (C4 Level 2)

### Diagram

```mermaid
C4Container
  title Container Diagram — Authentication System

  Person(endUser, "End User", "Interacts via a consuming application UI")
  Person(tenantAdmin, "Tenant Admin", "Calls admin REST API")
  Person(developer, "Developer", "Integrates via OAuth 2.0")
  Person(superAdmin, "Super Admin", "Provisions tenants; platform ops")

  System_Boundary(authBoundary, "Authentication System") {
    Container(goAuthAPI, "Go Auth API", "Go 1.22 + Gin", "Core REST API server. Handles all auth flows, OAuth 2.0 authorization server, tenant management, RBAC, audit logging. Stateless; horizontally scalable.")
    ContainerDb(postgres, "PostgreSQL 16", "PostgreSQL", "Primary data store. Global schema holds tenant registry. Each tenant has a dedicated schema (schema-per-tenant, ADR-001). Connection pooling via pgxpool.")
    ContainerDb(redis, "Redis 7", "Redis", "Distributed cache and ephemeral store. Stores: rate limiting counters (sliding window), revoked token denylist (opaque refresh token hashes), session counters.")
    Container(secretsManager, "HashiCorp Vault", "Vault OSS / HCP Vault", "Dedicated secrets manager (ADR-009). Stores JWT signing keys (RS256 private/public key pairs), DB credentials, client secrets, email API keys. Accessed via mTLS.")
    Container(emailWorker, "Email Worker", "Go goroutine / async queue", "Sends transactional emails via email provider API. Decoupled from request path via an in-process channel or lightweight queue. Retries on failure.")
  }

  System_Ext(googleOAuth, "Google OAuth / OIDC", "External IdP")
  System_Ext(emailProvider, "Email Provider (Resend / SendGrid)", "Transactional email delivery")
  System_Ext(coldStorage, "Cold Storage (S3-compatible)", "Audit log archive")
  System_Ext(resourceServer, "Resource Servers", "Downstream APIs")

  Rel(endUser, goAuthAPI, "POST /auth/register, /auth/login, /auth/token/refresh, etc.", "HTTPS/REST JSON")
  Rel(tenantAdmin, goAuthAPI, "POST /admin/users/invite, GET /admin/audit-log, etc.", "HTTPS/REST JSON")
  Rel(developer, goAuthAPI, "GET /oauth/authorize, POST /oauth/token, GET /.well-known/jwks.json", "HTTPS/REST JSON")
  Rel(superAdmin, goAuthAPI, "POST /admin/tenants, GET /admin/tenants/:id", "HTTPS/REST JSON")

  Rel(goAuthAPI, postgres, "Reads/writes user, session, role, audit, OAuth data. Sets search_path per tenant per request.", "TCP / pgx driver / TLS")
  Rel(goAuthAPI, redis, "Increments rate limit counters, checks/stores token denylist, reads session counters.", "TCP / go-redis / TLS")
  Rel(goAuthAPI, secretsManager, "Fetches RS256 signing keys, DB credentials, client secrets at startup and on rotation.", "HTTPS mTLS / Vault SDK")
  Rel(goAuthAPI, emailWorker, "Publishes email send tasks (verification, reset, invite) to internal channel.", "In-process Go channel")
  Rel(emailWorker, emailProvider, "HTTP POST to transactional email API (Resend/SendGrid).", "HTTPS / REST API")
  Rel(goAuthAPI, googleOAuth, "Fetches OIDC discovery, validates ID tokens, exchanges authorization codes.", "HTTPS / OIDC")
  Rel(goAuthAPI, coldStorage, "Scheduled export: audit records > 1 year → S3 Parquet/JSONL.", "HTTPS / S3 API")
  Rel(resourceServer, goAuthAPI, "GET /.well-known/jwks.json (periodic key fetch), POST /oauth/introspect.", "HTTPS/REST JSON")
```

### Container Descriptions

| Container | Technology | Responsibility |
|---|---|---|
| Go Auth API | Go 1.22, Gin framework | Stateless HTTP API server. All business logic for auth, OAuth, tenant management, RBAC. Horizontally scalable behind a load balancer. |
| PostgreSQL 16 | PostgreSQL | Durable storage. Global `public` schema holds `tenants` table and super-admin audit log. Each tenant gets a dedicated schema (e.g., `tenant_acme`) with its own users, sessions, roles, audit log. |
| Redis 7 | Redis | Ephemeral distributed state. Rate limit sliding window counters, refresh token denylist (hashed tokens), concurrent session tracking. TTL-based self-cleanup. |
| HashiCorp Vault | Vault OSS or HCP Vault | Secrets manager per ADR-009. JWT RS256 key pairs, DB credentials, OAuth client secrets, API keys. Supports dynamic secrets and key rotation without application restart. |
| Email Worker | Go goroutine pool | Decouples email delivery from request path. Reads from a buffered Go channel; calls email provider API. Retries with exponential backoff. |

---

## 3. Component Diagram (C4 Level 3) — Go Auth API Internals

### Clean Architecture Layers

```mermaid
flowchart TD
    subgraph Transport["Transport Layer (internal/handler + internal/middleware)"]
        H_Auth["AuthHandler\n/auth/* endpoints"]
        H_Admin["AdminHandler\n/admin/* endpoints"]
        H_OAuth["OAuthHandler\n/oauth/* endpoints"]
        H_User["UserHandler\n/users/me endpoints"]
        H_Well["WellKnownHandler\n/.well-known/jwks.json"]
        MW_Tenant["TenantMiddleware\nExtracts tenant_id, sets search_path"]
        MW_Auth["AuthMiddleware\nJWT validation, role extraction"]
        MW_RateLimit["RateLimitMiddleware\nRedis sliding window"]
        MW_SecureHeaders["SecureHeadersMiddleware\nHSTS, CSP, X-Frame-Options"]
        MW_Logger["LoggerMiddleware\nStructured request logging"]
        MW_CORS["CORSMiddleware\nConfigurable per-tenant origins"]
    end

    subgraph Application["Application Layer (internal/service)"]
        S_Auth["AuthService\nregister, login, logout, refresh"]
        S_Email["EmailVerificationService\nverify, resend"]
        S_Password["PasswordService\nforgot, reset, change"]
        S_Session["SessionService\nrefresh rotation, family revocation"]
        S_Tenant["TenantService\nprovision schema, migrate, credentials"]
        S_RBAC["RBACService\nassign/unassign roles"]
        S_OAuth["OAuthService\nclient reg, authorize, token exchange, introspect"]
        S_Google["GoogleOAuthService\nOIDC flow, account linking"]
        S_Audit["AuditService\nappend events, query log"]
        S_Admin["AdminService\ninvite, disable, delete users"]
        S_JWT["JWTService\nissue, verify, rotate signing keys"]
    end

    subgraph Domain["Domain Layer (internal/domain)"]
        E_User["User entity\nid, email, password_hash, status, tenant_id"]
        E_Session["Session entity\nrefresh_token_hash, family_id, expires_at"]
        E_Role["Role entity\nid, name, tenant_id"]
        E_OAuthClient["OAuthClient entity\nclient_id, redirect_uris, scopes"]
        E_AuditEvent["AuditEvent entity\nevent_type, actor, ip, metadata"]
        E_Tenant["Tenant entity\nid, slug, schema_name, config"]
        VO_Password["PasswordHash value object\nArgon2id params"]
        VO_Token["Token value objects\nAccessToken, RefreshToken, AuthCode"]
        R_User["UserRepository interface"]
        R_Session["SessionRepository interface"]
        R_Role["RoleRepository interface"]
        R_OAuth["OAuthRepository interface"]
        R_Audit["AuditRepository interface"]
        R_Tenant["TenantRepository interface"]
    end

    subgraph Infrastructure["Infrastructure Layer (internal/infrastructure + internal/repository)"]
        PG_User["PostgresUserRepo\nimplements UserRepository"]
        PG_Session["PostgresSessionRepo\nimplements SessionRepository"]
        PG_Role["PostgresRoleRepo\nimplements RoleRepository"]
        PG_OAuth["PostgresOAuthRepo\nimplements OAuthRepository"]
        PG_Audit["PostgresAuditRepo\nimplements AuditRepository"]
        PG_Tenant["PostgresTenantRepo\nimplements TenantRepository"]
        PG_Pool["pgxpool.Pool\nPostgreSQL connection pool"]
        RC_Client["RedisClient\ngo-redis/v9"]
        RC_RateLimit["RedisRateLimiter\nsliding window EVAL script"]
        RC_Denylist["RedisDenylist\nrefresh token hash storage"]
        EXT_Vault["VaultClient\nhashicorp/vault-client-go"]
        EXT_Email["EmailClient\nResend API wrapper"]
        EXT_Google["GoogleOIDCClient\nOIDC discovery + token verification"]
        MIG_Runner["MigrationRunner\ngolang-migrate per-schema"]
    end

    %% Transport → Application
    H_Auth --> S_Auth
    H_Auth --> S_Email
    H_Auth --> S_Password
    H_Auth --> S_Session
    H_Auth --> S_Google
    H_Admin --> S_Admin
    H_Admin --> S_Tenant
    H_Admin --> S_RBAC
    H_Admin --> S_Audit
    H_OAuth --> S_OAuth
    H_User --> S_Auth
    H_Well --> S_JWT

    %% Middleware wires
    MW_Tenant --> PG_Pool
    MW_Auth --> S_JWT
    MW_RateLimit --> RC_RateLimit

    %% Application → Domain (uses domain entities and interfaces)
    S_Auth --> R_User
    S_Auth --> R_Session
    S_Auth --> S_JWT
    S_Auth --> S_Audit
    S_Session --> R_Session
    S_Session --> RC_Denylist
    S_Email --> R_User
    S_Password --> R_User
    S_Password --> R_Session
    S_Tenant --> R_Tenant
    S_Tenant --> MIG_Runner
    S_RBAC --> R_Role
    S_RBAC --> R_User
    S_OAuth --> R_OAuth
    S_Google --> EXT_Google
    S_Google --> R_User
    S_Audit --> R_Audit

    %% Domain → Infrastructure (dependency inversion)
    R_User --> PG_User
    R_Session --> PG_Session
    R_Role --> PG_Role
    R_OAuth --> PG_OAuth
    R_Audit --> PG_Audit
    R_Tenant --> PG_Tenant

    %% Infrastructure → external
    PG_User --> PG_Pool
    PG_Session --> PG_Pool
    PG_Role --> PG_Pool
    PG_OAuth --> PG_Pool
    PG_Audit --> PG_Pool
    PG_Tenant --> PG_Pool
    RC_RateLimit --> RC_Client
    RC_Denylist --> RC_Client
    EXT_Vault --> S_JWT
    EXT_Email --> S_Email
    EXT_Email --> S_Password
    EXT_Email --> S_Admin
```

### Layer Responsibilities

| Layer | Package | Responsibility |
|---|---|---|
| Transport | `internal/handler`, `internal/middleware` | HTTP request parsing, input validation, response serialization, middleware chain (auth, tenant routing, rate limiting, secure headers, logging). No business logic. |
| Application | `internal/service` | Use cases / application services. Orchestrates domain entities and calls repository interfaces. Owns transaction boundaries. No HTTP concerns. |
| Domain | `internal/domain` | Entities (User, Session, Role, OAuthClient, AuditEvent, Tenant), value objects (PasswordHash, Token), and repository interfaces. Pure Go — no framework dependencies. |
| Infrastructure | `internal/repository`, `internal/infrastructure` | Concrete implementations of domain repository interfaces. PostgreSQL queries (pgx), Redis commands, Vault SDK, email API client, Google OIDC client, migration runner. |

---

## 4. Tech Stack Decision Matrix

### 4.1 Backend Runtime: Go 1.22 + Gin

| Criterion | Go + Gin | Node.js + Express | Python + FastAPI | Java + Spring Boot |
|---|---|---|---|---|
| Performance (p95 login < 300ms) | Excellent — compiled binary, ~50MB RSS, goroutines for concurrency | Good — event loop; CPU-bound auth (Argon2id) blocks without worker threads | Good — async; Argon2 requires C extension | Good — JVM warmup penalty; high memory footprint |
| Compiled / Static Binary | Yes — single binary deployment | No | No | No (JAR + JVM) |
| Concurrency Model | Goroutines — lightweight; ideal for I/O-heavy auth flows | Event loop — single-threaded CPU bottleneck for Argon2id | GIL limits true parallelism | Thread pool — high memory per thread |
| Type Safety | Strong static typing — catches errors at compile time | Weak unless TypeScript (runtime overhead) | Gradual typing | Strong |
| Crypto / JWT ecosystem | `golang-jwt/jwt`, `crypto/rsa` — standard library, mature | `jose`, `jsonwebtoken` — mature | `python-jose` — mature | `nimbus-jose-jwt` — excellent |
| Deployment footprint | Tiny — ~10MB binary + scratch container | Moderate — node_modules, 200MB+ images | Moderate | Large — 200MB+ JVM |
| Team familiarity (assumed) | Required by PRD scope | Alternative | Alternative | Alternative |
| **Decision** | **Selected** | Rejected — CPU bottleneck on Argon2id | Rejected — GIL, no compiled binary | Rejected — JVM overhead, slower cold start |

**Framework: Gin** over Echo, Fiber, Chi because Gin has the largest Go HTTP framework ecosystem, mature middleware ecosystem, best documentation, and is the most widely understood for team onboarding. Chi is a strong alternative if middleware composability is preferred — acceptable substitution.

### 4.2 Database: PostgreSQL 16 with Schema-per-Tenant

| Criterion | Schema-per-Tenant (Selected) | Row-Level Security (RLS) | Database-per-Tenant | MongoDB |
|---|---|---|---|---|
| Isolation guarantee | Architectural — impossible to leak via missing WHERE | Policy-based — a misconfigured policy leaks data | Strongest — full DB isolation | Collection-level — requires app-layer filtering |
| Cross-tenant leak risk | Zero — wrong schema physically cannot return other tenant data | Non-zero — RLS policy bug leaks data | Zero | Non-zero |
| GDPR compliance (erasure) | Simple — drop or wipe schema | Complex — requires filtering across shared tables | Simple — drop database | Complex |
| Backup granularity | Per-schema pg_dump | Full DB only | Per-database | Per-collection |
| Migration complexity | Must run per-schema — mitigated by migration runner | Single migration for all tenants | Must run per-DB | Schema-less — different tradeoff |
| Operational cost at scale | Medium — schema count grows | Low | High — connection pool per DB | Low |
| Connection pooling | Shared pool; search_path set per request | Shared pool; SET ROLE per request | Separate pool per tenant DB | Shared |
| Audit / compliance boundary | Clean schema boundary for SOC2 | Shared table — less clean boundary | Cleanest | Less clear |
| **Decision** | **Selected per ADR-001** | Rejected — weaker isolation for security-critical system | Rejected — too expensive at pilot scale | Rejected — relational model better suited to auth |

### 4.3 Cache / Ephemeral Store: Redis 7

| Criterion | Redis 7 | In-Memory (process) | Memcached | DynamoDB |
|---|---|---|---|---|
| Distributed rate limiting | Yes — atomic EVAL scripts across instances | No — per-instance only; breaks horizontal scaling | Partial — no Lua scripting | Yes — but higher latency |
| Token denylist | Yes — SETEX with TTL, hashed tokens | No — lost on restart | Partial — no expiry per key in older versions | Yes — but cold-start latency |
| Atomic operations | Yes — INCR, SETNX, EVAL | Yes — in-process only | Limited | Yes — conditional writes |
| p95 < 5ms for rate check | Yes — local Redis: ~0.5ms | Yes — but not distributed | Yes | No — ~5-20ms typical |
| Data persistence | Optional (RDB/AOF) — not needed for ephemeral data | None | None | Yes |
| Operational simplicity | Moderate — requires Redis instance | Trivial — no infra | Moderate | High — managed but complex IAM |
| **Decision** | **Selected** | Rejected — breaks horizontal scaling | Rejected — no Lua scripting for atomic sliding window | Rejected — latency too high for PERF-05 |

### 4.4 Token Strategy: JWT RS256 + Opaque Refresh Tokens

| Criterion | JWT RS256 + Opaque Refresh (Selected) | HS256 JWT | Session Cookies (opaque only) | Symmetric JWT (HS256) |
|---|---|---|---|---|
| Resource server validation | Local — resource servers verify with public key (JWKS), no API call | Local — but requires sharing secret with all resource servers | Remote — every request hits auth server | Local — but shared secret model |
| Secret exposure risk | Private key stays in auth service only | Shared secret across all services | N/A — session stored server-side | Shared secret = breach of one service = all compromised |
| Multi-tenant isolation | `tenant_id` in JWT claims; verified by signature | Same — but shared secret risk | Session table must be tenant-scoped | Same risk as HS256 |
| Revocation of access tokens | Not immediate (15-min TTL accepted trade-off, documented) | Same | Immediate | Same as RS256 |
| Refresh token security | Opaque, hashed in DB, rotated on every use, family revocation | N/A | Cookie-based — SameSite, HttpOnly | N/A |
| Key rotation | Zero-downtime via JWKS — publish new key, keep old for TTL duration | Requires rotating shared secret across all services simultaneously | N/A | Same as HS256 |
| Standards compliance | RFC 7519 (JWT), RFC 7517 (JWK), RFC 7662 (introspection) | RFC 7519 only | RFC 6265 | RFC 7519 only |
| **Decision** | **Selected per ADR-004** | Rejected — shared secret model incompatible with multi-tenant resource server model | Rejected — stateful, server-side session does not scale and conflicts with OAuth 2.0 design | Rejected per ADR-004 |

### 4.5 OAuth Library Recommendation

**Recommendation: `ory/fosite`** (github.com/ory/fosite)

| Criterion | ory/fosite | go-oauth2/oauth2 | Custom implementation |
|---|---|---|---|
| RFC compliance | RFC 6749, 7636 (PKCE), 7662 (introspection), 7009 (revocation) | Partial | Risk of non-compliance |
| PKCE S256 support | Yes — built-in, mandatory enforcement possible | Limited | Must implement |
| Client Credentials grant | Yes | Yes | Must implement |
| Authorization Code + PKCE | Yes | Partial | Must implement |
| Active maintenance | Yes — Ory is a major OSS auth company | Moderate | N/A |
| Integration with custom storage | Yes — storage adapter interface | Limited | N/A |
| **Decision** | **Selected** | Rejected — incomplete PKCE/introspection | Rejected per PRD recommendation and Sprint 5 risk note |

### 4.6 Secrets Manager Recommendation

**Recommendation: HashiCorp Vault (HCP Vault Dedicated or self-hosted Vault OSS)**

| Criterion | HashiCorp Vault | AWS Secrets Manager | GCP Secret Manager | Azure Key Vault |
|---|---|---|---|---|
| Cloud-agnostic | Yes — runs anywhere | AWS-only | GCP-only | Azure-only |
| Dynamic secrets | Yes — generates DB credentials on-demand | No | No | No |
| Key rotation (JWT signing keys) | Yes — Transit secrets engine; zero-downtime rotation | Yes — manual rotation | Yes — manual rotation | Yes |
| Audit trail | Yes — detailed audit log of every secret access | Yes | Yes | Yes |
| mTLS auth from Go | Yes — Vault SDK + AppRole or K8s auth | IAM role (if on AWS) | Workload Identity | Managed Identity |
| Pilot cost | OSS free; HCP Dev tier affordable | Per-secret pricing | Per-secret pricing | Per-secret pricing |
| Vendor lock-in risk | Low — self-hostable | High — AWS-specific SDKs | High | High |
| **Decision** | **Selected per ADR-009** | Acceptable if deploying on AWS — lower ops burden. Use aws-sdk-go-v2/secretsmanager. | Acceptable if deploying on GCP. | Acceptable if deploying on Azure. |

**Deployment-specific note:** If the team deploys on AWS (Fly.io or ECS), AWS Secrets Manager is acceptable as a simpler alternative that still satisfies ADR-009. Vault is preferred for cloud-agnosticism.

### 4.7 Email Provider Recommendation

**Recommendation: Resend (resend.com)**

| Criterion | Resend | SendGrid | AWS SES | Postmark |
|---|---|---|---|---|
| Developer experience | Excellent — modern API, simple Go SDK | Good — complex legacy API | Poor — complex IAM | Good |
| Deliverability | High | High | High (with configuration) | Excellent (transactional focused) |
| Go SDK | `resend-go` — official | `sendgrid-go` — official | `aws-sdk-go-v2/ses` | Third-party |
| Cost (pilot scale) | Free tier: 3,000/month; $20/month for 50K | $19.95/month for 50K | $0.10/1000 (very cheap at scale) | $15/month for 10K |
| Template support | Yes | Yes | Yes | Yes |
| Webhook events | Yes | Yes | Yes (SNS) | Yes |
| **Decision** | **Recommended** | Strong alternative — more mature at high volume | Best cost at scale — steeper setup | Strong alternative for pure transactional use |

### 4.8 Deployment Platform Recommendation

**Recommendation: Fly.io (pilot) → AWS ECS Fargate (production scale)**

| Criterion | Fly.io | Railway | AWS ECS Fargate | GCP Cloud Run | Self-hosted K8s |
|---|---|---|---|---|---|
| Complexity for pilot | Low — `fly deploy` from CLI | Very low | Medium — IAM, VPC, task definitions | Low-medium | High |
| Go binary deployment | Excellent — Dockerfile-based | Excellent | Excellent | Excellent | Excellent |
| PostgreSQL managed | Fly Postgres (built-in) | Railway Postgres | RDS PostgreSQL | Cloud SQL | Self-managed |
| Redis managed | Upstash Redis (partner) | Railway Redis | ElastiCache | Memorystore | Self-managed |
| Secrets integration | Fly Secrets (env-based, use Vault for ADR-009 compliance) | Railway Secrets | AWS Secrets Manager native | GCP Secret Manager native | Vault |
| Horizontal scaling | Yes — replicas | Limited | Yes — task count | Yes — instances | Yes |
| Cost (pilot 3 instances) | ~$50–$100/month | ~$30–$80/month | ~$100–$200/month | ~$80–$150/month | High ops cost |
| Production readiness | Medium — suitable for pilot | Low — less mature | High — enterprise grade | High | High — ops heavy |
| **Decision** | **Fly.io for pilot** | Rejected — less mature at scale | Recommended for production scale-up | Strong alternative | Rejected — ops overhead not justified for pilot |

---

## 5. Database Schema (ERD)

### Schema Organization

- **`public` schema** (global, platform-level): `tenants`, `global_audit_log`
- **`tenant_{slug}` schema** (one per tenant, e.g., `tenant_acme`): all tables below are created inside this schema for every tenant

### 5.1 Global Schema ERD

```mermaid
erDiagram
    tenants {
        uuid id PK "gen_random_uuid()"
        varchar(100) slug UK "URL-safe identifier e.g. acme-corp; immutable after creation"
        varchar(255) name "Display name"
        varchar(63) schema_name UK "PostgreSQL schema name e.g. tenant_acme_corp"
        varchar(255) admin_email "Initial admin email address"
        varchar(50) status "active | suspended | deleted"
        jsonb config "Tenant-level config: session_ttl_seconds, lockout_threshold, lockout_duration_seconds, password_policy{}, allowed_redirect_uris[], google_client_id"
        timestamptz created_at "DEFAULT now()"
        timestamptz updated_at "DEFAULT now()"
        timestamptz deleted_at "NULL unless soft-deleted"
    }

    global_audit_log {
        uuid id PK "gen_random_uuid()"
        uuid tenant_id FK "References tenants.id; NULL for platform-level events"
        varchar(100) event_type "TENANT_PROVISIONED | TENANT_SUSPENDED | SIGNING_KEY_ROTATED | SUPER_ADMIN_LOGIN | etc."
        uuid actor_id "Super-admin user ID performing the action"
        varchar(45) actor_ip "IPv4 or IPv6"
        jsonb metadata "Arbitrary event-specific data"
        timestamptz occurred_at "DEFAULT now(); indexed"
        boolean archived "false until cold-storage export job marks it"
    }

    tenants ||--o{ global_audit_log : "tenant_id references"
```

**Global Schema Indexes:**
```sql
-- tenants
CREATE UNIQUE INDEX idx_tenants_slug ON public.tenants(slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_tenants_schema_name ON public.tenants(schema_name);
CREATE INDEX idx_tenants_status ON public.tenants(status);

-- global_audit_log
CREATE INDEX idx_global_audit_occurred_at ON public.global_audit_log(occurred_at DESC);
CREATE INDEX idx_global_audit_tenant_id ON public.global_audit_log(tenant_id);
CREATE INDEX idx_global_audit_event_type ON public.global_audit_log(event_type);
```

### 5.2 Tenant Schema Template ERD

All tables below exist inside `tenant_{slug}` schema. Foreign keys reference tables within the same schema.

```mermaid
erDiagram
    users {
        uuid id PK "gen_random_uuid()"
        varchar(255) email UK "Lowercased; unique within tenant schema"
        varchar(255) password_hash "Argon2id hash; NULL if social-only account"
        varchar(50) status "unverified | active | disabled | deleted"
        varchar(255) first_name
        varchar(255) last_name
        boolean mfa_enabled "DEFAULT false"
        varchar(255) mfa_totp_secret "Encrypted at rest; NULL if MFA not set up"
        integer failed_login_count "DEFAULT 0; reset on successful login"
        timestamptz locked_until "NULL if not locked; set on lockout"
        timestamptz email_verified_at "NULL until email verified"
        timestamptz last_login_at
        timestamptz created_at "DEFAULT now()"
        timestamptz updated_at "DEFAULT now()"
        timestamptz deleted_at "NULL unless soft-deleted (GDPR tombstone)"
    }

    roles {
        uuid id PK "gen_random_uuid()"
        varchar(100) name UK "e.g. admin, user, editor; unique within tenant"
        varchar(500) description
        boolean is_system "true for built-in roles (admin, user); cannot be deleted"
        timestamptz created_at "DEFAULT now()"
    }

    user_roles {
        uuid id PK "gen_random_uuid()"
        uuid user_id FK "References users.id"
        uuid role_id FK "References roles.id"
        uuid assigned_by "User ID of admin who assigned role"
        timestamptz assigned_at "DEFAULT now()"
    }

    sessions {
        uuid id PK "gen_random_uuid()"
        uuid user_id FK "References users.id"
        varchar(512) refresh_token_hash UK "SHA-256 hash of the opaque refresh token"
        uuid family_id "Groups tokens for family revocation on reuse detection"
        varchar(45) ip_address "IPv4 or IPv6 of client at time of issuance"
        varchar(500) user_agent
        timestamptz issued_at "DEFAULT now()"
        timestamptz expires_at "Absolute session expiry; configurable per tenant (default now()+24h)"
        timestamptz last_used_at "Updated on every valid token refresh"
        timestamptz revoked_at "NULL unless revoked; set on logout or suspicious reuse"
        boolean is_revoked "DEFAULT false; indexed for fast denylist check"
    }

    password_reset_tokens {
        uuid id PK "gen_random_uuid()"
        uuid user_id FK "References users.id"
        varchar(512) token_hash UK "SHA-256 hash of the opaque reset token"
        timestamptz expires_at "now() + 1 hour"
        boolean used "DEFAULT false; set true on first use"
        timestamptz used_at
        timestamptz created_at "DEFAULT now()"
    }

    email_verification_tokens {
        uuid id PK "gen_random_uuid()"
        uuid user_id FK "References users.id"
        varchar(512) token_hash UK "SHA-256 hash of the opaque verification token"
        timestamptz expires_at "now() + 24 hours"
        boolean used "DEFAULT false"
        timestamptz used_at
        timestamptz created_at "DEFAULT now()"
    }

    oauth_clients {
        uuid id PK "gen_random_uuid()"
        varchar(100) client_id UK "Public identifier; non-secret"
        varchar(512) client_secret_hash "Argon2id hash; NULL for public clients"
        varchar(255) name "Human-readable client name"
        text[] redirect_uris "Strict matching; no wildcards"
        text[] allowed_scopes "e.g. openid, profile, email, offline_access"
        varchar(50) client_type "confidential | public"
        text[] allowed_grant_types "authorization_code | client_credentials | refresh_token"
        boolean is_active "DEFAULT true"
        timestamptz created_at "DEFAULT now()"
        timestamptz updated_at "DEFAULT now()"
    }

    oauth_authorization_codes {
        uuid id PK "gen_random_uuid()"
        uuid client_id FK "References oauth_clients.id"
        uuid user_id FK "References users.id; NULL for client_credentials"
        varchar(512) code_hash UK "SHA-256 hash of the authorization code"
        varchar(512) code_challenge "PKCE S256 challenge (BASE64URL(SHA256(verifier)))"
        varchar(10) code_challenge_method "S256 only"
        varchar(2048) redirect_uri "Must match exactly at exchange time"
        text[] scopes "Requested scopes"
        varchar(255) state "Client-provided CSRF token; stored for verification"
        boolean used "DEFAULT false; set true on exchange; second use = revoke all"
        timestamptz expires_at "now() + 10 minutes"
        timestamptz created_at "DEFAULT now()"
    }

    oauth_social_accounts {
        uuid id PK "gen_random_uuid()"
        uuid user_id FK "References users.id"
        varchar(50) provider "google | microsoft"
        varchar(255) provider_user_id UK "Provider's unique user identifier"
        varchar(255) provider_email "Email from provider at time of linking"
        jsonb provider_metadata "Raw profile data from provider"
        timestamptz linked_at "DEFAULT now()"
    }

    audit_log {
        uuid id PK "gen_random_uuid()"
        varchar(100) event_type "LOGIN_SUCCESS | LOGIN_FAILURE | LOGOUT | LOGOUT_ALL | PASSWORD_RESET_REQUESTED | PASSWORD_RESET_COMPLETED | PASSWORD_CHANGED | EMAIL_VERIFIED | ACCOUNT_LOCKED | TOKEN_REFRESHED | SUSPICIOUS_TOKEN_REUSE | ROLE_ASSIGNED | ROLE_UNASSIGNED | USER_INVITED | USER_DISABLED | OAUTH_CLIENT_CREATED | OAUTH_CODE_ISSUED | OAUTH_TOKEN_ISSUED | USER_DELETED"
        uuid actor_id "User ID performing the action; NULL for system events"
        varchar(45) actor_ip "IPv4 or IPv6"
        varchar(500) actor_user_agent
        uuid target_user_id "User affected by the action; may differ from actor_id"
        jsonb metadata "Event-specific data: scope, client_id, role_name, reason, etc."
        timestamptz occurred_at "DEFAULT now(); indexed; append-only — no UPDATE/DELETE at application layer"
        boolean archived "DEFAULT false; set by cold-archive job"
    }

    tenant_config {
        uuid id PK "gen_random_uuid()"
        varchar(100) key UK "Config key e.g. session_ttl_seconds, lockout_threshold"
        text value "Config value (serialized)"
        varchar(50) data_type "integer | boolean | string | json"
        varchar(500) description
        timestamptz updated_at "DEFAULT now()"
        uuid updated_by "User ID of admin who changed this config"
    }

    schema_migrations {
        bigint version PK "Migration version number (timestamp-based)"
        varchar(255) name "Human-readable migration name"
        boolean dirty "DEFAULT false; true if migration failed mid-run"
        timestamptz applied_at "DEFAULT now()"
    }

    users ||--o{ user_roles : "has"
    roles ||--o{ user_roles : "assigned via"
    users ||--o{ sessions : "has"
    users ||--o{ password_reset_tokens : "has"
    users ||--o{ email_verification_tokens : "has"
    users ||--o{ oauth_social_accounts : "has"
    oauth_clients ||--o{ oauth_authorization_codes : "issues"
    users ||--o{ oauth_authorization_codes : "authorizes"
    users ||--o{ audit_log : "actor or target"
```

**Tenant Schema Indexes:**
```sql
-- users (critical hot-path indexes)
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_locked_until ON users(locked_until) WHERE locked_until IS NOT NULL;

-- user_roles (composite unique — one role per user)
CREATE UNIQUE INDEX idx_user_roles_user_role ON user_roles(user_id, role_id);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);

-- sessions (critical — token refresh hot path)
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_family_id ON sessions(family_id);
CREATE INDEX idx_sessions_is_revoked ON sessions(is_revoked) WHERE is_revoked = false;
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- password_reset_tokens
CREATE UNIQUE INDEX idx_prt_token_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_prt_user_id ON password_reset_tokens(user_id);

-- email_verification_tokens
CREATE UNIQUE INDEX idx_evt_token_hash ON email_verification_tokens(token_hash);
CREATE INDEX idx_evt_user_id ON email_verification_tokens(user_id);

-- oauth_clients
CREATE UNIQUE INDEX idx_oauth_clients_client_id ON oauth_clients(client_id);

-- oauth_authorization_codes
CREATE UNIQUE INDEX idx_oac_code_hash ON oauth_authorization_codes(code_hash);
CREATE INDEX idx_oac_client_id ON oauth_authorization_codes(client_id);

-- oauth_social_accounts
CREATE UNIQUE INDEX idx_osa_provider_user ON oauth_social_accounts(provider, provider_user_id);

-- audit_log (compliance reads + archiving)
CREATE INDEX idx_audit_occurred_at ON audit_log(occurred_at DESC);
CREATE INDEX idx_audit_actor_id ON audit_log(actor_id);
CREATE INDEX idx_audit_target_user_id ON audit_log(target_user_id);
CREATE INDEX idx_audit_event_type ON audit_log(event_type);
CREATE INDEX idx_audit_archived ON audit_log(archived) WHERE archived = false;
```

---

## 6. API Design

### Standard Error Response Format

All error responses use this envelope. Engineers must not deviate from this format.

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "The email or password is incorrect.",
    "details": [],
    "request_id": "req_01HZ..."
  }
}
```

| Field | Type | Description |
|---|---|---|
| `code` | string | Machine-readable uppercase snake_case error code |
| `message` | string | Human-readable message safe to display (no internal details) |
| `details` | array | Field-level validation errors: `[{"field": "email", "issue": "invalid format"}]` |
| `request_id` | string | Trace ID from X-Request-ID header (for log correlation) |

### Standard Success Envelope

Success responses return data directly (no wrapper envelope) unless noted. HTTP status code communicates success/failure.

### API Versioning

All endpoints are prefixed with `/api/v1`. Examples: `POST /api/v1/auth/register`. The `/api/v1` prefix is omitted from endpoint paths below for readability but must be present in implementation.

### Tenant Identification

Tenant is identified per request by the `X-Tenant-ID` header (UUID format). This is validated against the `tenants` table in the global schema. For authenticated endpoints, the tenant_id in the JWT must match the `X-Tenant-ID` header. For unauthenticated endpoints (register, login, forgot-password), only the header is used.

---

### 6.1 POST /auth/register

**Auth:** None
**Description:** Register a new user account. Triggers email verification.

**Request Body:**
```json
{
  "email": "maya@example.com",
  "password": "S3cur3P@ssw0rd!",
  "first_name": "Maya",
  "last_name": "Johnson"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `email` | string | Yes | Valid RFC 5322 email; lowercased before storage |
| `password` | string | Yes | Must satisfy tenant password_policy config (min_length, require_uppercase, require_number, require_special) |
| `first_name` | string | Yes | 1–100 characters |
| `last_name` | string | Yes | 1–100 characters |

**Success Response — 201 Created:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "maya@example.com",
  "status": "unverified",
  "message": "Registration successful. Please check your email to verify your account."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 409 Conflict | `EMAIL_ALREADY_EXISTS` | Email already registered in this tenant (do not reveal account status) |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Invalid email format, or password fails complexity rules. `details[]` contains field-specific messages. |
| 400 Bad Request | `INVALID_REQUEST` | Missing required fields |
| 503 Service Unavailable | `SERVICE_UNAVAILABLE` | DB unavailable |

---

### 6.2 POST /auth/verify-email

**Auth:** None
**Description:** Verifies the user's email using a single-use token sent by email.

**Request Body:**
```json
{
  "token": "vt_aBcDeFgHiJkLmNoPqRsTuVwXyZ"
}
```

**Success Response — 200 OK:**
```json
{
  "message": "Email verified successfully. You may now log in."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 400 Bad Request | `TOKEN_INVALID` | Token not found or already used |
| 400 Bad Request | `TOKEN_EXPIRED` | Token TTL exceeded. Response includes `"resend_available": true` |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Token field missing or malformed |

---

### 6.3 POST /auth/resend-verification

**Auth:** None
**Description:** Resends the email verification link. Previous token is invalidated. Returns 200 regardless of whether email exists (anti-enumeration).

**Request Body:**
```json
{
  "email": "maya@example.com"
}
```

**Success Response — 200 OK (always, even if email not found):**
```json
{
  "message": "If this email is registered and unverified, a new verification link has been sent."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 429 Too Many Requests | `RATE_LIMIT_EXCEEDED` | Rate limit on resend (e.g., max 3 resends per hour per email) |

---

### 6.4 POST /auth/login

**Auth:** None
**Description:** Authenticates a user and issues JWT access token + opaque refresh token.

**Request Body:**
```json
{
  "email": "maya@example.com",
  "password": "S3cur3P@ssw0rd!"
}
```

**Success Response — 200 OK:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0yMDI2MDIyNyJ9...",
  "refresh_token": "rt_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_expires_in": 86400
}
```

**JWT Payload Example:**
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "iss": "https://auth.example.com",
  "aud": ["https://api.example.com"],
  "exp": 1740000900,
  "iat": 1740000000,
  "tenant_id": "acme-corp",
  "roles": ["admin", "user"],
  "jti": "jwt_01HZ..."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 401 Unauthorized | `INVALID_CREDENTIALS` | Wrong email or password — message must not distinguish which |
| 403 Forbidden | `EMAIL_NOT_VERIFIED` | Account exists but email not verified |
| 403 Forbidden | `ACCOUNT_DISABLED` | Account disabled by admin |
| 403 Forbidden | `ACCOUNT_LOCKED` | Account locked due to too many failed attempts — do NOT reveal lockout details |
| 429 Too Many Requests | `RATE_LIMIT_EXCEEDED` | IP or user rate limit hit. `Retry-After` header included. |

---

### 6.5 POST /auth/token/refresh

**Auth:** None (refresh token is the credential)
**Description:** Exchanges a refresh token for a new access token + refresh token pair. Old token is immediately invalidated (rotation). Reuse of a rotated token triggers family revocation.

**Request Body:**
```json
{
  "refresh_token": "rt_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ"
}
```

**Success Response — 200 OK:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0yMDI2MDIyNyJ9...",
  "refresh_token": "rt_XyZaBcDeFgHiJkLmNoPqRsTuVw_02HZ",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_expires_in": 86400
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 401 Unauthorized | `INVALID_REFRESH_TOKEN` | Token not found, already used, or revoked |
| 401 Unauthorized | `REFRESH_TOKEN_EXPIRED` | Absolute session expiry exceeded |
| 401 Unauthorized | `SUSPICIOUS_TOKEN_REUSE` | Previously rotated token presented; entire family revoked; user must re-login |

---

### 6.6 POST /auth/logout

**Auth:** Bearer JWT (access token)
**Description:** Revokes the specific refresh token presented. Access token remains valid until its 15-min TTL (documented trade-off).

**Request Body:**
```json
{
  "refresh_token": "rt_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ"
}
```

**Success Response — 200 OK (idempotent — 200 even if token already expired):**
```json
{
  "message": "Logged out successfully."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 401 Unauthorized | `UNAUTHORIZED` | Access token missing or invalid |

---

### 6.7 POST /auth/logout/all

**Auth:** Bearer JWT (access token)
**Description:** Revokes all refresh tokens for the authenticated user across all devices.

**Request Body:** None

**Success Response — 200 OK:**
```json
{
  "message": "All sessions revoked successfully.",
  "sessions_revoked": 3
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 401 Unauthorized | `UNAUTHORIZED` | Access token missing or invalid |

---

### 6.8 POST /auth/forgot-password

**Auth:** None
**Description:** Initiates password reset. Response is always identical regardless of whether email exists (anti-enumeration). Response timing must be constant via async email send.

**Request Body:**
```json
{
  "email": "maya@example.com"
}
```

**Success Response — 200 OK (always):**
```json
{
  "message": "If this email is registered, you will receive a password reset link shortly."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 429 Too Many Requests | `RATE_LIMIT_EXCEEDED` | Rate limit: max 3 reset requests per email per hour |

---

### 6.9 POST /auth/reset-password

**Auth:** None
**Description:** Resets user password using a valid reset token. All existing sessions are revoked on success.

**Request Body:**
```json
{
  "token": "prt_aBcDeFgHiJkLmNoPqRsTuVwXyZ",
  "new_password": "N3wS3cur3P@ssw0rd!"
}
```

**Success Response — 200 OK:**
```json
{
  "message": "Password reset successful. All existing sessions have been revoked. Please log in again."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 400 Bad Request | `TOKEN_INVALID` | Token not found or already used |
| 400 Bad Request | `TOKEN_EXPIRED` | Reset token expired (1-hour TTL) |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | New password fails complexity rules |

---

### 6.10 POST /admin/tenants

**Auth:** Super-admin JWT (role: `super_admin`)
**Description:** Provisions a new tenant. Creates PostgreSQL schema, applies migrations, seeds default roles, issues client credentials, sends invitation email.

**Request Body:**
```json
{
  "name": "Acme Corporation",
  "slug": "acme-corp",
  "admin_email": "carlos@acme.com",
  "config": {
    "session_ttl_seconds": 86400,
    "lockout_threshold": 5,
    "lockout_duration_seconds": 900,
    "password_policy": {
      "min_length": 12,
      "require_uppercase": true,
      "require_number": true,
      "require_special": true
    }
  }
}
```

**Success Response — 201 Created:**
```json
{
  "tenant_id": "tenant-uuid-here",
  "slug": "acme-corp",
  "schema_name": "tenant_acme_corp",
  "client_id": "client_acme_corp_01HZ",
  "client_secret": "cs_SHOWN_ONCE_ONLY_aBcDeFgHiJkLmNoPqRsTuVwXyZ",
  "message": "Tenant provisioned. Invitation email sent to carlos@acme.com. Store client_secret securely — it will not be shown again."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 409 Conflict | `TENANT_ALREADY_EXISTS` | Slug already in use |
| 403 Forbidden | `FORBIDDEN` | Caller is not super-admin |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Invalid slug format or missing fields |

---

### 6.11 GET /admin/tenants/:id

**Auth:** Super-admin JWT
**Description:** Retrieves tenant details by tenant UUID.

**Path Parameter:** `id` — tenant UUID

**Success Response — 200 OK:**
```json
{
  "tenant_id": "tenant-uuid-here",
  "name": "Acme Corporation",
  "slug": "acme-corp",
  "schema_name": "tenant_acme_corp",
  "status": "active",
  "admin_email": "carlos@acme.com",
  "config": { "session_ttl_seconds": 86400, "lockout_threshold": 5 },
  "created_at": "2026-03-15T10:00:00Z"
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 404 Not Found | `TENANT_NOT_FOUND` | No tenant with this ID |
| 403 Forbidden | `FORBIDDEN` | Not super-admin |

---

### 6.12 POST /admin/users/invite

**Auth:** Bearer JWT (role: `admin` within tenant)
**Description:** Invites a new user to the tenant by email. Creates an unverified account and sends an invitation email with a setup link.

**Request Body:**
```json
{
  "email": "newuser@acme.com",
  "first_name": "New",
  "last_name": "User",
  "roles": ["user"]
}
```

**Success Response — 201 Created:**
```json
{
  "user_id": "user-uuid-here",
  "email": "newuser@acme.com",
  "status": "unverified",
  "message": "Invitation sent to newuser@acme.com."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 409 Conflict | `EMAIL_ALREADY_EXISTS` | Email already registered in this tenant |
| 403 Forbidden | `FORBIDDEN` | Caller does not have admin role |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Invalid email or unknown role names |

---

### 6.13 PUT /admin/users/:id/disable

**Auth:** Bearer JWT (role: `admin`)
**Description:** Disables a user account. All active sessions are revoked immediately.

**Path Parameter:** `id` — user UUID

**Request Body:** None

**Success Response — 200 OK:**
```json
{
  "user_id": "user-uuid-here",
  "status": "disabled",
  "sessions_revoked": 2,
  "message": "User disabled and all sessions revoked."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 404 Not Found | `USER_NOT_FOUND` | User not found in this tenant |
| 403 Forbidden | `FORBIDDEN` | Not admin; or admin attempting to disable themselves |
| 409 Conflict | `USER_ALREADY_DISABLED` | User already disabled |

---

### 6.14 DELETE /admin/users/:id

**Auth:** Bearer JWT (role: `admin`)
**Description:** Soft-deletes a user. Sets `deleted_at`, revokes all sessions, anonymizes audit log entries (GDPR). Hard delete is not performed at the application layer.

**Path Parameter:** `id` — user UUID

**Success Response — 200 OK:**
```json
{
  "message": "User deleted. PII has been anonymized per GDPR requirements."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 404 Not Found | `USER_NOT_FOUND` | User not found in this tenant |
| 403 Forbidden | `FORBIDDEN` | Not admin |

---

### 6.15 POST /admin/roles

**Auth:** Bearer JWT (role: `admin`)
**Description:** Creates a custom role within the tenant.

**Request Body:**
```json
{
  "name": "editor",
  "description": "Can edit content but not manage users."
}
```

**Success Response — 201 Created:**
```json
{
  "role_id": "role-uuid-here",
  "name": "editor",
  "description": "Can edit content but not manage users.",
  "is_system": false
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 409 Conflict | `ROLE_ALREADY_EXISTS` | Role name already exists in this tenant |
| 403 Forbidden | `FORBIDDEN` | Not admin |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Invalid role name format |

---

### 6.16 POST /admin/users/:id/roles

**Auth:** Bearer JWT (role: `admin`)
**Description:** Assigns a role to a user within the tenant.

**Path Parameter:** `id` — user UUID

**Request Body:**
```json
{
  "role_id": "role-uuid-here"
}
```

**Success Response — 200 OK:**
```json
{
  "user_id": "user-uuid-here",
  "role_id": "role-uuid-here",
  "role_name": "editor",
  "message": "Role assigned successfully."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 404 Not Found | `USER_NOT_FOUND` | User not found in tenant |
| 404 Not Found | `ROLE_NOT_FOUND` | Role not found in tenant |
| 409 Conflict | `ROLE_ALREADY_ASSIGNED` | User already has this role |
| 403 Forbidden | `FORBIDDEN` | Not admin |

---

### 6.17 DELETE /admin/users/:id/roles/:roleId

**Auth:** Bearer JWT (role: `admin`)
**Description:** Unassigns a role from a user.

**Path Parameters:** `id` — user UUID, `roleId` — role UUID

**Success Response — 200 OK:**
```json
{
  "message": "Role unassigned successfully."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 404 Not Found | `USER_NOT_FOUND` or `ROLE_NOT_FOUND` | User or role not found |
| 404 Not Found | `ROLE_NOT_ASSIGNED` | User does not have this role |
| 403 Forbidden | `FORBIDDEN` | Not admin |

---

### 6.18 GET /admin/audit-log

**Auth:** Bearer JWT (role: `admin`)
**Description:** Returns paginated audit log entries scoped to the authenticated tenant.

**Query Parameters:**
| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | integer | 1 | Page number |
| `page_size` | integer | 50 | Max 200 |
| `event_type` | string | all | Filter by event type |
| `actor_id` | uuid | all | Filter by actor user ID |
| `target_user_id` | uuid | all | Filter by target user |
| `from` | ISO8601 | 30 days ago | Start of time range |
| `to` | ISO8601 | now | End of time range |

**Success Response — 200 OK:**
```json
{
  "data": [
    {
      "id": "audit-uuid",
      "event_type": "LOGIN_SUCCESS",
      "actor_id": "user-uuid",
      "actor_ip": "203.0.113.45",
      "target_user_id": "user-uuid",
      "metadata": {},
      "occurred_at": "2026-03-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 1247,
    "total_pages": 25
  }
}
```

---

### 6.19 POST /oauth/clients

**Auth:** Bearer JWT (role: `admin`)
**Description:** Registers a new OAuth 2.0 client for the tenant.

**Request Body:**
```json
{
  "name": "My Application",
  "client_type": "confidential",
  "redirect_uris": ["https://myapp.example.com/callback"],
  "allowed_scopes": ["openid", "profile", "email", "offline_access"],
  "allowed_grant_types": ["authorization_code", "refresh_token"]
}
```

**Success Response — 201 Created:**
```json
{
  "client_id": "client_01HZ_aBcDeF",
  "client_secret": "cs_SHOWN_ONCE_aBcDeFgHiJkLmNoPqRsTuVwXyZ",
  "name": "My Application",
  "client_type": "confidential",
  "redirect_uris": ["https://myapp.example.com/callback"],
  "allowed_scopes": ["openid", "profile", "email", "offline_access"],
  "message": "Store client_secret securely — it will not be shown again."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Invalid redirect URI, unknown scope |
| 403 Forbidden | `FORBIDDEN` | Not admin |

---

### 6.20 GET /oauth/authorize

**Auth:** Bearer JWT (user must be logged in — passed as Authorization header or query param `access_token`)
**Description:** Validates an OAuth 2.0 authorization request. Returns authorization code in JSON (API-only; consuming app handles redirect). PKCE S256 mandatory.

**Query Parameters:**
| Parameter | Required | Description |
|---|---|---|
| `response_type` | Yes | Must be `code` |
| `client_id` | Yes | Registered client ID |
| `redirect_uri` | Yes | Must exactly match registered redirect URI |
| `scope` | Yes | Space-separated requested scopes |
| `state` | Yes | CSRF token (mandatory) |
| `code_challenge` | Yes | BASE64URL(SHA256(code_verifier)) |
| `code_challenge_method` | Yes | Must be `S256` — `plain` is rejected |

**Success Response — 200 OK:**
```json
{
  "code": "ac_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ",
  "state": "client-provided-state-value",
  "redirect_uri": "https://myapp.example.com/callback",
  "expires_in": 600
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 400 Bad Request | `INVALID_CLIENT` | client_id not found — do NOT redirect |
| 400 Bad Request | `INVALID_REDIRECT_URI` | redirect_uri mismatch — do NOT redirect (prevents open redirect) |
| 400 Bad Request | `INVALID_REQUEST` | Missing state, missing code_challenge, or code_challenge_method=plain |
| 400 Bad Request | `UNSUPPORTED_RESPONSE_TYPE` | response_type is not `code` |
| 401 Unauthorized | `UNAUTHORIZED` | User not authenticated |

---

### 6.21 POST /oauth/token (Authorization Code Exchange)

**Auth:** Client credentials via HTTP Basic Auth (confidential clients) or request body (public clients)
**Description:** Exchanges an authorization code for tokens. Validates PKCE S256.

**Request Body (application/x-www-form-urlencoded):**
```
grant_type=authorization_code
&code=ac_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ
&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
&client_id=client_01HZ_aBcDeF
&redirect_uri=https%3A%2F%2Fmyapp.example.com%2Fcallback
```

**Success Response — 200 OK:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "rt_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "openid profile email offline_access"
}
```

**Error Responses:**
| Status | Code | Description |
|---|---|---|
| 400 Bad Request | `invalid_grant` | Code not found, already used, expired |
| 400 Bad Request | `invalid_grant` | PKCE code_verifier does not match code_challenge |
| 400 Bad Request | `invalid_client` | Client authentication failed |
| 400 Bad Request | `invalid_request` | Missing required parameters |

Note: OAuth 2.0 error codes use snake_case per RFC 6749 (`invalid_grant`, not `INVALID_GRANT`).

---

### 6.22 POST /oauth/token (Client Credentials)

**Auth:** HTTP Basic Auth with client_id:client_secret
**Description:** Issues a service-to-service token (M2M). No user context.

**Request Body (application/x-www-form-urlencoded):**
```
grant_type=client_credentials
&scope=read%3Ausers%20write%3Adata
```

**Success Response — 200 OK:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read:users write:data"
}
```

**JWT Payload (M2M — no sub or roles):**
```json
{
  "iss": "https://auth.example.com",
  "aud": ["https://api.example.com"],
  "exp": 1740003600,
  "iat": 1740000000,
  "client_id": "client_01HZ_aBcDeF",
  "tenant_id": "acme-corp",
  "scope": "read:users write:data",
  "jti": "jwt_01HZ..."
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 400 Bad Request | `invalid_scope` | Requested scope not permitted for this client |
| 401 Unauthorized | `invalid_client` | client_id/client_secret invalid |

---

### 6.23 POST /auth/oauth/google

**Auth:** None
**Description:** Initiates Google OIDC login flow. Returns the Google authorization URL for the consuming app to redirect the user.

**Request Body:**
```json
{
  "redirect_uri": "https://myapp.example.com/auth/google/callback"
}
```

**Success Response — 200 OK:**
```json
{
  "authorization_url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=...&state=...&nonce=...",
  "state": "csrf_state_value_stored_in_redis"
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 400 Bad Request | `GOOGLE_NOT_CONFIGURED` | Tenant has not configured Google OAuth credentials |
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Invalid redirect_uri |

---

### 6.24 GET /auth/oauth/google/callback

**Auth:** None (Google redirects here with code + state)
**Description:** Handles Google OIDC callback. Validates state (CSRF), exchanges code for ID token, creates or links user account, issues internal JWT + refresh token.

**Query Parameters:** `code`, `state` (from Google redirect)

**Success Response — 200 OK:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "rt_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ",
  "token_type": "Bearer",
  "expires_in": 900,
  "is_new_user": false,
  "account_linked": false
}
```

**Account Linking Response — 200 OK (when Google email matches existing password account):**
```json
{
  "linking_required": true,
  "linking_token": "lt_aBcDeFgHiJkLmNoPqRsTuVwXyZ",
  "message": "A password account with this email already exists. Please verify your password to link your Google account.",
  "expires_in": 600
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 400 Bad Request | `INVALID_STATE` | State mismatch (CSRF) |
| 400 Bad Request | `GOOGLE_TOKEN_INVALID` | ID token verification failed |
| 400 Bad Request | `GOOGLE_AUTH_ERROR` | Google returned an error |

---

### 6.25 GET /.well-known/jwks.json

**Auth:** None
**Description:** Returns public keys for JWT verification. Resource servers cache this response. No prefix — this is at the root path.

**Success Response — 200 OK:**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "key-20260227",
      "n": "sZ6_yLmv...",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "key-20260101",
      "n": "oldKeyN...",
      "e": "AQAB"
    }
  ]
}
```

Note: Multiple keys are returned during key rotation. Clients must select the key matching the JWT header `kid` field.

**Cache-Control Header:** `Cache-Control: public, max-age=3600`

---

### 6.26 POST /oauth/introspect

**Auth:** HTTP Basic Auth (resource server must be a registered client)
**Description:** RFC 7662 token introspection. Allows resource servers to validate opaque tokens or get active status of JWTs.

**Request Body (application/x-www-form-urlencoded):**
```
token=rt_aBcDeFgHiJkLmNoPqRsTuVwXyZ_01HZ
&token_type_hint=refresh_token
```

**Success Response — 200 OK (active token):**
```json
{
  "active": true,
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "client_01HZ_aBcDeF",
  "scope": "openid profile",
  "exp": 1740000900,
  "iat": 1740000000,
  "tenant_id": "acme-corp",
  "roles": ["user"]
}
```

**Success Response — 200 OK (inactive/invalid token):**
```json
{
  "active": false
}
```

Note: Per RFC 7662, inactive tokens always return `{"active": false}` with 200 OK — never 401 or 404.

---

### 6.27 GET /users/me

**Auth:** Bearer JWT
**Description:** Returns the authenticated user's profile.

**Success Response — 200 OK:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "maya@example.com",
  "first_name": "Maya",
  "last_name": "Johnson",
  "status": "active",
  "mfa_enabled": false,
  "roles": ["user"],
  "tenant_id": "acme-corp",
  "email_verified_at": "2026-03-15T10:05:00Z",
  "last_login_at": "2026-03-20T09:00:00Z",
  "created_at": "2026-03-15T10:00:00Z"
}
```

---

### 6.28 PUT /users/me

**Auth:** Bearer JWT
**Description:** Updates the authenticated user's profile (name fields only in Must Have scope).

**Request Body:**
```json
{
  "first_name": "Maya",
  "last_name": "Johnson-Smith"
}
```

**Success Response — 200 OK:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "first_name": "Maya",
  "last_name": "Johnson-Smith",
  "updated_at": "2026-03-20T09:30:00Z"
}
```

**Error Responses:**
| Status | Code | Condition |
|---|---|---|
| 422 Unprocessable Entity | `VALIDATION_ERROR` | Field validation failure |
| 401 Unauthorized | `UNAUTHORIZED` | JWT invalid or expired |

---

## 7. Go Project Structure

The structure follows Clean Architecture with the Handler→Service→Repository pattern. All packages under `internal/` are private to the module.

```
auth-system/
├── cmd/
│   └── api/
│       └── main.go                    # Entry point: wire dependencies, start HTTP server
│
├── internal/
│   ├── config/
│   │   ├── config.go                  # Config struct; loaded from Vault + env at startup
│   │   └── config_test.go
│   │
│   ├── domain/                        # Pure domain layer — no framework dependencies
│   │   ├── entity/
│   │   │   ├── user.go                # User entity, status constants, domain methods
│   │   │   ├── session.go             # Session entity, family ID logic
│   │   │   ├── role.go                # Role entity
│   │   │   ├── oauth_client.go        # OAuthClient entity, redirect URI validation
│   │   │   ├── oauth_code.go          # AuthorizationCode entity, PKCE validation methods
│   │   │   ├── audit_event.go         # AuditEvent entity, event type constants
│   │   │   └── tenant.go              # Tenant entity, config struct
│   │   ├── valueobject/
│   │   │   ├── password.go            # PasswordHash value object; Argon2id params
│   │   │   ├── token.go               # AccessToken, RefreshToken, AuthCode value objects
│   │   │   └── email.go               # Email value object; normalization + validation
│   │   └── repository/                # Repository interfaces (implemented in infrastructure)
│   │       ├── user_repository.go
│   │       ├── session_repository.go
│   │       ├── role_repository.go
│   │       ├── oauth_repository.go
│   │       ├── audit_repository.go
│   │       └── tenant_repository.go
│   │
│   ├── handler/                       # Transport layer — HTTP handlers
│   │   ├── auth_handler.go            # POST /auth/register, /login, /logout, /logout/all
│   │   ├── auth_handler_test.go
│   │   ├── email_handler.go           # POST /auth/verify-email, /auth/resend-verification
│   │   ├── email_handler_test.go
│   │   ├── password_handler.go        # POST /auth/forgot-password, /auth/reset-password
│   │   ├── password_handler_test.go
│   │   ├── session_handler.go         # POST /auth/token/refresh
│   │   ├── session_handler_test.go
│   │   ├── admin_handler.go           # POST /admin/users/invite, PUT /admin/users/:id/disable, DELETE /admin/users/:id
│   │   ├── admin_handler_test.go
│   │   ├── tenant_handler.go          # POST /admin/tenants, GET /admin/tenants/:id
│   │   ├── tenant_handler_test.go
│   │   ├── role_handler.go            # POST /admin/roles, POST/DELETE /admin/users/:id/roles/:roleId
│   │   ├── role_handler_test.go
│   │   ├── oauth_handler.go           # GET /oauth/authorize, POST /oauth/token, POST /oauth/introspect, POST /oauth/clients
│   │   ├── oauth_handler_test.go
│   │   ├── google_handler.go          # POST /auth/oauth/google, GET /auth/oauth/google/callback
│   │   ├── google_handler_test.go
│   │   ├── user_handler.go            # GET /users/me, PUT /users/me
│   │   ├── user_handler_test.go
│   │   ├── audit_handler.go           # GET /admin/audit-log
│   │   ├── audit_handler_test.go
│   │   ├── wellknown_handler.go       # GET /.well-known/jwks.json
│   │   └── wellknown_handler_test.go
│   │
│   ├── service/                       # Application layer — use cases
│   │   ├── auth_service.go            # Register, Login, Logout orchestration
│   │   ├── auth_service_test.go
│   │   ├── email_verification_service.go  # Verify, Resend token logic
│   │   ├── email_verification_service_test.go
│   │   ├── password_service.go        # ForgotPassword, ResetPassword
│   │   ├── password_service_test.go
│   │   ├── session_service.go         # Refresh rotation, family revocation
│   │   ├── session_service_test.go
│   │   ├── tenant_service.go          # Provision tenant, schema creation, migrate
│   │   ├── tenant_service_test.go
│   │   ├── rbac_service.go            # Assign/unassign roles, list roles
│   │   ├── rbac_service_test.go
│   │   ├── oauth_service.go           # fosite integration: authorize, token, introspect
│   │   ├── oauth_service_test.go
│   │   ├── google_oauth_service.go    # OIDC flow, account linking
│   │   ├── google_oauth_service_test.go
│   │   ├── admin_service.go           # Invite, disable, delete users
│   │   ├── admin_service_test.go
│   │   ├── audit_service.go           # Append events, query log, cold-archive export
│   │   ├── audit_service_test.go
│   │   └── jwt_service.go             # Issue JWT, verify JWT, JWKS rotation
│   │
│   ├── repository/                    # Infrastructure layer — concrete repository implementations
│   │   ├── postgres/
│   │   │   ├── user_repo.go           # Implements domain/repository.UserRepository
│   │   │   ├── user_repo_test.go
│   │   │   ├── session_repo.go        # Implements SessionRepository
│   │   │   ├── session_repo_test.go
│   │   │   ├── role_repo.go           # Implements RoleRepository
│   │   │   ├── role_repo_test.go
│   │   │   ├── oauth_repo.go          # Implements OAuthRepository (fosite storage adapter)
│   │   │   ├── oauth_repo_test.go
│   │   │   ├── audit_repo.go          # Implements AuditRepository
│   │   │   ├── audit_repo_test.go
│   │   │   └── tenant_repo.go         # Implements TenantRepository (global schema)
│   │   └── redis/
│   │       ├── rate_limiter.go        # Sliding window rate limit via Lua EVAL
│   │       ├── rate_limiter_test.go
│   │       └── token_denylist.go      # SETEX hashed refresh tokens; check existence
│   │
│   ├── middleware/                    # Gin middleware
│   │   ├── auth.go                    # JWT validation; populates ctx with user claims
│   │   ├── auth_test.go
│   │   ├── tenant.go                  # Extracts tenant_id; sets PostgreSQL search_path
│   │   ├── tenant_test.go
│   │   ├── rate_limit.go              # Per-IP and per-user rate limiting via Redis
│   │   ├── rate_limit_test.go
│   │   ├── secure_headers.go          # HSTS, CSP, X-Frame-Options, nosniff, Referrer-Policy
│   │   ├── cors.go                    # CORS with per-tenant allowed origins
│   │   ├── logger.go                  # Structured request/response logging (slog)
│   │   ├── request_id.go              # X-Request-ID header injection and propagation
│   │   └── super_admin.go             # Validates super_admin role claim
│   │
│   ├── infrastructure/                # External system clients
│   │   ├── postgres/
│   │   │   ├── pool.go                # pgxpool initialization with Vault-sourced credentials
│   │   │   └── search_path.go         # SetSearchPath(ctx, schema) helper for tenant routing
│   │   ├── redis/
│   │   │   └── client.go              # go-redis/v9 client initialization
│   │   ├── vault/
│   │   │   ├── client.go              # Vault SDK client; AppRole auth; secret fetching
│   │   │   └── key_rotation.go        # Background goroutine watching for signing key rotation
│   │   ├── email/
│   │   │   ├── client.go              # Resend API client wrapper
│   │   │   ├── templates.go           # Go text/template for email bodies
│   │   │   └── worker.go              # Buffered channel consumer; retry with backoff
│   │   ├── google/
│   │   │   ├── oidc_client.go         # OIDC discovery, token verification (coreos/go-oidc)
│   │   │   └── state_store.go         # Redis-backed state/nonce storage for CSRF prevention
│   │   └── migrations/
│   │       ├── runner.go              # golang-migrate runner; per-schema execution
│   │       └── tenant_provisioner.go  # Schema creation + initial migration + role seeding
│   │
│   └── router/
│       └── router.go                  # Gin router setup; all routes registered here
│
├── migrations/
│   ├── global/                        # Migrations for public schema
│   │   ├── 000001_create_tenants.up.sql
│   │   ├── 000001_create_tenants.down.sql
│   │   ├── 000002_create_global_audit_log.up.sql
│   │   └── 000002_create_global_audit_log.down.sql
│   └── tenant/                        # Template migrations applied to every tenant schema
│       ├── 000001_create_users.up.sql
│       ├── 000001_create_users.down.sql
│       ├── 000002_create_roles.up.sql
│       ├── 000002_create_roles.down.sql
│       ├── 000003_create_user_roles.up.sql
│       ├── 000003_create_user_roles.down.sql
│       ├── 000004_create_sessions.up.sql
│       ├── 000004_create_sessions.down.sql
│       ├── 000005_create_password_reset_tokens.up.sql
│       ├── 000005_create_password_reset_tokens.down.sql
│       ├── 000006_create_email_verification_tokens.up.sql
│       ├── 000006_create_email_verification_tokens.down.sql
│       ├── 000007_create_oauth_clients.up.sql
│       ├── 000007_create_oauth_clients.down.sql
│       ├── 000008_create_oauth_authorization_codes.up.sql
│       ├── 000008_create_oauth_authorization_codes.down.sql
│       ├── 000009_create_oauth_social_accounts.up.sql
│       ├── 000009_create_oauth_social_accounts.down.sql
│       ├── 000010_create_audit_log.up.sql
│       ├── 000010_create_audit_log.down.sql
│       ├── 000011_create_tenant_config.up.sql
│       ├── 000011_create_tenant_config.down.sql
│       └── 000012_seed_default_roles.up.sql
│
├── pkg/                               # Shared utilities — no internal dependencies
│   ├── crypto/
│   │   ├── argon2.go                  # Argon2id hash + verify; configurable params
│   │   ├── argon2_test.go
│   │   ├── token.go                   # Secure random token generation (crypto/rand)
│   │   └── token_test.go
│   ├── jwtutil/
│   │   ├── jwt.go                     # RS256 sign + verify; kid selection; JWKS generation
│   │   └── jwt_test.go
│   ├── validator/
│   │   ├── validator.go               # go-playground/validator v10 wrapper; custom rules
│   │   └── password_policy.go         # Tenant-configurable password complexity checker
│   ├── paginator/
│   │   └── paginator.go               # Cursor/offset pagination helpers
│   ├── timeutil/
│   │   └── timeutil.go                # Time helpers; constant-time sleep for anti-timing
│   └── apierror/
│       └── apierror.go                # Standard API error types and response builders
│
├── test/
│   ├── integration/
│   │   ├── auth_flow_test.go          # Full register → verify → login → refresh → logout
│   │   ├── password_reset_test.go
│   │   ├── oauth_pkce_test.go
│   │   └── google_oauth_test.go
│   └── isolation/
│       └── cross_tenant_test.go       # US-08b: cross-tenant isolation test suite (CI-blocking)
│
├── scripts/
│   ├── migrate.sh                     # CLI: run migrations for all tenants or specific tenant
│   └── provision_tenant.sh            # CLI: provision a tenant (calls provisioning API)
│
├── Dockerfile                         # Multi-stage build; scratch final image
├── docker-compose.yml                 # Local dev: postgres, redis, vault (dev mode), mailhog
├── docker-compose.test.yml            # Integration test environment
├── fly.toml                           # Fly.io deployment config
├── go.mod
├── go.sum
└── Makefile                           # build, test, lint, migrate, docker targets
```

### Key Dependency Choices

| Package | Version | Purpose |
|---|---|---|
| `github.com/gin-gonic/gin` | v1.10+ | HTTP router and middleware framework |
| `github.com/jackc/pgx/v5` | v5+ | PostgreSQL driver with pgxpool |
| `github.com/redis/go-redis/v9` | v9+ | Redis client |
| `github.com/golang-jwt/jwt/v5` | v5+ | JWT signing and verification |
| `github.com/ory/fosite` | v0.47+ | OAuth 2.0 / OIDC authorization server |
| `github.com/coreos/go-oidc/v3` | v3+ | OIDC client for Google social login |
| `github.com/golang-migrate/migrate/v4` | v4+ | Database migration runner |
| `github.com/hashicorp/vault-client-go` | latest | Vault SDK |
| `golang.org/x/crypto` | latest | Argon2id, bcrypt |
| `github.com/go-playground/validator/v10` | v10+ | Request validation |
| `log/slog` | stdlib (Go 1.21+) | Structured logging |
| `github.com/google/uuid` | v1+ | UUID generation |
| `github.com/stretchr/testify` | v1+ | Test assertions |

---

## 8. Schema-per-Tenant Implementation Design

### 8.A Tenant Schema Routing Middleware

#### Overview

Every authenticated database operation must execute within the correct PostgreSQL schema. The approach uses a Go Gin middleware that:

1. Extracts and validates the JWT from the `Authorization: Bearer` header
2. Reads `tenant_id` from the JWT claims
3. Resolves `tenant_id` to the tenant's `schema_name` (e.g., `tenant_acme_corp`)
4. Acquires a connection from `pgxpool` and sets `search_path` for that connection's request lifetime
5. Stores the schema-scoped connection (or schema name) in the Gin context for downstream handlers

#### Unauthenticated Endpoints

For endpoints that do not require a JWT (register, login, forgot-password, verify-email), the tenant is identified by the `X-Tenant-ID` header. The middleware performs a lookup in the **global `public` schema** `tenants` table to resolve the tenant slug to a schema name. No JWT parsing occurs on these paths.

#### Code Pattern

```go
// internal/middleware/tenant.go

package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
)

const (
    CtxKeySchemaName = "tenant_schema_name"
    CtxKeyTenantID   = "tenant_id"
    CtxKeyDBConn     = "db_conn"
)

// TenantMiddleware resolves the tenant schema from JWT claims (authenticated)
// or X-Tenant-ID header (unauthenticated), then sets PostgreSQL search_path.
func TenantMiddleware(pool *pgxpool.Pool, tenantCache TenantCache) gin.HandlerFunc {
    return func(c *gin.Context) {
        var schemaName string
        var tenantID string

        // Attempt to read tenant_id from already-validated JWT claims
        // (AuthMiddleware must run before TenantMiddleware for authenticated routes)
        if claims, exists := c.Get("jwt_claims"); exists {
            jwtClaims := claims.(JWTClaims)
            tenantID = jwtClaims.TenantID

            // Validate JWT tenant_id matches X-Tenant-ID header if present
            headerTenantID := c.GetHeader("X-Tenant-ID")
            if headerTenantID != "" && headerTenantID != tenantID {
                c.AbortWithStatusJSON(http.StatusForbidden, APIError{
                    Code:    "TENANT_MISMATCH",
                    Message: "JWT tenant does not match X-Tenant-ID header.",
                })
                return
            }
        } else {
            // Unauthenticated endpoint — use X-Tenant-ID header
            tenantID = c.GetHeader("X-Tenant-ID")
            if tenantID == "" {
                c.AbortWithStatusJSON(http.StatusBadRequest, APIError{
                    Code:    "MISSING_TENANT_ID",
                    Message: "X-Tenant-ID header is required.",
                })
                return
            }
        }

        // Resolve tenant_id → schema_name (with in-memory cache + TTL)
        schema, err := tenantCache.GetSchema(c.Request.Context(), tenantID)
        if err != nil || schema == "" {
            c.AbortWithStatusJSON(http.StatusForbidden, APIError{
                Code:    "INVALID_TENANT",
                Message: "Tenant not found or inactive.",
            })
            return
        }
        schemaName = schema

        // Validate schema name format before use in SQL to prevent injection
        // Schema names must match: ^tenant_[a-z0-9_]{1,50}$
        if !isValidSchemaName(schemaName) {
            c.AbortWithStatusJSON(http.StatusInternalServerError, APIError{
                Code:    "INTERNAL_ERROR",
                Message: "Invalid tenant configuration.",
            })
            return
        }

        // Store schema name in context — repositories use this to set search_path
        c.Set(CtxKeySchemaName, schemaName)
        c.Set(CtxKeyTenantID, tenantID)

        c.Next()
    }
}

// isValidSchemaName validates schema name against allowlist pattern.
// This prevents SQL injection in SET search_path statements.
func isValidSchemaName(name string) bool {
    matched, _ := regexp.MatchString(`^tenant_[a-z0-9_]{1,50}$`, name)
    return matched
}
```

```go
// internal/infrastructure/postgres/search_path.go

package postgres

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantSchema executes fn with a PostgreSQL connection whose search_path
// is set to the tenant schema. The connection is returned to the pool after fn completes.
// The schema name is pre-validated by TenantMiddleware before this function is called.
func WithTenantSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string, fn func(conn *pgx.Conn) error) error {
    conn, err := pool.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquire connection: %w", err)
    }
    defer conn.Release()

    // Set search_path for this connection's lifetime.
    // schemaName is pre-validated with isValidSchemaName — safe from injection.
    // Always include public schema for cross-schema functions (e.g., gen_random_uuid).
    _, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
    if err != nil {
        return fmt.Errorf("set search_path: %w", err)
    }

    return fn(conn.Conn())
}
```

```go
// Usage pattern in a repository:
// internal/repository/postgres/user_repo.go

func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
    schemaName, ok := ctx.Value(CtxKeySchemaName).(string)
    if !ok || schemaName == "" {
        return nil, errors.New("schema name not found in context — tenant middleware not applied")
    }

    return postgres.WithTenantSchema(ctx, r.pool, schemaName, func(conn *pgx.Conn) error {
        // Query executes within the tenant schema due to search_path
        row := conn.QueryRow(ctx,
            "SELECT id, email, password_hash, status FROM users WHERE email = $1 AND deleted_at IS NULL",
            strings.ToLower(email),
        )
        // ... scan row into entity
        return nil
    })
}
```

#### TenantCache Design

The tenant ID → schema name mapping is cached in-memory with a 60-second TTL to avoid a global schema lookup on every request. Cache invalidation occurs on tenant status changes via a Redis pub/sub channel.

```go
type TenantCache interface {
    GetSchema(ctx context.Context, tenantID string) (string, error)
}

// Implementation: in-memory map with sync.RWMutex + TTL expiry
// Refresh: background goroutine re-fetches from public.tenants every 60s
// Invalidation: Redis PUBLISH on tenant status change → cache entry cleared
```

---

### 8.B Migration Runner Design

#### Recommended Library: `golang-migrate/migrate/v4`

**Why:** Supports PostgreSQL natively, tracks applied migrations in a `schema_migrations` table per schema, supports transactional migrations (DDL in transactions), and has a well-tested Go API for programmatic use.

#### Migration Tracking

Each tenant schema has its own `schema_migrations` table (created by `golang-migrate` automatically). This provides per-schema migration state tracking with no cross-schema interference.

#### New Tenant Schema Provisioning

When a super-admin calls `POST /admin/tenants`, the `TenantService` calls `TenantProvisioner.Provision()`:

```go
// internal/infrastructure/migrations/tenant_provisioner.go

package migrations

import (
    "context"
    "fmt"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/jackc/pgx/v5/pgxpool"
)

type TenantProvisioner struct {
    pool           *pgxpool.Pool
    migrationsPath string // e.g., "file://migrations/tenant"
    dbURL          string // base postgres:// URL without search_path
}

// Provision creates the tenant schema and applies all tenant migrations.
// This operation is transactional: if migration fails, schema is dropped.
func (p *TenantProvisioner) Provision(ctx context.Context, schemaName string) error {
    // Step 1: Create schema atomically
    conn, err := p.pool.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquire conn for schema creation: %w", err)
    }
    defer conn.Release()

    // Schema name pre-validated by caller
    _, err = conn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
    if err != nil {
        return fmt.Errorf("create schema %s: %w", schemaName, err)
    }
    conn.Release()

    // Step 2: Run all tenant migrations against the new schema
    // golang-migrate uses search_path via the options parameter
    dbURLWithSchema := fmt.Sprintf("%s&search_path=%s,public&x-migrations-table=schema_migrations",
        p.dbURL, schemaName)

    m, err := migrate.New(p.migrationsPath, dbURLWithSchema)
    if err != nil {
        // Rollback: drop schema if migration setup fails
        p.dropSchema(ctx, schemaName)
        return fmt.Errorf("create migrator for schema %s: %w", schemaName, err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        p.dropSchema(ctx, schemaName)
        return fmt.Errorf("apply migrations to schema %s: %w", schemaName, err)
    }

    // Step 3: Seed default roles (admin, user) — part of migration 000012
    // This is handled by migration 000012_seed_default_roles.up.sql
    // No additional code needed here

    return nil
}

func (p *TenantProvisioner) dropSchema(ctx context.Context, schemaName string) {
    conn, _ := p.pool.Acquire(ctx)
    if conn != nil {
        defer conn.Release()
        conn.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
    }
}
```

#### Existing Tenant Schema Migration on Deployment

On every application startup (and available as a CLI command), the migration runner iterates all active tenant schemas and applies pending migrations:

```go
// internal/infrastructure/migrations/runner.go

package migrations

import (
    "context"
    "fmt"
    "log/slog"
    "github.com/golang-migrate/migrate/v4"
)

// RunAllTenantMigrations fetches all active tenant schema names from the global
// registry and applies pending migrations to each. Failed tenants are logged
// but do not block other tenants from being migrated.
func (r *MigrationRunner) RunAllTenantMigrations(ctx context.Context) error {
    schemas, err := r.tenantRepo.ListActiveSchemaNames(ctx)
    if err != nil {
        return fmt.Errorf("list tenant schemas: %w", err)
    }

    var failedSchemas []string
    for _, schemaName := range schemas {
        if err := r.migrateTenantSchema(ctx, schemaName); err != nil {
            slog.Error("migration failed for tenant schema",
                "schema", schemaName,
                "error", err,
            )
            failedSchemas = append(failedSchemas, schemaName)
            // Continue — do not block other tenants
        } else {
            slog.Info("migrations applied", "schema", schemaName)
        }
    }

    if len(failedSchemas) > 0 {
        // Alert via monitoring — failed schemas need manual intervention
        return fmt.Errorf("migrations failed for schemas: %v", failedSchemas)
    }
    return nil
}

func (r *MigrationRunner) migrateTenantSchema(ctx context.Context, schemaName string) error {
    dbURLWithSchema := fmt.Sprintf("%s&search_path=%s,public&x-migrations-table=schema_migrations",
        r.dbURL, schemaName)

    m, err := migrate.New(r.migrationsPath, dbURLWithSchema)
    if err != nil {
        return err
    }
    defer m.Close()

    err = m.Up()
    if err == migrate.ErrNoChange {
        return nil // Already up to date
    }
    return err
}
```

#### Developer CLI Commands

```makefile
# Makefile targets

migrate-global:
    go run ./cmd/migrate -scope=global -direction=up

migrate-tenant:
    go run ./cmd/migrate -scope=tenant -tenant=$(TENANT) -direction=up

migrate-all-tenants:
    go run ./cmd/migrate -scope=all-tenants -direction=up

migrate-create:
    go run ./cmd/migrate -create -name=$(NAME) -scope=$(SCOPE)
```

#### Migration Idempotency Requirements

- All `UP` migrations must be idempotent where possible: use `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
- Migrations run inside a PostgreSQL transaction; if any statement fails, the entire migration is rolled back and `schema_migrations.dirty = true` is set
- A dirty migration blocks further migrations for that schema and requires manual intervention

---

## 9. Security Architecture

### 9.1 Token Lifecycle — Registration to Logout

```mermaid
sequenceDiagram
    participant C as Client (Consuming App)
    participant A as Auth API
    participant DB as PostgreSQL (Tenant Schema)
    participant R as Redis
    participant E as Email Provider

    Note over C,E: Registration Flow
    C->>A: POST /auth/register {email, password}
    A->>A: Validate input; check password complexity
    A->>DB: INSERT user (status=unverified, password_hash=Argon2id)
    A->>DB: INSERT email_verification_token (token_hash, expires_at=+24h)
    A->>E: Send verification email (async)
    A-->>C: 201 Created {user_id, status=unverified}

    Note over C,E: Email Verification
    C->>A: POST /auth/verify-email {token}
    A->>DB: SELECT token; validate not used, not expired
    A->>DB: UPDATE user SET status=active, email_verified_at=now()
    A->>DB: UPDATE token SET used=true
    A-->>C: 200 OK

    Note over C,E: Login Flow
    C->>A: POST /auth/login {email, password}
    A->>R: Check IP rate limit (sliding window)
    A->>DB: SELECT user WHERE email=? AND deleted_at IS NULL
    A->>A: Argon2id.Verify(password, hash) [constant-time]
    A->>A: Check locked_until; check status=active
    A->>A: Generate JWT (RS256, 15min, kid=current_key_id)
    A->>A: Generate opaque refresh token (32 bytes, crypto/rand)
    A->>DB: INSERT session (refresh_token_hash=SHA256(token), family_id=new_uuid, expires_at=+24h)
    A->>DB: INSERT audit_log (LOGIN_SUCCESS)
    A-->>C: 200 OK {access_token, refresh_token, expires_in=900}

    Note over C,E: Token Refresh (Rotation)
    C->>A: POST /auth/token/refresh {refresh_token}
    A->>DB: SELECT session WHERE token_hash=SHA256(refresh_token) AND is_revoked=false
    A->>A: Check expires_at not exceeded
    A->>DB: UPDATE session SET is_revoked=true, revoked_at=now() [old token]
    A->>A: Generate new JWT + new opaque refresh token
    A->>DB: INSERT session (new_token_hash, same family_id, new expires_at)
    A->>DB: INSERT audit_log (TOKEN_REFRESHED)
    A-->>C: 200 OK {new access_token, new refresh_token}

    Note over C,E: Suspicious Reuse Detection
    C->>A: POST /auth/token/refresh {old_refresh_token already rotated}
    A->>DB: SELECT session WHERE token_hash=SHA256(old_token)
    A->>A: Session found BUT is_revoked=true → SUSPICIOUS REUSE
    A->>DB: UPDATE sessions SET is_revoked=true WHERE family_id=compromised_family_id [ALL tokens in family]
    A->>DB: INSERT audit_log (SUSPICIOUS_TOKEN_REUSE)
    A-->>C: 401 Unauthorized {code: SUSPICIOUS_TOKEN_REUSE}

    Note over C,E: Logout
    C->>A: POST /auth/logout {refresh_token} + Authorization: Bearer {access_token}
    A->>DB: UPDATE session SET is_revoked=true WHERE token_hash=SHA256(refresh_token)
    A->>DB: INSERT audit_log (LOGOUT)
    A-->>C: 200 OK
    Note over C,A: Access token remains valid until 15-min TTL (documented trade-off)
```

### 9.2 Refresh Token Family Revocation Design

The family revocation mechanism defends against refresh token theft:

**Normal rotation:**
- Token `rt_A` (family `F1`) is presented → `rt_A` revoked, `rt_B` (family `F1`) issued
- Token `rt_B` presented → `rt_B` revoked, `rt_C` (family `F1`) issued

**Theft detection:**
- Attacker steals `rt_A` after it has been rotated
- Attacker presents `rt_A` → auth system finds session with `rt_A.hash`, checks `is_revoked = true`
- This proves `rt_A` was already consumed — either the legitimate user or an attacker has the new token
- **Action:** Revoke ALL sessions in family `F1` (set `is_revoked = true WHERE family_id = F1`)
- **Write:** `SUSPICIOUS_TOKEN_REUSE` audit log event with IP, user agent
- **Response to attacker (and legitimate user):** 401 `SUSPICIOUS_TOKEN_REUSE` — both are now logged out; legitimate user must re-authenticate

**Implementation:**
```sql
-- Revoke entire family on reuse detection
UPDATE sessions
SET is_revoked = true, revoked_at = now()
WHERE family_id = $1
  AND is_revoked = false;
```

**Database constraint:** `family_id` is indexed on `sessions` table. The update affects at most 1–3 rows in typical usage.

### 9.3 Secrets Rotation Strategy (Zero-Downtime Key Rotation)

The JWT signing key rotation uses a **grace period overlap pattern**:

**Rotation Steps:**

1. **New key generated in Vault** (Transit secrets engine or manual rotation): new RS256 key pair with `kid = key-YYYYMMDD`
2. **Vault notifies application** via a background goroutine polling Vault's key list every 5 minutes (or via Vault agent with template re-render)
3. **Application loads new key** into an in-memory key map: `map[kid]rsa.PrivateKey`
4. **JWKS endpoint** begins serving both old and new public keys
5. **New tokens are signed** with the new key (new `kid` in JWT header)
6. **Old tokens remain valid** — JWKS still serves old public key for the duration of the old key's max TTL (15 min access token + 24h session = 24h15m maximum overlap period)
7. **Old key retired** from JWKS after 25 hours (24h + 1h buffer)

```go
// internal/infrastructure/vault/key_rotation.go

// KeyRotationWatcher polls Vault for signing key updates every 5 minutes.
// On new key detection, it updates the in-memory key store used by JWTService.
func (w *KeyRotationWatcher) Watch(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            keys, err := w.vaultClient.ListSigningKeys(ctx)
            if err != nil {
                slog.Error("failed to fetch signing keys from Vault", "error", err)
                continue
            }
            w.keyStore.Update(keys) // Atomic swap of in-memory key map
        case <-ctx.Done():
            return
        }
    }
}
```

**Token validation** always checks the `kid` header field and looks up the corresponding public key from the in-memory map. If `kid` is not found, the token is rejected.

### 9.4 OAuth 2.0 PKCE Flow

```mermaid
sequenceDiagram
    participant UA as User Agent (Browser)
    participant CA as Client App
    participant A as Auth API
    participant DB as PostgreSQL

    Note over UA,DB: Client App Setup (one-time, not per-flow)
    CA->>A: POST /oauth/clients {redirect_uris, scopes} + Admin JWT
    A-->>CA: {client_id, client_secret (shown once)}

    Note over UA,DB: Authorization Code Flow with PKCE S256

    CA->>CA: Generate code_verifier = random 43-128 char string (crypto/rand)
    CA->>CA: code_challenge = BASE64URL(SHA256(code_verifier))
    CA->>CA: Generate state = random CSRF token

    CA->>UA: Redirect user to Auth API (app UI navigates to authorize endpoint)
    UA->>A: GET /oauth/authorize?response_type=code&client_id=...&redirect_uri=...&scope=...&state=...&code_challenge=...&code_challenge_method=S256
    Note over A: User must be authenticated (JWT in Authorization header)
    A->>A: Validate: client_id exists, redirect_uri matches exactly, state present, code_challenge_method=S256
    A->>A: Generate authorization_code (random, single-use, 10min TTL)
    A->>DB: INSERT oauth_authorization_codes (code_hash, code_challenge, redirect_uri, scopes, user_id, client_id)
    A-->>UA: 200 OK {code, state, redirect_uri}
    UA->>CA: Delivers code + state to client app

    CA->>CA: Verify returned state matches stored state (CSRF check)
    CA->>A: POST /oauth/token {grant_type=authorization_code, code, code_verifier, client_id, redirect_uri}
    A->>DB: SELECT auth_code WHERE code_hash=SHA256(code) AND used=false AND expires_at>now()
    A->>A: Verify: SHA256(code_verifier) == stored code_challenge (PKCE S256)
    A->>A: Verify: redirect_uri matches stored redirect_uri
    A->>DB: UPDATE auth_code SET used=true
    A->>A: Generate access_token (JWT, 15min) + refresh_token (opaque)
    A->>DB: INSERT session (refresh_token_hash, family_id, user_id)
    A-->>CA: 200 OK {access_token, refresh_token, token_type, expires_in, scope}

    Note over CA,DB: Code Reuse Attack Prevention
    CA->>A: POST /oauth/token with same code (second attempt)
    A->>DB: SELECT auth_code → found but used=true
    A->>DB: REVOKE all tokens issued from this code (family revocation)
    A-->>CA: 400 Bad Request {error: invalid_grant}
```

### 9.5 Rate Limiting Architecture (Redis Sliding Window)

**Pattern:** Redis sliding window using a Lua script for atomicity. Two independent limiters run per request:

1. **IP-based limiter:** `rate_limit:ip:{ip_address}:{endpoint}` — prevents distributed brute force
2. **User-based limiter:** `rate_limit:user:{tenant_id}:{user_id}` — prevents targeted account attacks

**Lua Script (atomic sliding window):**

```lua
-- Sliding window rate limiter
-- KEYS[1] = rate limit key (e.g., "rate_limit:ip:203.0.113.45:login")
-- ARGV[1] = current timestamp (milliseconds)
-- ARGV[2] = window size (milliseconds, e.g., 60000 for 1 minute)
-- ARGV[3] = max requests in window
-- Returns: {current_count, is_limited (0/1)}

local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local min_score = now - window

-- Remove requests outside the window
redis.call('ZREMRANGEBYSCORE', key, '-inf', min_score)

-- Count requests in window
local count = redis.call('ZCARD', key)

if count >= limit then
    return {count, 1}
end

-- Add current request
redis.call('ZADD', key, now, now .. '-' .. math.random(1000000))
redis.call('PEXPIRE', key, window)

return {count + 1, 0}
```

**Default Rate Limits (configurable per tenant via `tenant_config`):**

| Endpoint | IP Limit | User Limit | Window |
|---|---|---|---|
| POST /auth/login | 20 req/min | 5 req/min | 60s |
| POST /auth/register | 10 req/min | — | 60s |
| POST /auth/forgot-password | 5 req/min | 3 req/hour | 3600s |
| POST /auth/token/refresh | 60 req/min | 30 req/min | 60s |
| POST /auth/verify-email | 10 req/min | — | 60s |
| POST /oauth/token | 30 req/min | — | 60s |

**Fail-closed behavior (AVAIL-02):** If Redis is unavailable, rate limiting fails closed — all rate-limited endpoints return `503 Service Unavailable` rather than bypassing the limit check.

**Response headers on limit hit:**
```
HTTP/1.1 429 Too Many Requests
Retry-After: 42
X-RateLimit-Limit: 20
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1740001000
```

### 9.6 Threat Model — Top 5 Threats and Mitigations

| # | Threat | STRIDE Category | Impact | Mitigation |
|---|---|---|---|---|
| T1 | **Credential Stuffing / Brute Force** Attacker uses leaked credential lists to attempt login across all tenant users. | Spoofing | High — account takeover | 1. Redis sliding window rate limiting per IP + per user. 2. Account lockout after N failures (configurable per tenant). 3. Argon2id with tuned parameters slows hash verification to ~300ms per attempt. 4. Ambiguous error messages (no email/password distinction). 5. `ACCOUNT_LOCKED` audit event + alerting on spike > 10x baseline. |
| T2 | **Refresh Token Theft + Replay** Attacker intercepts or extracts a refresh token from a compromised client device. | Spoofing, Elevation of Privilege | High — persistent session hijack | 1. Refresh token rotation on every use — stolen old token is detected as suspicious reuse. 2. Family revocation on reuse detection — invalidates entire session tree. 3. Opaque tokens stored hashed (SHA-256) in DB — DB breach does not expose tokens. 4. `SUSPICIOUS_TOKEN_REUSE` audit event triggers alert. 5. Absolute session expiry (24h) limits attack window. |
| T3 | **Cross-Tenant Data Access** Bug in query logic or middleware allows tenant A's JWT to access tenant B's data. | Information Disclosure | Critical — multi-tenant data breach | 1. Schema-per-tenant (ADR-001) — architectural isolation; no WHERE clause needed. 2. TenantMiddleware validates JWT `tenant_id` matches `X-Tenant-ID` header. 3. Schema name validated with regex allowlist before use in SET search_path. 4. US-08b automated cross-tenant isolation test suite runs on every CI PR. 5. No shared tables between tenants in the data model. |
| T4 | **OAuth Open Redirect / Authorization Code Injection** Attacker manipulates `redirect_uri` to steal authorization codes or craft phishing redirects. | Spoofing, Elevation of Privilege | High — account takeover via code theft | 1. `redirect_uri` strict exact-match against pre-registered URIs — no wildcard, no partial match. 2. On invalid `client_id` or `redirect_uri`: return error directly, never redirect (prevents open redirect). 3. `state` parameter mandatory on all OAuth flows (CSRF prevention). 4. PKCE S256 mandatory — authorization code is useless without `code_verifier`. 5. Authorization codes are single-use, 10-min TTL, stored hashed. |
| T5 | **Signing Key Compromise** Attacker obtains the RS256 private signing key, enabling arbitrary JWT forgery. | Elevation of Privilege | Critical — complete authentication bypass | 1. Private keys stored only in Vault (ADR-009) — never in environment variables, config files, or code. 2. Keys fetched at runtime via mTLS-authenticated Vault API — never written to disk. 3. Zero-downtime key rotation (Section 9.3) enables rapid rotation on suspicion of compromise. 4. Vault audit log captures every key access — supports forensic investigation. 5. `kid` in JWT header enables precise key targeting and facilitates fast rotation. |

---

## 10. Non-Functional Architecture

### 10.1 Scalability

**Horizontal Scaling Model:**

The Go Auth API is designed to be **stateless** at the application tier. All shared state is externalized:
- Session state → PostgreSQL (tenant schema `sessions` table)
- Rate limit counters → Redis
- Token denylist → Redis (hashed refresh tokens with TTL)
- Signing keys → Vault (fetched at startup, cached in-memory with rotation polling)

Multiple instances of the Go Auth API can run behind a load balancer with no sticky sessions required.

**Bottleneck Analysis:**

| Component | Bottleneck | Mitigation |
|---|---|---|
| PostgreSQL | Connection pool exhaustion; Argon2id hash verification is CPU-bound (intentional) | pgxpool with `MaxConns = 20` per instance; Argon2id runs in goroutines off the request goroutine; add read replicas for audit log queries |
| Redis | Single-threaded command processing; network latency | Redis Cluster or Sentinel for HA; Lua scripts pipeline multiple commands atomically |
| Vault | Secret fetch latency at startup | In-memory key cache; Vault is not in hot path for individual requests |
| Schema-per-tenant `search_path` | `SET search_path` adds ~0.1ms overhead per request | Acceptable; connection pool is per-schema if latency becomes an issue at scale |
| Email sending | Synchronous email would block requests | Async via buffered Go channel; email worker is decoupled |

**Schema Count at Scale:**

PostgreSQL supports tens of thousands of schemas per database. At 1,000 tenants, schema count is well within operational limits. Monitor `pg_stat_user_tables` per schema for table bloat. Above 5,000 tenants, evaluate migration to DB-per-tenant or a hybrid model.

### 10.2 Observability

**Structured Logging Fields (slog, JSON format):**

Every log entry includes these fields:

```json
{
  "time": "2026-03-15T10:00:00.123Z",
  "level": "INFO",
  "msg": "request completed",
  "service": "auth-api",
  "version": "1.2.3",
  "request_id": "req_01HZ...",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "latency_ms": 142,
  "tenant_id": "acme-corp",
  "user_id": "550e8400-...",
  "ip": "203.0.113.45",
  "user_agent": "MyApp/1.0"
}
```

Security events additionally include: `event_type`, `actor_id`, `target_user_id`, `outcome`.

**Prometheus Metrics to Expose (`/metrics`):**

| Metric | Type | Labels | Description |
|---|---|---|---|
| `auth_requests_total` | Counter | `method`, `path`, `status`, `tenant_id` | Total HTTP requests |
| `auth_request_duration_seconds` | Histogram | `method`, `path`, `tenant_id` | Request latency (buckets: 10ms, 50ms, 100ms, 300ms, 1s) |
| `auth_login_attempts_total` | Counter | `outcome`, `tenant_id` | Login attempts (success/failure) |
| `auth_token_refresh_total` | Counter | `outcome`, `tenant_id` | Token refresh attempts |
| `auth_suspicious_reuse_total` | Counter | `tenant_id` | Suspicious token reuse detections |
| `auth_active_sessions_gauge` | Gauge | `tenant_id` | Active non-revoked, non-expired sessions |
| `auth_rate_limit_hits_total` | Counter | `limiter_type`, `tenant_id` | Rate limit hits (ip/user) |
| `auth_db_pool_acquired_total` | Counter | — | DB pool acquire calls |
| `auth_db_pool_wait_duration_seconds` | Histogram | — | Time waiting for DB connection |
| `auth_email_send_total` | Counter | `type`, `outcome` | Email send attempts and outcomes |
| `auth_vault_fetch_total` | Counter | `outcome` | Vault secret fetch attempts |

**Alerting Rules (AVAIL-04):**

```yaml
# Alert: error rate > 1%
- alert: HighErrorRate
  expr: rate(auth_requests_total{status=~"5.."}[5m]) / rate(auth_requests_total[5m]) > 0.01

# Alert: p95 latency > 500ms
- alert: HighLatency
  expr: histogram_quantile(0.95, rate(auth_request_duration_seconds_bucket[5m])) > 0.5

# Alert: failed login spike > 10x baseline
- alert: LoginFailureSpike
  expr: rate(auth_login_attempts_total{outcome="failure"}[5m]) > 10 * rate(auth_login_attempts_total{outcome="failure"}[1h] offset 1d)

# Alert: suspicious token reuse (any occurrence)
- alert: SuspiciousTokenReuse
  expr: increase(auth_suspicious_reuse_total[5m]) > 0
```

**Distributed Tracing:**

Use OpenTelemetry (OTEL) SDK for Go with trace propagation via `traceparent` header (W3C Trace Context). Export to Jaeger (local dev) or OTLP-compatible backend (production). Instrument:
- Every HTTP request → span with tenant_id, user_id attributes
- Every DB query → child span with query type (not full SQL — avoid logging PII)
- Every Redis command → child span
- Every email send → child span

### 10.3 Deployment Architecture

**Local Development (Docker Compose):**

```yaml
# docker-compose.yml
services:
  api:
    build: .
    ports: ["8080:8080"]
    environment:
      VAULT_ADDR: http://vault:8200
      VAULT_TOKEN: dev-root-token
      POSTGRES_URL: postgres://auth:auth@postgres:5432/authdb
      REDIS_URL: redis://redis:6379
    depends_on: [postgres, redis, vault, mailhog]

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: authdb
      POSTGRES_USER: auth
      POSTGRES_PASSWORD: auth
    volumes: ["postgres_data:/var/lib/postgresql/data"]
    ports: ["5432:5432"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  vault:
    image: hashicorp/vault:latest
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: dev-root-token
      VAULT_DEV_LISTEN_ADDRESS: 0.0.0.0:8200
    cap_add: [IPC_LOCK]
    ports: ["8200:8200"]

  mailhog:
    image: mailhog/mailhog
    ports: ["1025:1025", "8025:8025"]  # SMTP + Web UI
```

**Production Architecture (Fly.io — Pilot):**

```
Internet
    │
    ▼
[Fly.io Anycast Edge — TLS termination, HSTS]
    │
    ▼
[Fly.io Load Balancer]
    │
    ├──────────────────────────────────┐
    ▼                                  ▼
[Go Auth API — Instance 1]    [Go Auth API — Instance 2]
  fly-region: sjc               fly-region: sjc
  vm.size: shared-cpu-2x        vm.size: shared-cpu-2x
    │                                  │
    ├──────────────────────────────────┤
    ▼                                  ▼
[Fly Postgres — Primary]      [Upstash Redis]
  PostgreSQL 16                  Redis 7 (managed)
  2 vCPU, 4GB RAM                TLS enforced
  Volume: 20GB SSD
    │
    ▼
[Fly Postgres — Replica]      [HCP Vault — Dedicated]
  Read replica for                Vault 1.15+
  audit log queries               mTLS auth
    │
    ▼
[S3-compatible (Tigris or AWS S3)]
  Audit log cold archive
  Lifecycle: Glacier after 1 year
```

**Dockerfile (Multi-stage):**

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /auth-api ./cmd/api

# Final stage — minimal scratch image
FROM scratch
COPY --from=builder /auth-api /auth-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENTRYPOINT ["/auth-api"]
```

### 10.4 CI/CD Pipeline Design (GitHub Actions)

```yaml
# .github/workflows/ci.yml
# Triggered on: push to any branch, PR to main

name: CI/CD Pipeline

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - uses: golangci/golangci-lint-action@v4
        with: { version: latest }

  unit-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./... -race -count=1 -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out | tail -1
        # Fail if coverage < 80%
      - uses: codecov/codecov-action@v4

  integration-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env: { POSTGRES_DB: authdb_test, POSTGRES_USER: auth, POSTGRES_PASSWORD: auth }
        options: --health-cmd pg_isready
      redis:
        image: redis:7-alpine
        options: --health-cmd "redis-cli ping"
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./test/integration/... -v -timeout 120s

  isolation-test:
    # US-08b: Cross-tenant isolation — blocks merge on any failure
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env: { POSTGRES_DB: authdb_test, POSTGRES_USER: auth, POSTGRES_PASSWORD: auth }
      redis:
        image: redis:7-alpine
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./test/isolation/... -v -timeout 180s
        # Any cross-tenant data leak = immediate CI failure

  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
      - name: Run gosec
        uses: securego/gosec@master
        with: { args: '-severity medium ./...' }

  build:
    needs: [lint, unit-test, integration-test, isolation-test, security-scan]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: docker build -t auth-api:${{ github.sha }} .
      - name: Push to registry
        if: github.ref == 'refs/heads/main'
        run: |
          echo ${{ secrets.REGISTRY_PASSWORD }} | docker login -u ${{ secrets.REGISTRY_USER }} --password-stdin
          docker push auth-api:${{ github.sha }}

  deploy-staging:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --image auth-api:${{ github.sha }} --app auth-staging
        env: { FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }} }

  deploy-production:
    needs: deploy-staging
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    environment: production  # Requires manual approval gate
    steps:
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --image auth-api:${{ github.sha }} --app auth-production
        env: { FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN_PROD } }
```

**Pipeline Stage Summary:**

| Stage | Trigger | Blocks Deploy | Duration |
|---|---|---|---|
| Lint (golangci-lint) | Every push/PR | Yes | ~2 min |
| Unit Tests (≥ 80% coverage) | Every push/PR | Yes | ~3 min |
| Integration Tests | Every push/PR | Yes | ~5 min |
| Cross-Tenant Isolation Tests | Every push/PR | Yes — hard block | ~3 min |
| Security Scan (govulncheck, gosec) | Every push/PR | Yes | ~2 min |
| Docker Build | main branch only | Yes | ~4 min |
| Deploy Staging | main branch only | — | ~3 min |
| Deploy Production | main branch + manual approval | — | ~3 min |

### 10.5 Disaster Recovery

**RTO/RPO Targets:**

| Scenario | RTO (Recovery Time) | RPO (Data Loss) |
|---|---|---|
| Single API instance failure | < 30 seconds (load balancer health check → reroute) | 0 (stateless API) |
| Redis failure | < 5 minutes (restart or failover) | Rate limit counts reset (acceptable — fail-closed on outage per AVAIL-02) |
| PostgreSQL primary failure | < 5 minutes (Fly Postgres automatic failover to replica) | < 5 seconds (streaming replication lag) |
| Full region failure | 30–60 minutes (manual failover; multi-region not in scope for pilot) | < 5 minutes (last backup) |
| Vault unavailability | API degrades — new logins fail (keys cannot be fetched); existing JWT validation continues from in-memory cache for up to 25h | 0 (Vault is not a primary data store) |

**Backup Strategy:**

| Component | Backup Method | Frequency | Retention |
|---|---|---|---|
| PostgreSQL (all schemas) | `pg_dump` per database (includes all tenant schemas) | Every 6 hours | 7 days rolling |
| PostgreSQL WAL | Continuous WAL archiving to S3 | Continuous | 7 days |
| Per-tenant schema backup | `pg_dump -n tenant_{slug}` (GDPR export + restore capability) | Daily | 30 days |
| Audit log cold archive | Scheduled job: export rows where `archived=false AND occurred_at < now()-1year` to S3 JSONL | Daily | Indefinite (cold storage lifecycle: S3 Standard → Glacier after 90 days) |
| Vault | HCP Vault manages its own HA and backup; self-hosted: Vault snapshot API daily | Daily | 30 days |

**Per-Tenant Schema Restore Procedure:**

```bash
# Restore a single tenant schema (e.g., for GDPR data breach isolation)
pg_restore --schema=tenant_acme_corp -d authdb tenant_acme_corp_backup_20260315.dump

# Alternatively, restore to a test database first for verification
createdb authdb_restore_test
pg_restore -d authdb_restore_test tenant_acme_corp_backup_20260315.dump
```

---

## 11. Architecture Decision Records (ADRs)

---

### ADR-001: Tenant Isolation Strategy — Schema-per-Tenant

**Status:** Accepted

**Context:**
The system must serve multiple tenants with strong data isolation. Three isolation models were evaluated for PostgreSQL: Row-Level Security (RLS), Schema-per-Tenant, and Database-per-Tenant. The system must satisfy SOC2 Type II and GDPR requirements, including per-tenant data export and deletion.

**Decision:**
Each tenant receives a dedicated PostgreSQL schema (e.g., `tenant_acme_corp`). All tenant-scoped tables (users, sessions, roles, audit_log, etc.) exist within this schema. The Go middleware sets `search_path` to the tenant schema on every authenticated request.

**Consequences:**
- Pros: Architectural isolation — cross-tenant data leakage via a missing WHERE clause is impossible. Clean GDPR compliance boundary (drop or wipe one schema). Clean SOC2 audit boundary. Easy per-tenant backup and restore with `pg_dump -n`. No application-layer row filtering required.
- Cons: Every database migration must run against every tenant schema — mitigated by the migration runner (US-07b). Schema count grows with tenants — monitor above 5,000 schemas. Connection pool management requires `SET search_path` per request.

**Alternatives Considered:**
- **Row-Level Security (RLS):** Rejected. RLS is enforced by database policies, but a misconfigured policy or a bypass via `SECURITY DEFINER` function exposes all tenant data. For a security-critical multi-tenant system, architectural isolation is preferred over policy-based isolation.
- **Database-per-Tenant:** Rejected. Requires a separate connection pool per tenant database, separate Vault credentials per database, and significantly higher operational overhead. Not cost-effective at pilot scale.

---

### ADR-002: Cross-Tenant User Identity — Single Tenant per User

**Status:** Accepted

**Context:**
The data model must decide whether a user identity can span multiple tenants (e.g., a consultant working across organizations) or whether a user is always strictly scoped to one tenant.

**Decision:**
A user belongs to exactly one tenant. There is no global user identity table. The `users` table lives entirely within the tenant schema. A person who belongs to two real-world organizations must have two separate accounts.

**Consequences:**
- Pros: Dramatically simplifies the data model. No UserIdentity + TenantMembership join tables. Reduces complexity of every auth flow. Consistent with schema-per-tenant isolation (ADR-001).
- Cons: Cross-tenant identity requires a future data model migration (not retrofittable without significant effort). Users with multiple tenant memberships have multiple credentials to manage.

**Alternatives Considered:**
- **Global user identity with tenant memberships:** Rejected for the pilot. Adds a global `users` table (breaking ADR-001's clean schema isolation), complex identity linking flows, and significantly higher data model complexity. Revisit in v2 if demand is confirmed.

---

### ADR-003: Interface — API Only (No Hosted Login UI)

**Status:** Accepted

**Context:**
Authentication systems often provide a hosted login page (e.g., Auth0 Universal Login, Cognito hosted UI). The team evaluated whether to include a hosted UI in the pilot scope.

**Decision:**
The system exposes only REST API endpoints. No hosted login page is provided. Consuming applications are responsible for building their own login forms. The OAuth 2.0 `/oauth/authorize` endpoint returns a JSON response (not an HTML redirect page).

**Consequences:**
- Pros: Removes 2–4 sprints of frontend scope. Forces clean API-first design. Allows consuming apps full control over branding and UX.
- Cons: Consuming applications must build login UI. OAuth 2.0 Authorization Code flow requires the consuming app to handle the code delivery (our API returns the code in JSON; the app performs the redirect itself). More integration work for API consumers.

**Alternatives Considered:**
- **Hosted login page (Next.js):** Rejected for the pilot. The 0.5 frontend resource is only sufficient for the OAuth consent mechanism. Full hosted login would require 2+ frontend sprints and a branding/UX design phase. Deferred to post-MVP.

---

### ADR-004: Token Strategy — JWT RS256 Access Tokens + Opaque Refresh Tokens

**Status:** Accepted

**Context:**
The token strategy must balance security, performance, and resource server autonomy. The system serves multiple resource servers that need to validate tokens without calling back to the auth server on every request.

**Decision:**
- Access tokens: JWT signed with RS256 (asymmetric), 15-minute TTL. `kid` in header enables multi-key support for zero-downtime rotation.
- Refresh tokens: Cryptographically random opaque strings (32 bytes, hex or base64url encoded). Stored as SHA-256 hash in the database. Rotated on every use. Family-based revocation on reuse detection.

**Consequences:**
- Pros: Resource servers validate JWTs locally using the public key (JWKS) — no auth server call on every request (scalable). Private key stays exclusively in the auth service. RS256 supports multi-service token validation without sharing secrets. Opaque refresh tokens are unguessable and revocable at the database level.
- Cons: Access token revocation is not immediate — a revoked user's access token remains valid for up to 15 minutes (documented trade-off, acceptable for SOC2 CC6.3 which specifies "within one access token TTL"). Opaque refresh tokens require a database lookup on every refresh.

**Alternatives Considered:**
- **HS256 (symmetric JWT):** Rejected. Requires sharing the signing secret with all resource servers — a breach of any resource server compromises all tokens.
- **Session cookies (fully opaque, stateful):** Rejected. Does not support OAuth 2.0 flows, requires server-side session storage for all tokens, and does not scale as cleanly to multiple resource servers.

---

### ADR-005: Session Expiry — Configurable per Tenant, Default Absolute 24h

**Status:** Accepted

**Context:**
Different tenants have different security requirements. A high-security financial services tenant may want 1-hour sessions; a consumer app may want 30-day sessions. The system needs a secure default with per-tenant flexibility.

**Decision:**
Session expiry is configurable per tenant via `tenant_config`. The default is absolute expiry at 24 hours from initial login, regardless of activity. Sliding expiry (extend TTL on each refresh) is opt-in per tenant. Absolute maximum is enforced even with sliding expiry enabled.

**Consequences:**
- Pros: Secure default (24h limits attack window from stolen refresh tokens). Tenant customization satisfies diverse compliance requirements. Absolute expiry ensures no session lasts indefinitely.
- Cons: Some users will be logged out mid-session if they are active for more than 24 hours continuously. Tenant admins must be educated on the trade-off between security and UX.

**Alternatives Considered:**
- **Fixed expiry (no per-tenant config):** Rejected. SOC2 requirements vary by industry; a single TTL cannot satisfy all tenant compliance needs.
- **Sliding expiry as default:** Rejected. Sliding expiry can result in indefinitely long sessions if the user is continuously active, which is a security risk for stolen tokens.

---

### ADR-006: Social Login Account Linking — Verify Password Before Link

**Status:** Accepted

**Context:**
When a user attempts to log in with Google and the Google email matches an existing password account, there are multiple possible behaviors: auto-link (insecure), block (poor UX), or prompt for password verification before linking.

**Decision:**
If a Google-authenticated email matches an existing password account, the auth system returns a `linking_required` response with a short-lived linking token (10-minute TTL). The consuming app prompts the user to enter their password. Upon successful password verification, the Google account is linked to the existing account and a session is issued.

**Consequences:**
- Pros: Prevents account takeover — an attacker who creates a Google account with someone else's email cannot gain access to the victim's existing account. User explicitly consents to account linking.
- Cons: Slightly more complex user flow for account linking. Consuming app must handle the linking_required response and present a password prompt.

**Alternatives Considered:**
- **Auto-link (match email, no verification):** Rejected. Critical security vulnerability — anyone can create a Google account with a target's email and gain access to their account.
- **Block login (require separate accounts):** Rejected. Unacceptable UX — users expect to log in with Google if they registered with the same email address.

---

### ADR-007: Audit Log Retention — 1 Year Hot + Cold Archive

**Status:** Accepted

**Context:**
SOC2 Type II requires demonstrating that authentication events are logged and retained. GDPR requires the ability to export and delete user data. The team must decide how long audit logs are kept in the primary database and when they are archived.

**Decision:**
Audit logs are retained in the primary PostgreSQL tenant schema for 1 year ("hot" retention). After 1 year, a scheduled job exports records to S3-compatible cold storage (JSONL format, Parquet optional for analytics). The cold storage lifecycle policy transitions to Glacier (or equivalent) after 90 days. Audit log entries are append-only at the application layer — no UPDATE or DELETE operations in code. GDPR: user PII in audit log entries is anonymized (actor_id/target_user_id replaced with tombstone UUID) on user deletion, not deleted.

**Consequences:**
- Pros: Satisfies SOC2 Type II evidence requirements. Cold archive is cost-effective for long-term retention. GDPR-compatible via anonymization rather than deletion (retains audit integrity while removing PII).
- Cons: Cold archive export job must be implemented and monitored. Schema-per-tenant means the export job must iterate all tenant schemas.

**Alternatives Considered:**
- **Retain all audit logs in PostgreSQL indefinitely:** Rejected. Table bloat at scale; high storage cost; no performance differentiation between hot and cold data.
- **Delete audit logs after 1 year:** Rejected. SOC2 Type II typically requires 2-3 years of evidence. Cold archive is a better trade-off.

---

### ADR-008: SOC2 — Post-Launch Goal, Evidence Collection Starts at Launch

**Status:** Accepted

**Context:**
SOC2 Type II certification requires a minimum audit period (typically 3–6 months) of demonstrated controls. The team must decide when to start evidence collection and when to engage a compliance consultant.

**Decision:**
SOC2 Type II certification is a post-launch goal. Evidence collection (audit logs, access controls, change management records) begins at launch. A compliance consultant is engaged by Sprint 2 to ensure the architecture satisfies SOC2 Trust Service Criteria before controls are implemented. The consultant maps controls to the system's implemented features.

**Consequences:**
- Pros: Compliance consultant engagement early prevents retrofitting. Evidence collection from launch means certification is achievable within 6–9 months post-launch.
- Cons: SOC2 Type II certificate is not available at launch — this must be communicated to enterprise prospects.

**Alternatives Considered:**
- **SOC2 Type I first (point-in-time):** Acceptable stepping stone. SOC2 Type I can be achieved at launch if the compliance consultant certifies controls are in place. Recommend pursuing Type I at launch, Type II after 3-month audit period.

---

### ADR-009: Secrets Management — Dedicated Secrets Manager

**Status:** Accepted

**Context:**
The system handles highly sensitive secrets: JWT signing keys, database credentials, OAuth client secrets, API keys for email providers and external IdPs. These must not be stored in environment variables, committed config files, or application code.

**Decision:**
All secrets are stored in HashiCorp Vault (HCP Vault Dedicated for pilot; self-hosted Vault OSS is acceptable). The Go application authenticates to Vault using AppRole (or Kubernetes auth if deployed on K8s). Secrets are fetched at runtime via the Vault SDK over mTLS. No secrets are written to disk, environment variables, or logs. For teams deploying on AWS: AWS Secrets Manager is an acceptable alternative.

**Consequences:**
- Pros: Centralized secret management with fine-grained access control. Vault audit log records every secret access. Dynamic secrets capability (DB credentials that auto-rotate). Zero-downtime key rotation via Vault Transit secrets engine. Eliminates secret sprawl.
- Cons: Vault is an additional operational component. The application cannot start if Vault is unreachable (by design — fail-safe). AppRole credentials themselves must be bootstrapped securely (handled by DevOps during infrastructure provisioning).

**Alternatives Considered:**
- **Environment variables:** Rejected. Secrets in env vars are visible to all processes, logged in many deployment platforms, and cannot be rotated without restarting the application.
- **Config files (encrypted):** Rejected. Key management for the encryption key is circular; config files risk being committed to version control.

---

### ADR-010 (New): Tenant Identification via X-Tenant-ID Header

**Status:** Accepted

**Context:**
The PRD noted that tenant routing on the login endpoint needs a decision: subdomain (e.g., `acme.auth.example.com`), request header, or request body field. Each approach has different operational implications.

**Decision:**
Tenant is identified via the `X-Tenant-ID` HTTP header (containing the tenant UUID or slug). This applies to all requests — both authenticated and unauthenticated. For authenticated requests, the middleware validates that the `X-Tenant-ID` header matches the `tenant_id` claim in the JWT.

**Consequences:**
- Pros: Single deployment (no wildcard DNS required). Easy for API consumers to implement. Compatible with API-only design (ADR-003). Consistent across all endpoints.
- Cons: API consumers must always include the header. Slightly more verbose than subdomain-based routing.

**Alternatives Considered:**
- **Subdomain routing (`acme.auth.example.com`):** Rejected. Requires wildcard TLS certificate and wildcard DNS record. Adds operational complexity for the pilot. More appropriate for a hosted SaaS product with branded domains.
- **Request body field (`"tenant_id": "acme-corp"`):** Rejected. Cannot be used on GET requests or non-JSON requests (e.g., OAuth form-encoded endpoints). Inconsistent API contract.

---

### ADR-011 (New): OAuth 2.0 Library — ory/fosite

**Status:** Accepted

**Context:**
The PRD and Project Management Plan both recommend using a battle-tested OAuth library rather than implementing the spec from scratch. Sprint 5 has a strict timebox and OAuth edge cases are a known source of overrun.

**Decision:**
Use `github.com/ory/fosite` as the OAuth 2.0 / OIDC authorization server library. Fosite provides RFC-compliant implementations of Authorization Code + PKCE, Client Credentials, Refresh Token grant, Token Introspection (RFC 7662), and Token Revocation (RFC 7009). The team implements custom storage adapters (PostgreSQL + Redis) that satisfy fosite's storage interfaces.

**Consequences:**
- Pros: RFC 6749, 7636, 7662, 7009 compliance handled by the library. PKCE S256 enforcement built-in. Security edge cases (code reuse, PKCE bypass attempts) handled. Significantly reduced Sprint 5 scope risk.
- Cons: fosite has a steep learning curve for the storage adapter interface. Library opinions on request/response structure must be followed — some customization requires forking or middleware.

**Alternatives Considered:**
- **go-oauth2/oauth2:** Rejected. Incomplete PKCE and introspection support. Less actively maintained.
- **Custom implementation:** Rejected per PRD risk note. OAuth 2.0 has many security-critical edge cases (PKCE bypass, authorization code injection, open redirect) that battle-tested libraries handle correctly. Custom implementations have a poor track record in security audits.

---

### ADR-012 (New): Async Email Sending via In-Process Worker

**Status:** Accepted

**Context:**
Email sending (verification, password reset, invitation) must not block the HTTP request path. The team must decide between an in-process async approach (Go channels) and an external message queue (RabbitMQ, Redis Streams, SQS).

**Decision:**
For the pilot, email sending is decoupled from the request path via a buffered Go channel and an in-process worker goroutine pool. The handler enqueues an email task to the channel and returns immediately. The worker reads from the channel and calls the Resend API with exponential backoff retry.

**Consequences:**
- Pros: Zero additional infrastructure for the pilot. Simple implementation. Adequate reliability for low-to-medium email volume (thousands per day). Emails survive transient API failures via retry.
- Cons: If the application instance crashes, queued emails are lost (at-most-once delivery). For the pilot scale, this is acceptable — the user can re-request verification or reset emails.

**Alternatives Considered:**
- **Redis Streams / external queue:** Recommended for production scale. Provides at-least-once delivery and survives instance restarts. Defer to post-pilot when email volume justifies the operational complexity.
- **Synchronous email sending:** Rejected. Adds 100–500ms to registration and password reset response times. Blocks on email provider outages.

---

*End of Solution Architecture Document v1.0*
*Authentication System — Pilot Project*
*Ready for handoff to: Developer Agent, Tester Agent, DevOps Engineer*
