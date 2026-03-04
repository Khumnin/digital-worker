# Incident Response Runbook — Auth System

**Version:** 1.0
**Last reviewed:** 2026-03-03
**Owner:** Platform / On-call Engineer

---

## Severity Levels

| Level | Definition | Response Time |
|-------|-----------|---------------|
| SEV-1 | Auth service completely down — all logins failing, /health returns 503 | Immediate (page on-call now) |
| SEV-2 | Partial degradation — Redis down, MFA broken, Google login broken, email delivery failed | 15 min |
| SEV-3 | Performance degraded — p95 login latency > 500 ms, elevated error rate | 1 hour |

---

## On-Call Contacts and Escalation Path

1. **Primary on-call** — Check PagerDuty rotation.
2. **Secondary (escalate after 10 min if no acknowledgment)** — Engineering lead.
3. **Incident commander (SEV-1 only)** — CTO or designated deputy.
4. **Communication channel** — #incidents in Slack. Create an incident thread immediately.

---

## SEV-1 Playbook: Auth Service Completely Down

**Symptoms:** All POST /api/v1/auth/login calls return 5xx or timeout. GET /health returns 503.

### Step 1 — Triage /health

```bash
curl -s https://auth.example.com/health | jq .
```

Expected healthy response:
```json
{ "status": "ok", "checks": { "postgres": "ok", "redis": "ok" } }
```

If `postgres` is `"error"`:
- Check PostgreSQL connection pool (DB_MAX_CONNS in environment).
- Check `fly postgres connect -a auth-system-db` — can you reach the DB?
- Check pg_stat_activity for connection exhaustion:
  ```sql
  SELECT count(*), state FROM pg_stat_activity GROUP BY state;
  ```

If `redis` is `"error"`:
- See SEV-2 Redis playbook below.
- Rate limiting and session operations fail open — logins may still partially work.

### Step 2 — Check Fly.io Machine Status

```bash
fly status -a auth-system
fly logs -a auth-system --tail
```

Look for: OOM kills, crash loops, deployment failures.

### Step 3 — Check PostgreSQL Connection Pool

Environment variable `DB_MAX_CONNS` controls the maximum pool size. If connections are exhausted:

```bash
fly ssh console -a auth-system
# inside the machine:
env | grep DB_
```

Increase `DB_MAX_CONNS` via `fly secrets set DB_MAX_CONNS=50 -a auth-system` and restart.

### Step 4 — Rollback Procedure

If a recent deploy caused the outage:

```bash
# List recent images
fly releases -a auth-system

# Roll back to the previous image tag
fly deploy --image registry.fly.io/auth-system:<previous-tag> -a auth-system
```

Post-rollback: verify GET /health returns `{"status":"ok"}` before closing the incident.

---

## SEV-2 Playbooks: Partial Degradation

### Redis Down

**Impact:** Rate limiting is disabled (fail-open). MFA brute-force protection is unguarded. Token denylist lookups fail — revoked tokens may temporarily appear valid.

**Action:**

```bash
# Check Redis status
fly redis connect  # attempts to connect to the Redis instance
```

```bash
# In Fly.io dashboard: check Redis app status
fly status -a auth-system-redis
```

Steps:
1. Page on-call immediately — this is a security-relevant degradation.
2. Restart the Redis app: `fly machine restart -a auth-system-redis`.
3. Verify the auth service reconnects: watch `fly logs -a auth-system` for Redis reconnection.
4. Monitor failed login spike — rate limiting was disabled. Check audit log.
5. If Redis remains down for > 30 min, consider temporarily disabling MFA-required tenants.

---

### Vault Down

**Impact:** JWT key rotation halts. New key pairs cannot be loaded. Existing tokens remain valid until their 15-minute expiry. No new signing key can be fetched.

**Action:**

```bash
fly logs -a auth-system-vault --tail
fly status -a auth-system-vault
```

Steps:
1. Check if Vault is sealed: `fly ssh console -a auth-system-vault -C "vault status"`.
2. If sealed, unseal using the unseal key(s) stored in your secrets manager.
3. Verify the `kv` mount and JWT key path:
   ```bash
   fly ssh console -a auth-system-vault -C "vault kv list secret/auth-system/"
   ```
4. Restart the auth service to pick up Vault connection: `fly machine restart -a auth-system`.
5. Verify GET /.well-known/jwks.json returns the key set.

---

### Email Worker Stalled

**Impact:** Registration verification emails, password reset emails, and invitation emails are not delivered. Users cannot complete self-service flows.

**Symptoms:** Users report not receiving emails. Mailhog (dev) or Resend (production) shows no sends.

**Action:**

```bash
fly logs -a auth-system | grep -i email
```

Steps:
1. Check the async Go channel buffer — if the worker goroutine has panicked, the process will have logged a panic.
2. Check the Resend API key is valid: `fly secrets list -a auth-system | grep RESEND`.
3. Restart the service to reinitialize the email worker: `fly machine restart -a auth-system`.
4. Spot-test: trigger a POST /api/v1/auth/forgot-password and check Resend dashboard.

---

## SEV-3 Playbook: Performance Degraded (p95 > 500 ms)

**Trigger:** Alert fires on p95 login latency exceeding 500 ms, or error rate > 1%.

### Step 1 — Check /health

```bash
curl -s https://auth.example.com/health | jq .
```

A degraded response indicates a dependency is slow but not fully down.

### Step 2 — Check PostgreSQL Query Performance

```bash
fly postgres connect -a auth-system-db
```

```sql
-- Find long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY duration DESC
LIMIT 10;
```

Check the login query uses an index:
```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM tenant_schema.users WHERE email = 'test@example.com' LIMIT 1;
```

If doing a sequential scan, the index on `email` is missing:
```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON users(email);
```

### Step 3 — Check Redis Latency

```bash
fly ssh console -a auth-system
redis-cli -u $REDIS_URL --latency
```

If latency is high (> 5 ms), check Redis memory usage and eviction policy.

### Step 4 — Check Fly.io Autoscaling

```bash
fly scale show -a auth-system
fly scale count 3 -a auth-system  # temporarily add machines
```

---

## Rollback Procedure

### Application Rollback

```bash
# 1. List available image tags
fly releases -a auth-system

# 2. Deploy the previous image
fly deploy --image registry.fly.io/auth-system:<previous-tag> -a auth-system

# 3. Verify health
curl -s https://auth.example.com/health | jq .
```

### Database Migration Rollback

If a migration caused the incident:

```bash
fly ssh console -a auth-system

# Roll back the most recent tenant migration (1 step down)
go run ./cmd/migrate -scope=tenant -direction=down -steps=1

# Roll back the most recent global migration (1 step down)
go run ./cmd/migrate -scope=global -direction=down -steps=1
```

Notify the team in #incidents before running a down migration — confirm with the SA first.

---

## Key Commands Reference

| Task | Command |
|------|---------|
| Tail application logs | `fly logs -a auth-system` |
| SSH into machine | `fly ssh console -a auth-system` |
| Connect to PostgreSQL | `fly postgres connect -a auth-system-db` |
| Connect to Redis | `fly redis connect` |
| List Fly.io releases | `fly releases -a auth-system` |
| Check machine status | `fly status -a auth-system` |
| Restart machines | `fly machine restart -a auth-system` |
| Scale machines up | `fly scale count N -a auth-system` |
| View secrets (keys only) | `fly secrets list -a auth-system` |
| Set a secret | `fly secrets set KEY=VALUE -a auth-system` |
| Run migrations (up) | `go run ./cmd/migrate -direction=up` |
| Run migrations (down, 1 step) | `go run ./cmd/migrate -direction=down -steps=1` |

---

## Post-Incident Checklist

After every SEV-1 or SEV-2 incident:

- [ ] Root cause identified and documented in #incidents.
- [ ] Timeline written (detection → acknowledgment → resolution).
- [ ] Action items filed as GitHub issues with priority labels.
- [ ] Runbook updated if steps were unclear or missing.
- [ ] Monitoring alert threshold reviewed — did it fire too late?
- [ ] Blameless post-mortem scheduled within 3 business days (SEV-1 only).
