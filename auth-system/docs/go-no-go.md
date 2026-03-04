# Go / No-Go Checklist — Auth System Launch

**Date:** ___________
**Decision maker:** Product Owner + Solution Architect + Project Manager
**Threshold:** ALL Critical items must be checked. Launch is blocked on any unchecked Critical item.

---

## Critical (launch blocked if any fail)

- [ ] `go build ./...` exits 0 — no compilation errors
- [ ] All 67 unit tests pass (`go test ./... -v` in `auth-system/backend/`)
- [ ] All Playwright E2E API tests pass (`npx playwright test --project=api`)
- [ ] `GET /health` returns 200 with `postgres:"ok"` and `redis:"ok"` on the production environment
- [ ] Zero open Critical or High pentest findings (attach pentest report)
- [ ] GDPR right-to-erasure endpoint verified: `DELETE /users/me` returns 204 and subsequent login returns 401
- [ ] Admin erasure verified: `DELETE /admin/users/:id` returns 204 and user data is anonymized
- [ ] JWT RS256 signing keys loaded from Vault (not from disk or environment variable)
- [ ] All DB migrations applied on production: global schema + all active tenant schemas (verify with `SELECT * FROM schema_migrations;`)
- [ ] Rate limiting active: 6 rapid failed logins in 60 seconds triggers account lockout (manually tested)
- [ ] Secure headers present on all responses: `Strict-Transport-Security`, `Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options` (verify with `curl -I`)
- [ ] Cross-tenant isolation test suite passing: `npx playwright test e2e/08-cross-tenant-isolation.spec.ts`
- [ ] Audit log writing events: spot-check that a successful login generates a `LOGIN_SUCCESS` event in the audit log

---

## Important (flag and mitigate — do not block launch)

- [ ] p95 login latency < 300 ms at 100 concurrent users (attach load test results from k6 / hey)
- [ ] Email delivery verified: Resend API key is live; a test registration email was received end-to-end
- [ ] Vault auto-unseal configured for production (not manual unseal keys)
- [ ] Fly.io autoscaling policy set: min 2 machines, max 10, CPU trigger at 70%
- [ ] Monitoring alerts configured in production:
  - [ ] Error rate alert: > 1% 5xx over 5 minutes
  - [ ] p95 latency alert: > 500 ms over 5 minutes
  - [ ] Failed login spike: > 50 failed logins per minute per tenant
- [ ] Rollback procedure tested in staging: `fly deploy --image <previous-tag>` executed successfully in staging within the last sprint
- [ ] GDPR data retention policy enforced: audit logs older than 1 year purged or archived to cold storage
- [ ] `VAULT_ADDR` and `VAULT_TOKEN` are environment-specific and not shared between staging and production

---

## Sign-off

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Product Owner | | | |
| Solution Architect | | | |
| Project Manager | | | |
| QA Lead | | | |

---

## Launch Decision

```
[ ] GO     — All Critical items checked. Important items reviewed and mitigated.
[ ] NO-GO  — One or more Critical items remain unchecked. Launch is blocked.
             Reason: ___________________________________________________
```

**Next review date (if NO-GO):** ___________
