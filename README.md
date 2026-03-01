# 🤖 Digital Worker — AI Agent Squad

An AI-powered agentic project where **5 role-based agents** collaborate to design, plan, architect, implement, and test a production-grade **Multi-Tenant Authentication System**.

---

## 📋 Project Overview

This project uses Claude-based AI agents, each playing a specific professional role, to produce a complete software delivery pipeline — from product requirements through implementation guides and quality assurance test plans.

### 🎯 Current Project: Multi-Tenant Authentication System

A cloud-native, API-first authentication service supporting multi-tenant isolation with enterprise-grade security.

**Key Constraints (Non-Negotiable)**
| Decision | Detail |
|----------|--------|
| ADR-001 | Schema-per-tenant PostgreSQL |
| ADR-002 | One user = one tenant |
| ADR-003 | API-only, no hosted UI |
| ADR-004 | JWT RS256 (15min) + opaque refresh tokens |

---

## 🧑‍💼 Agent Squad

Agents live in `.claude/agents/` and are invoked to generate and maintain project documentation.

| Agent | Role | Methodology |
|-------|------|-------------|
| **Product Owner** | Requirements & Backlog | INVEST stories, MoSCoW, Gherkin, Agile ceremonies |
| **Project Manager** | Planning & Coordination | Hybrid methodology, WBS, RACI, Risk Register, Budget |
| **Solution Architect** | Architecture & Design | Clean Architecture, C4 Diagrams, DDD, Cloud-Native |
| **Developer** | Implementation Guide | Next.js + Go/Gin, Handler→Service→Repository pattern |
| **Tester** | Quality Assurance | ISTQB, ISO 25010, Playwright, Quality Gates |

---

## 🏗️ Tech Stack

```
Backend       Go + Gin
Frontend      Next.js (TypeScript)
Database      PostgreSQL (schema-per-tenant)
Cache/Queue   Redis
Auth Library  ory/fosite
Secrets       HashiCorp Vault
Email         Resend
Deployment    Fly.io
```

---

## 📁 Project Structure

```
digital-worker/
├── .claude/
│   └── agents/
│       ├── product-owner/
│       ├── project-manager/
│       ├── solution-architect/
│       ├── developer/
│       └── tester/
├── docs/
│   └── auth-system/
│       ├── prd.md                      # Product Requirements Document (42KB)
│       ├── project-management-plan.md  # Project Management Plan (100KB)
│       ├── solution-architecture.md    # Solution Architecture (131KB)
│       ├── implementation-guide.md     # Developer Implementation Guide (197KB)
│       └── test-plan.md                # QA Test Plan (235KB)
├── CLAUDE.md                           # Agent squad context & pipeline status
└── README.md
```

---

## ✅ Pipeline Status

| Phase | Agent | Status | Output |
|-------|-------|--------|--------|
| 1. Requirements | Product Owner | ✅ Complete | `docs/auth-system/prd.md` |
| 2. Planning | Project Manager | ✅ Complete | `docs/auth-system/project-management-plan.md` |
| 3. Architecture | Solution Architect | ✅ Complete | `docs/auth-system/solution-architecture.md` |
| 4. Implementation | Developer | ✅ Complete | `docs/auth-system/implementation-guide.md` |
| 5. Testing | Tester | ✅ Complete | `docs/auth-system/test-plan.md` |

---

## 🚀 Getting Started

All planning and documentation is complete. Follow the steps below to begin implementation:

### 1. Scaffold the Go Monorepo
Refer to `docs/auth-system/implementation-guide.md` for the full project scaffold.

### 2. Start Local Infrastructure
```bash
docker compose up -d
```
This brings up: **PostgreSQL** · **Redis** · **HashiCorp Vault** · **Mailhog**

### 3. Begin Sprint 1
Implement the core auth flows in order:
1. User Registration
2. Login (JWT issuance)
3. Email Verification
4. Password Reset

---

## 📖 Documentation

| Document | Description |
|----------|-------------|
| [PRD](docs/auth-system/prd.md) | Product requirements, user stories, acceptance criteria |
| [Project Plan](docs/auth-system/project-management-plan.md) | WBS, RACI, risk register, timeline, budget |
| [Architecture](docs/auth-system/solution-architecture.md) | C4 diagrams, ADRs, data models, API contracts |
| [Implementation Guide](docs/auth-system/implementation-guide.md) | Code scaffold, patterns, Go/Next.js setup |
| [Test Plan](docs/auth-system/test-plan.md) | Test strategy, cases, quality gates, Playwright setup |

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.
