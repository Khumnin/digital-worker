# Auth System Pilot — Project Context

## What This Is
An AI agent squad project. 5 role-based agents (PO, PM, SA, Dev, Tester) in `.claude/agents/` are being used to design and document a multi-tenant Authentication System.

## Agent Squad
```
.claude/agents/
├── product-owner/       INVEST stories, MoSCoW, Gherkin, agile ceremonies
├── project-manager/     Hybrid methodology, WBS, RACI, risk register, budget
├── solution-architect/  Clean arch, C4 diagrams, DDD, cloud-native (opus model)
├── developer/           Next.js + Go/Gin stack, Handler→Service→Repository
└── tester/              ISTQB, ISO 25010, Playwright, quality gates
```

## Current Project: Authentication System Pilot

### Pipeline Status — ✅ ALL COMPLETE
| Agent | Status | Document |
|-------|--------|----------|
| Product Owner | ✅ Done | `docs/auth-system/prd.md` (42KB) |
| Project Manager | ✅ Done | `docs/auth-system/project-management-plan.md` (100KB) |
| Solution Architect | ✅ Done | `docs/auth-system/solution-architecture.md` (131KB) |
| Developer | ✅ Done | `docs/auth-system/implementation-guide.md` (197KB) |
| Tester | ✅ Done | `docs/auth-system/test-plan.md` (235KB) |

### Key Constraints (Non-Negotiable)
- Schema-per-tenant PostgreSQL (ADR-001)
- One user = one tenant (ADR-002)
- API-only, no hosted UI (ADR-003)
- JWT RS256 15min + opaque refresh tokens (ADR-004)
- Tech stack: Go/Gin backend + Next.js frontend + PostgreSQL + Redis
- OAuth library: ory/fosite | Secrets: HashiCorp Vault | Email: Resend | Deploy: Fly.io

### Next Steps
All planning/documentation is complete. Ready for implementation:
1. Set up Go monorepo using `implementation-guide.md` scaffold
2. Run `docker compose up` to bring up PostgreSQL + Redis + Vault + Mailhog
3. Start Sprint 1: Register, Login, Email Verify, Password Reset
