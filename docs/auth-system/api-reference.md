# tigersoft-auth — API Reference

**Base URL:** `https://auth.tgstack.dev`
**API Version:** v1
**Protocol:** HTTPS only
**Content-Type:** `application/json` (unless noted)

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Multi-Tenancy](#multi-tenancy)
4. [Rate Limiting](#rate-limiting)
5. [Error Format](#error-format)
6. [Endpoints](#endpoints)
   - [Health Check](#health-check)
   - [JWKS](#jwks)
   - [Auth — Register / Login / Logout](#auth)
   - [Auth — Email Verification](#email-verification)
   - [Auth — Password Reset](#password-reset)
   - [Auth — Token Refresh](#token-refresh)
   - [Auth — Google OAuth](#google-oauth)
   - [User Profile](#user-profile)
   - [MFA (TOTP)](#mfa)
   - [Admin — Users](#admin-users)
   - [Admin — Roles](#admin-roles)
   - [Admin — Audit Log](#admin-audit-log)
   - [Admin — Tenants](#admin-tenants)
   - [OAuth 2.0 / M2M](#oauth-20)
7. [JWT Token Structure](#jwt-token-structure)
8. [Webhook Events (Audit)](#audit-event-types)
9. [Environment Configuration](#environment-configuration)

---

## Overview

tigersoft-auth is a **multi-tenant authentication service** providing:

- Email/password registration and login
- RS256 JWT access tokens + rotating refresh tokens
- TOTP-based MFA (Google Authenticator compatible)
- Google Social Login (OIDC)
- OAuth 2.0 Authorization Code flow with PKCE
- Machine-to-Machine (M2M) client credentials
- Role-based access control (RBAC) per tenant
- GDPR-compliant user erasure
- Schema-per-tenant PostgreSQL isolation

All tokens are signed with RSA-2048 and can be verified independently using the public keys at `/.well-known/jwks.json`.

---

## Authentication

Protected endpoints require a **Bearer token** in the `Authorization` header:

```http
Authorization: Bearer <access_token>
```

Tokens expire after **15 minutes** (configurable). Use the [Token Refresh](#token-refresh) endpoint before expiry.

### Verifying tokens independently

Any service can verify issued tokens without calling back to the auth server:

```bash
# Fetch public keys
curl https://auth.tgstack.dev/.well-known/jwks.json
```

Use any JWT RS256 library with the returned public key. Check the `kid` header in the JWT against the matching key in the JWKS.

---

## Multi-Tenancy

Every request that involves a user **must include a tenant identifier**. The service resolves the tenant in this order:

1. `tenant_id` claim in the JWT (preferred — set after login)
2. `X-Tenant-ID` header (required for unauthenticated endpoints like register/login)

```http
X-Tenant-ID: <tenant_uuid_or_slug>
```

**If the tenant is not found or is suspended, all requests return `403 INVALID_TENANT`.**

---

## Rate Limiting

Rate limits are enforced per-IP and per-user via Redis. Exceeding a limit returns `429 Too Many Requests`.

| Endpoint | IP Limit | User Limit |
|---|---|---|
| `POST /auth/login` | 20/min | 5/min |
| `POST /auth/register` | 10/min | — |
| `POST /auth/forgot-password` | 5/min | 3/hour |
| `POST /auth/token/refresh` | 60/min | 30/min |
| `POST /auth/verify-email` | 10/min | — |
| TOTP verify | — | locked 15min after 5 failures |

---

## Error Format

All errors return a consistent JSON envelope:

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Email or password is incorrect.",
    "details": [
      { "field": "email", "reason": "invalid format" }
    ],
    "request_id": "req_01HXZ...",
    "timestamp": "2026-03-03T10:00:00Z"
  }
}
```

### Common Error Codes

| Code | HTTP | Description |
|---|---|---|
| `UNAUTHORIZED` | 401 | Missing or invalid token |
| `FORBIDDEN` | 403 | Token valid but insufficient permissions |
| `NOT_FOUND` | 404 | Resource does not exist |
| `INVALID_REQUEST` | 400 | Malformed request body |
| `VALIDATION_ERROR` | 422 | Field validation failed |
| `EMAIL_ALREADY_EXISTS` | 409 | Email already registered |
| `INVALID_CREDENTIALS` | 401 | Wrong email or password |
| `EMAIL_NOT_VERIFIED` | 403 | Account email not yet verified |
| `ACCOUNT_DISABLED` | 403 | User disabled by admin |
| `ACCOUNT_LOCKED` | 403 | Locked after too many failed logins |
| `INVALID_TENANT` | 403 | Tenant not found or suspended |
| `TENANT_MISMATCH` | 403 | JWT tenant differs from X-Tenant-ID header |
| `MFA_REQUIRED` | 202 | Login incomplete — TOTP code needed |
| `INVALID_TOTP_CODE` | 422 | TOTP code incorrect or expired |
| `TOTP_RATE_LIMITED` | 429 | Too many TOTP attempts — wait 15 min |
| `MFA_ALREADY_ENABLED` | 409 | MFA already active |
| `MFA_NOT_ENABLED` | 409 | MFA not active |
| `ROLE_NOT_FOUND` | 404 | Role does not exist |
| `ROLE_ALREADY_ASSIGNED` | 409 | User already has this role |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Endpoints

---

### Health Check

```http
GET /health
```

Returns the health of downstream dependencies. Used by load balancers and readiness probes.

**Response `200 OK`**
```json
{
  "status": "ok",
  "checks": {
    "postgres": "ok",
    "redis": "ok"
  }
}
```

**Response `503 Service Unavailable`**
```json
{
  "status": "degraded",
  "checks": {
    "postgres": "error",
    "redis": "ok"
  }
}
```

---

### JWKS

```http
GET /.well-known/jwks.json
```

Returns the RSA public keys used to verify JWTs. Rotate-safe: multiple keys may be returned when a key rotation is in progress. Match using the `kid` header in the JWT.

**Response `200 OK`**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "key-prod-1",
      "n": "<base64url modulus>",
      "e": "AQAB"
    }
  ]
}
```

---

### Auth

#### Register

```http
POST /api/v1/auth/register
X-Tenant-ID: <tenant>
```

Creates a new user account. Sends an email verification link.

**Request**
```json
{
  "email": "alice@example.com",
  "password": "Secure@Password1",
  "first_name": "Alice",
  "last_name": "Smith"
}
```

| Field | Type | Rules |
|---|---|---|
| `email` | string | Required, valid email |
| `password` | string | Required, 8–128 chars |
| `first_name` | string | Required, 1–100 chars |
| `last_name` | string | Required, 1–100 chars |

**Response `201 Created`**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "status": "pending_verification",
  "message": "Registration successful. Please check your email to verify your account."
}
```

**Errors:** `400 INVALID_REQUEST`, `409 EMAIL_ALREADY_EXISTS`, `403 INVALID_TENANT`

---

#### Login

```http
POST /api/v1/auth/login
X-Tenant-ID: <tenant>
```

Authenticates a user and returns a token pair.

**Request**
```json
{
  "email": "alice@example.com",
  "password": "Secure@Password1",
  "totp_code": "123456"
}
```

| Field | Type | Rules |
|---|---|---|
| `email` | string | Required |
| `password` | string | Required |
| `totp_code` | string | Optional. Required if MFA is enabled |

**Response `200 OK`**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS1wcm9kLTEifQ...",
  "refresh_token": "def50200a8b9c...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_expires_in": 86400
}
```

**Response `202 Accepted` — MFA code required (not an error)**
```json
{
  "mfa_required": true,
  "message": "TOTP code required to complete login."
}
```
Resend the same request body with `totp_code` populated.

**Errors:** `401 INVALID_CREDENTIALS`, `403 EMAIL_NOT_VERIFIED`, `403 ACCOUNT_DISABLED`, `403 ACCOUNT_LOCKED`, `403 INVALID_TENANT`

---

#### Logout

```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

Revokes the provided refresh token (single session).

**Request**
```json
{
  "refresh_token": "def50200a8b9c..."
}
```

**Response `200 OK`**
```json
{
  "message": "Logged out successfully."
}
```

---

#### Logout All Sessions

```http
POST /api/v1/auth/logout/all
Authorization: Bearer <access_token>
```

Revokes **all** refresh tokens for the current user.

**Response `200 OK`**
```json
{
  "message": "All sessions revoked successfully.",
  "sessions_revoked": 3
}
```

---

### Email Verification

#### Verify Email

```http
POST /api/v1/auth/verify-email
X-Tenant-ID: <tenant>
```

**Request**
```json
{
  "token": "<token from verification email>"
}
```

**Response `200 OK`**
```json
{
  "message": "Email verified successfully."
}
```

**Errors:** `401 UNAUTHORIZED` (invalid or expired token)

---

#### Resend Verification Email

```http
POST /api/v1/auth/resend-verification
X-Tenant-ID: <tenant>
```

Always returns `200` regardless of whether the email exists, to prevent enumeration.

**Request**
```json
{
  "email": "alice@example.com"
}
```

**Response `200 OK`**
```json
{
  "message": "If an account with that email exists, a new verification email has been sent."
}
```

---

### Password Reset

#### Forgot Password

```http
POST /api/v1/auth/forgot-password
X-Tenant-ID: <tenant>
```

Always returns `200` to prevent enumeration. Sends a reset link valid for 1 hour.

**Request**
```json
{
  "email": "alice@example.com"
}
```

**Response `200 OK`**
```json
{
  "message": "If an account with that email exists, a password reset link has been sent."
}
```

---

#### Reset Password

```http
POST /api/v1/auth/reset-password
X-Tenant-ID: <tenant>
```

**Request**
```json
{
  "token": "<token from reset email>",
  "new_password": "NewSecure@Password2"
}
```

| Field | Type | Rules |
|---|---|---|
| `token` | string | Required |
| `new_password` | string | Required, 8–128 chars |

**Response `200 OK`**
```json
{
  "message": "Password has been reset successfully. Please log in with your new password."
}
```

**Errors:** `401 UNAUTHORIZED` (invalid or expired token)

---

### Token Refresh

```http
POST /api/v1/auth/token/refresh
X-Tenant-ID: <tenant>
```

Issues a new access token and **rotates** the refresh token. The old refresh token is invalidated immediately. Reusing a revoked token triggers `SUSPICIOUS_TOKEN_REUSE` and revokes all sessions.

**Request**
```json
{
  "refresh_token": "def50200a8b9c..."
}
```

**Response `200 OK`**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS1wcm9kLTEifQ...",
  "refresh_token": "new-rotated-token...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Errors:** `401 INVALID_REFRESH_TOKEN`, `401 SUSPICIOUS_TOKEN_REUSE`

---

### Google OAuth

#### Initiate Google Login

```http
POST /api/v1/auth/oauth/google
X-Tenant-ID: <tenant>  (optional)
```

Returns a Google authorization URL. Redirect the user's browser to this URL.

**Request**
```json
{
  "redirect_uri": "https://app.example.com/auth/callback"
}
```

**Response `200 OK`**
```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=...&state=..."
}
```

**Errors:** `400 GOOGLE_NOT_CONFIGURED`

---

#### Google OAuth Callback

```http
GET /api/v1/auth/oauth/google/callback?code=...&state=...&tenant_id=...
```

Google redirects the user's browser here after consent. Returns tokens or redirects to your app.

| Param | Description |
|---|---|
| `code` | Authorization code from Google |
| `state` | CSRF state token (must match what was issued) |
| `tenant_id` | Optional — tenant context |
| `password` | Optional — for linking existing account to Google |

**Response `200 OK`**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS1wcm9kLTEifQ...",
  "refresh_token": "def50200a8b9c...",
  "token_type": "Bearer",
  "expires_in": 900,
  "is_new_user": true
}
```

**Errors:** `400 INVALID_REQUEST`, `401 INVALID_STATE`, `409 PASSWORD_REQUIRED` (existing account needs linking)

---

### User Profile

All routes require `Authorization: Bearer <access_token>`.

#### Get Profile

```http
GET /api/v1/users/me
Authorization: Bearer <access_token>
```

**Response `200 OK`**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "first_name": "Alice",
  "last_name": "Smith",
  "mfa_enabled": false,
  "created_at": "2026-01-15T10:30:00Z",
  "tenant_id": "tenant-uuid",
  "roles": ["user"]
}
```

---

#### Update Profile

```http
PUT /api/v1/users/me
Authorization: Bearer <access_token>
```

One operation per request: update name, change password, or change email.

**Update name**
```json
{
  "first_name": "Alice",
  "last_name": "Jones"
}
```
**Response `200 OK`** — Returns updated profile object.

---

**Change password** — Revokes all other sessions on success.
```json
{
  "current_password": "OldPassword123!",
  "new_password": "NewPassword456!"
}
```
**Response `200 OK`**
```json
{ "message": "Password changed successfully. All other sessions have been revoked." }
```

---

**Change email** — Sends verification to new address; change takes effect after verification.
```json
{
  "new_email": "alice.new@example.com"
}
```
**Response `200 OK`**
```json
{ "message": "A verification email has been sent to the new address. Complete verification to apply the change." }
```

**Errors:** `401 INVALID_CREDENTIALS`, `409 EMAIL_ALREADY_EXISTS`, `422 INVALID_EMAIL`

---

#### Delete Account (GDPR Self-Erasure)

```http
DELETE /api/v1/users/me
Authorization: Bearer <access_token>
```

Permanently deletes the account. PII is anonymized, sessions revoked, MFA removed.

**Request**
```json
{
  "password": "YourCurrentPassword123!"
}
```

**Response `204 No Content`**

The access token becomes invalid immediately after this call.

---

### MFA

All routes require `Authorization: Bearer <access_token>`.

#### Generate TOTP Secret

```http
POST /api/v1/users/me/mfa/generate
Authorization: Bearer <access_token>
```

Returns a secret and `otpauth://` URL. Render the URL as a QR code for the user to scan with their authenticator app. The secret is **not yet activated** — call `/mfa/confirm` to enable it.

**Response `200 OK`**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "otp_auth_url": "otpauth://totp/tigersoft-auth:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=tigersoft-auth",
  "message": "Scan the QR code or enter the secret in your authenticator app, then call /mfa/confirm with a valid code."
}
```

**Errors:** `409 MFA_ALREADY_ENABLED`

---

#### Confirm TOTP Enrollment

```http
POST /api/v1/users/me/mfa/confirm
Authorization: Bearer <access_token>
```

Activates MFA using a valid TOTP code from the authenticator app. Returns one-time backup codes — **show them to the user once only**.

**Request**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "code": "123456"
}
```

| Field | Type | Rules |
|---|---|---|
| `secret` | string | Required, min 16 chars |
| `code` | string | Required, exactly 6 digits |

**Response `200 OK`**
```json
{
  "backup_codes": [
    "TIGER-AABB-1234",
    "TIGER-CCDD-5678",
    "TIGER-EEFF-9012"
  ],
  "message": "MFA has been enabled. Save these backup codes securely — they will not be shown again."
}
```

**Errors:** `409 MFA_ALREADY_ENABLED`, `422 INVALID_TOTP_CODE`, `429 TOTP_RATE_LIMITED`

---

#### Disable MFA

```http
DELETE /api/v1/users/me/mfa
Authorization: Bearer <access_token>
```

**Request**
```json
{
  "password": "YourCurrentPassword123!"
}
```

**Response `204 No Content`**

**Errors:** `401 INVALID_CREDENTIALS`, `409 MFA_NOT_ENABLED`

---

### Admin Users

All routes require `Authorization: Bearer <access_token>` with the `admin` role.

#### Invite User

```http
POST /api/v1/admin/users/invite
Authorization: Bearer <access_token>  (admin role)
```

Creates an account and sends an invitation email with a setup link.

**Request**
```json
{
  "email": "bob@example.com",
  "first_name": "Bob",
  "last_name": "Brown"
}
```

**Response `201 Created`**
```json
{
  "user_id": "uuid",
  "email": "bob@example.com",
  "status": "invited"
}
```

---

#### List Users

```http
GET /api/v1/admin/users?limit=20&offset=0
Authorization: Bearer <access_token>  (admin role)
```

**Response `200 OK`**
```json
{
  "data": [
    {
      "user_id": "uuid",
      "email": "alice@example.com",
      "first_name": "Alice",
      "last_name": "Smith",
      "status": "active",
      "created_at": "2026-01-15T10:30:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

---

#### Disable User

```http
PUT /api/v1/admin/users/:user_id/disable
Authorization: Bearer <access_token>  (admin role)
```

**Response `200 OK`**
```json
{ "message": "User has been disabled." }
```

---

#### Delete User (GDPR Erasure)

```http
DELETE /api/v1/admin/users/:user_id
Authorization: Bearer <access_token>  (admin role)
```

Full GDPR erasure: anonymizes PII, revokes all sessions, removes MFA, clears OAuth links.

**Response `204 No Content`**

---

### Admin Roles

All routes require `admin` role.

#### Create Role

```http
POST /api/v1/admin/roles
Authorization: Bearer <access_token>  (admin role)
```

**Request**
```json
{
  "name": "editor",
  "description": "Can edit content"
}
```

**Response `201 Created`**
```json
{
  "role_id": "uuid",
  "name": "editor",
  "description": "Can edit content",
  "is_system": false,
  "created_at": "2026-01-15T10:30:00Z"
}
```

---

#### List Roles

```http
GET /api/v1/admin/roles
Authorization: Bearer <access_token>  (admin role)
```

**Response `200 OK`**
```json
{
  "data": [
    { "role_id": "uuid", "name": "admin", "description": "Full tenant access", "is_system": true },
    { "role_id": "uuid", "name": "editor", "description": "Can edit content", "is_system": false }
  ]
}
```

---

#### Assign Role to User

```http
POST /api/v1/admin/users/:user_id/roles
Authorization: Bearer <access_token>  (admin role)
```

**Request**
```json
{
  "role_id": "uuid"
}
```

**Response `200 OK`**
```json
{ "message": "Role assigned successfully." }
```

**Errors:** `404 ROLE_NOT_FOUND`, `409 ROLE_ALREADY_ASSIGNED`

---

#### Remove Role from User

```http
DELETE /api/v1/admin/users/:user_id/roles/:role_id
Authorization: Bearer <access_token>  (admin role)
```

**Response `204 No Content`**

---

#### Update Tenant MFA Policy

```http
PUT /api/v1/admin/tenant/mfa
Authorization: Bearer <access_token>  (admin role)
```

When `mfa_required: true`, users who haven't enrolled in MFA will be blocked at login until they do.

**Request**
```json
{
  "mfa_required": true
}
```

**Response `200 OK`**
```json
{
  "mfa_required": true,
  "message": "MFA enforcement setting updated."
}
```

---

### Admin Audit Log

```http
GET /api/v1/admin/audit-log
Authorization: Bearer <access_token>  (admin role)
```

**Query Parameters**

| Param | Type | Description |
|---|---|---|
| `limit` | int | Max 500, default 50 |
| `offset` | int | Pagination offset, default 0 |
| `event_type` | string | Filter by event type (see below) |
| `actor_id` | uuid | Filter by acting user |
| `target_user_id` | uuid | Filter by target user |
| `from` | RFC3339 | Start of time range |
| `to` | RFC3339 | End of time range |

**Example**
```
GET /api/v1/admin/audit-log?event_type=LOGIN&from=2026-03-01T00:00:00Z&limit=100
```

**Response `200 OK`**
```json
{
  "data": [
    {
      "id": "uuid",
      "event_type": "LOGIN",
      "actor_id": "user-uuid",
      "target_user_id": "user-uuid",
      "actor_ip": "203.0.113.42",
      "metadata": { "user_agent": "Mozilla/5.0..." },
      "occurred_at": "2026-03-03T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

---

### Admin Tenants

All routes require `super_admin` role unless noted.

#### Provision Tenant

```http
POST /api/v1/admin/tenants
Authorization: Bearer <access_token>  (super_admin role)
```

Creates a new tenant with an isolated PostgreSQL schema and sends an invitation to the admin email.

**Request**
```json
{
  "name": "Acme Corporation",
  "slug": "acme",
  "admin_email": "admin@acme.com"
}
```

| Field | Type | Rules |
|---|---|---|
| `name` | string | Required, 2–100 chars |
| `slug` | string | Required, 3–50 chars, URL-safe |
| `admin_email` | string | Required, valid email |

**Response `201 Created`**
```json
{
  "tenant_id": "uuid",
  "name": "Acme Corporation",
  "slug": "acme",
  "schema_name": "tenant_acme_abc123",
  "status": "active",
  "created_at": "2026-01-15T10:30:00Z"
}
```

---

#### Get Tenant

```http
GET /api/v1/admin/tenants/:tenant_id
Authorization: Bearer <access_token>  (super_admin role)
```

**Response `200 OK`** — Returns same shape as provision response.

---

#### List Tenants

```http
GET /api/v1/admin/tenants?limit=20&offset=0
Authorization: Bearer <access_token>  (super_admin role)
```

**Response `200 OK`**
```json
{
  "data": [
    { "tenant_id": "uuid", "name": "Acme Corporation", "slug": "acme", "status": "active", "created_at": "..." }
  ],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

---

#### Suspend Tenant

```http
PUT /api/v1/admin/tenants/:tenant_id/suspend
Authorization: Bearer <access_token>  (super_admin role)
```

Blocks all logins for the tenant. Existing sessions remain valid until they expire.

**Response `200 OK`**
```json
{ "message": "Tenant has been suspended." }
```

---

#### Generate M2M Credentials

```http
POST /api/v1/admin/tenants/:tenant_id/credentials
Authorization: Bearer <access_token>  (super_admin role)
```

Creates client credentials for machine-to-machine access (service accounts, CI/CD pipelines, etc.).

**Response `201 Created`**
```json
{
  "client_id": "client_acme_xyz789",
  "client_secret": "secret_verylongrandomstring...",
  "warning": "This is the only time the client_secret will be shown. Store it securely."
}
```

---

#### Rotate M2M Credentials

```http
POST /api/v1/admin/tenants/:tenant_id/credentials/rotate
Authorization: Bearer <access_token>  (super_admin role)
```

Immediately revokes the previous `client_secret` and issues a new one.

**Response `201 Created`** — Same shape as Generate response.

---

### OAuth 2.0

#### Register OAuth Client

```http
POST /api/v1/admin/oauth/clients
Authorization: Bearer <access_token>  (admin role)
```

Registers a new OAuth client (e.g., a web app, mobile app).

**Request**
```json
{
  "name": "Mobile App",
  "redirect_uris": ["https://app.example.com/callback"],
  "scopes": ["openid", "profile", "email"]
}
```

**Response `201 Created`**
```json
{
  "client_id": "client_mobile_abc123",
  "client_secret": "secret_verylongrandomstring...",
  "name": "Mobile App",
  "warning": "Store client_secret securely — it will not be shown again."
}
```

---

#### Authorization Endpoint (PKCE recommended)

```http
GET /api/v1/oauth/authorize
Authorization: Bearer <access_token>
```

**Query Parameters**

| Param | Required | Description |
|---|---|---|
| `client_id` | Yes | Registered client ID |
| `redirect_uri` | Yes | Must match registered URI |
| `response_type` | Yes | Only `code` supported |
| `state` | Yes | CSRF token from your app |
| `scope` | No | Space-separated: `openid profile email` |
| `code_challenge` | Recommended | PKCE code challenge |
| `code_challenge_method` | Recommended | `S256` or `plain` |

**Response `200 OK`**
```json
{
  "code": "auth_code_xyz789",
  "state": "<state_from_request>"
}
```

**Errors:** `400 INVALID_REQUEST`, `401 INVALID_CLIENT`, `400 INVALID_REDIRECT_URI`, `400 INVALID_SCOPE`

---

#### Token Endpoint — Authorization Code

```http
POST /api/v1/oauth/token
Content-Type: application/json
```

**Request**
```json
{
  "grant_type": "authorization_code",
  "code": "auth_code_xyz789",
  "client_id": "client_mobile_abc123",
  "redirect_uri": "https://app.example.com/callback",
  "code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
}
```

**Response `200 OK`**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS1wcm9kLTEifQ...",
  "refresh_token": "def50200a8b9c...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "openid profile"
}
```

**Errors:** `400 INVALID_GRANT` (expired or PKCE mismatch), `400 INVALID_REQUEST`

---

#### Token Endpoint — Client Credentials (M2M)

```http
POST /api/v1/oauth/token
Content-Type: application/json
```

Used by backend services and automated pipelines to obtain tokens without user interaction.

**Request**
```json
{
  "grant_type": "client_credentials",
  "client_id": "client_acme_xyz789",
  "client_secret": "secret_verylongrandomstring...",
  "scope": "api"
}
```

**Response `200 OK`**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS1wcm9kLTEifQ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "api"
}
```

Note: No `refresh_token` is issued for client credentials.

**Errors:** `401 INVALID_CLIENT`, `400 INVALID_SCOPE`

---

#### Token Introspection

```http
POST /api/v1/oauth/introspect
Content-Type: application/json
```

Checks whether a token is active and returns its claims. Used by resource servers that don't perform local JWT verification.

**Request**
```json
{
  "token": "<access_token_or_refresh_token>"
}
```

**Response `200 OK` — Active token**
```json
{
  "active": true,
  "token_type": "Bearer",
  "sub": "user-uuid",
  "jti": "jwt-id",
  "iss": "https://auth.tgstack.dev",
  "iat": 1772534400,
  "exp": 1772535300,
  "tenant_id": "tenant-uuid",
  "roles": ["admin"],
  "scope": "openid profile",
  "client_id": "client_mobile_abc123"
}
```

**Response `200 OK` — Inactive/expired token**
```json
{
  "active": false
}
```

---

## JWT Token Structure

```
Header:
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "key-prod-1"        ← match against JWKS
}

Payload:
{
  "jti": "uuid",             ← unique token ID
  "sub": "user-uuid",        ← user ID
  "iss": "https://auth.tgstack.dev",
  "aud": ["https://api.tgstack.dev"],
  "iat": 1772534400,         ← issued at (unix)
  "exp": 1772535300,         ← expiry (unix)
  "tenant_id": "uuid",
  "roles": ["admin", "user"],
  "client_id": "...",        ← OAuth tokens only
  "scope": "openid profile"  ← OAuth tokens only
}
```

### Token Lifetimes

| Token | Default TTL | Env Variable |
|---|---|---|
| Access token | 15 min | `JWT_ACCESS_TOKEN_TTL_SECONDS` |
| Refresh token | 24 hours | `SESSION_DEFAULT_TTL_SECONDS` |
| M2M access token | 60 min | — |
| Email verification | 24 hours | `EMAIL_VERIFICATION_TOKEN_TTL_HOURS` |
| Password reset | 1 hour | `PASSWORD_RESET_TOKEN_TTL_HOURS` |
| OAuth state | 10 min | `OAUTH_STATE_TTL_SECONDS` |

---

## Audit Event Types

Events logged to the audit trail (queryable via `/api/v1/admin/audit-log`):

| Event | Trigger |
|---|---|
| `REGISTER` | New user registered |
| `LOGIN` | Successful login |
| `LOGIN_FAILED` | Failed login attempt |
| `LOGOUT` | Session revoked |
| `LOGOUT_ALL` | All sessions revoked |
| `EMAIL_VERIFIED` | Email verification completed |
| `PASSWORD_RESET_REQUESTED` | Forgot-password triggered |
| `PASSWORD_RESET` | Password successfully reset |
| `PASSWORD_CHANGED` | Password changed in profile |
| `EMAIL_CHANGE_REQUESTED` | New email verification sent |
| `MFA_ENABLED` | TOTP MFA activated |
| `MFA_DISABLED` | TOTP MFA removed |
| `USER_DISABLED` | Admin disabled a user |
| `USER_DELETED` | User account erased (GDPR) |
| `USER_INVITED` | Admin sent invitation |
| `ROLE_ASSIGNED` | Role granted to user |
| `ROLE_REMOVED` | Role removed from user |
| `TENANT_CREATED` | New tenant provisioned |
| `TENANT_SUSPENDED` | Tenant suspended |
| `CREDENTIALS_GENERATED` | M2M credentials created |
| `CREDENTIALS_ROTATED` | M2M credentials rotated |
| `SUSPICIOUS_TOKEN_REUSE` | Refresh token replay detected |

---

## Environment Configuration

### Required

```bash
# Server
PORT=8080
ENV=production                            # production | development
AUTH_SERVICE_BASE_URL=https://auth.tgstack.dev

# Database
DATABASE_URL=postgres://auth_user:pass@central-postgres.database.svc.cluster.local:5432/auth_system

# Cache
REDIS_URL=redis://:password@my-redis-master.redis.svc.cluster.local:6379

# JWT signing key (choose one)
JWT_RSA_PRIVATE_KEY_PEM=-----BEGIN PRIVATE KEY-----...  # K8s Secret / env var
JWT_PRIVATE_KEY_PATH=/path/to/key.pem                   # File path
VAULT_ADDR=https://vault.example.com                    # HashiCorp Vault (dev)

JWT_KEY_ID=key-prod-1
JWT_ISSUER=https://auth.tgstack.dev
JWT_AUDIENCE=https://api.tgstack.dev

# Email (Resend)
RESEND_API_KEY=re_xxxxxxxxxxxx
EMAIL_FROM=noreply@tigersoft.co.th
```

### Optional (with defaults)

```bash
# JWT
JWT_ACCESS_TOKEN_TTL_SECONDS=900          # 15 min
SESSION_DEFAULT_TTL_SECONDS=86400         # 24 hours

# Email tokens
EMAIL_VERIFICATION_TOKEN_TTL_HOURS=24
PASSWORD_RESET_TOKEN_TTL_HOURS=1

# Email workers
EMAIL_WORKER_CONCURRENCY=4
EMAIL_MAX_RETRIES=3

# CORS
CORS_ALLOWED_ORIGINS=https://app.tgstack.dev

# OAuth (Google)
OAUTH_GOOGLE_CLIENT_ID=xxxx.apps.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=xxxx
OAUTH_STATE_TTL_SECONDS=600

# Rate limits
RATE_LIMIT_LOGIN_IP_PER_MINUTE=20
RATE_LIMIT_REGISTER_IP_PER_MINUTE=10
RATE_LIMIT_FORGOT_PASSWORD_IP_PER_MINUTE=5
LOCKOUT_THRESHOLD=5                       # failed logins before lockout
LOCKOUT_DURATION_SECONDS=900              # 15 min lockout

# Database pool
DB_MAX_CONNS=20
DB_MIN_CONNS=5

# Tenant cache
TENANT_CACHE_TTL_SECONDS=60
```

---

## Quick Integration Checklist

- [ ] Register a tenant: `POST /api/v1/admin/tenants` (super_admin required)
- [ ] Register a user: `POST /api/v1/auth/register` with `X-Tenant-ID`
- [ ] Verify the public key: `GET /.well-known/jwks.json`
- [ ] Configure your app to verify JWTs locally (no round-trip needed)
- [ ] For SPA/mobile: use PKCE — `POST /api/v1/admin/oauth/clients` + `/authorize` + `/token`
- [ ] For backend services: use M2M — `POST /api/v1/admin/tenants/:id/credentials` + `client_credentials` grant
- [ ] Set `X-Tenant-ID` on every unauthenticated call
- [ ] Handle `202 MFA_REQUIRED` in your login flow
- [ ] Rotate refresh tokens — store only the most recent token
