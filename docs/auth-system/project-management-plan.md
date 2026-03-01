# Project Management Plan
# Authentication System — Pilot Project

**Version:** 1.0
**Date:** 2026-02-27
**Status:** Active — In Planning
**Author:** Project Manager Agent
**Intended Consumers:** Solution Architect, Developer, Tester, Product Owner agents and all project stakeholders
**Input Document:** PRD v1.1 (Authentication System, 2026-02-27)

---

## Revision History

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.0 | 2026-02-27 | Project Manager Agent | Initial plan — full scope, 10 sprints |

---

## Table of Contents

1. Project Charter
2. Work Breakdown Structure (WBS)
3. RACI Chart
4. Detailed Sprint Plan
5. Project Timeline / Gantt-style Milestones
6. Resource Allocation Plan
7. Risk Register
8. Budget Estimate
9. Communication Plan
10. Change Control Process
11. Vendor and External Dependencies
12. Project Closure Criteria

---

---

# Section 1: Project Charter

## 1.1 Project Overview

| Field | Value |
|-------|-------|
| Project Name | Authentication System — Pilot |
| Project Code | AUTH-2026-PILOT |
| Start Date | 2026-03-02 (Week 1) |
| Target Launch Date | 2026-07-24 (end of Week 20 / Sprint 9) |
| Project Manager | Project Manager Agent |
| Product Owner | Product Owner Agent |
| Solution Architect | Solution Architect Agent |

## 1.2 Objectives

**Primary Objective:** Deliver a self-hosted, multi-tenant, standards-compliant authentication platform that eliminates third-party vendor dependency and provides a foundation for SOC2 Type II certification.

**Specific Objectives:**

1. Implement all 20 Must Have user stories (~89 story points) spanning user registration, session management, multi-tenancy, RBAC, OAuth 2.0, and security hardening.
2. Complete 9 delivery sprints plus 1 architecture spike sprint within a 20-week / 5-month window.
3. Achieve a production-ready system that passes an external penetration test with no unresolved Critical or High findings.
4. Establish a compliance posture (GDPR right-to-erasure, complete audit logging, 1-year retention) sufficient to begin SOC2 Type II evidence collection at launch.
5. Enable new internal service integration with the auth platform in 2 business days or fewer.

## 1.3 Scope Statement

### In Scope

- Backend API server (Go / Gin) implementing all 18 Must Have features (M1–M18) from the PRD
- Minimal frontend component: OAuth consent mechanism only (Next.js, 0.5 FE resource)
- PostgreSQL database with schema-per-tenant isolation (ADR-001)
- Redis-backed rate limiting and session counter infrastructure
- Dedicated secrets manager integration (ADR-009)
- CI/CD pipeline, staging environment, and production-equivalent deployment
- Automated cross-tenant isolation test suite (US-08b) running in CI on every pull request
- External penetration test (booked Sprint 0, executed Sprint 6/7, remediated Sprint 8)
- GDPR right-to-erasure endpoint (COMP-01)
- Developer integration documentation
- Incident response runbook
- Should Have features S1 (TOTP MFA), S2 (MFA enforcement), S5 (User Profile API) — Sprint 7
- Should Have feature S7 (Developer Documentation Site) — partial, Sprint 8

### Out of Scope (Final — from PRD Section 9)

- SAML 2.0 (OS-01)
- Fine-grained policy engine / OPA / Casbin (OS-02)
- Billing or subscription management (OS-03)
- User behavior analytics / risk-based auth (OS-04)
- Hosted login pages (OS-05, ADR-003)
- Multi-region active-active (OS-06)
- Native mobile SDKs (OS-07)
- LDAP / Kerberos / WS-Federation (OS-08)
- Cross-tenant user identities (OS-09, ADR-002)
- HSM integration (OS-10)
- SCIM 2.0 provisioning (OS-11)
- White-labeled / branded login pages (OS-12)

### Scope Boundary Notes

Any request to add SAML 2.0 or other out-of-scope items must be processed through the Change Control Process (Section 10). SAML is explicitly logged as a high-probability scope creep risk in the Risk Register.

## 1.4 Constraints

| ID | Constraint | Source |
|----|-----------|--------|
| C-01 | API-only interface — no hosted login UI | ADR-003 |
| C-02 | Schema-per-tenant isolation — cannot be changed to RLS or DB-per-tenant | ADR-001 |
| C-03 | JWT access tokens: RS256 or ES256, 15-minute maximum TTL | ADR-004, SEC-02, SEC-03 |
| C-04 | All secrets must reside in dedicated secrets manager — never in env vars, code, or committed config | ADR-009, SEC-07 |
| C-05 | Single tenant per user — no cross-tenant identity in this pilot | ADR-002 |
| C-06 | 0.5 DevOps resource is shared — cannot be assumed full-time | Team definition |
| C-07 | 0.5 Frontend resource — minimal OAuth consent scope only | Team definition |
| C-08 | Penetration test must occur in the Sprint 6/7 window — not after Sprint 8 begins | PRD Risk Register |
| C-09 | US-08b cross-tenant isolation tests are non-negotiable DoD for all data-access stories | PRD DoD |
| C-10 | Budget is fixed for the pilot phase; scope changes require trade-off analysis | Project governance |
| C-11 | Compliance consultant must be engaged by Sprint 2 (SOC2 posture) | ADR-008 |

## 1.5 Assumptions

| ID | Assumption |
|----|-----------|
| A-01 | Team members are available at the percentages defined in the Team section from Sprint 0 start |
| A-02 | Cloud infrastructure (PostgreSQL, Redis, compute) can be provisioned by DevOps within Sprint 0 |
| A-03 | Google Cloud project and OAuth 2.0 credentials for Google Social Login will be available before Sprint 6 |
| A-04 | An email service provider (transactional email) is either already contracted or can be procured within Sprint 0 |
| A-05 | A penetration testing vendor can be contracted in Sprint 0 for availability in the Sprint 6/7 window |
| A-06 | A compliance consultant (SOC2) can be identified and engaged by Sprint 2 |
| A-07 | Stakeholders are available for sprint reviews at end of each two-week sprint |
| A-08 | Sprint velocity will stabilize at 25–32 points/sprint after Sprint 1 warm-up |
| A-09 | The Solution Architect is heavily involved in Sprint 0 and available in an advisory capacity thereafter |
| A-10 | The Product Owner is available for backlog refinement, sprint ceremonies, and acceptance sign-off throughout |
| A-11 | No team member is on extended leave for more than one consecutive sprint without a documented contingency |
| A-12 | External pentest findings will not require architectural changes (refactoring accepted; re-architecture is a project risk) |

## 1.6 Success Criteria

The project is deemed successful when ALL of the following are true at launch and within 6 months post-launch:

| Criterion | Measurement |
|-----------|-------------|
| All 20 Must Have stories delivered, accepted, and in production | PO sign-off, sprint reviews |
| Login p95 latency < 300ms at 1,000 concurrent sessions | Performance test results (Sprint 8) |
| Authentication error rate < 0.1% (excluding user error) | Monitoring dashboards |
| Zero unresolved Critical or High findings from external pentest | Pentest remediation report |
| 100% of defined audit events logged and verifiable | QA audit log completeness test |
| GDPR right-to-erasure endpoint functional and tested | QA sign-off |
| New service integration time ≤ 2 business days | Developer documentation + integration test |
| Tenant onboarding < 30 minutes self-service | End-to-end test in Sprint 9 |
| MFA adoption ≥ 60% for admin users (measured 90 days post-launch) | Analytics |
| Developer satisfaction ≥ 4.2 / 5.0 | Developer survey at 90-day mark |
| Cross-tenant isolation test suite passes on every CI run | CI pipeline green |
| No security incidents attributable to the auth layer post-launch | Security monitoring |

## 1.7 Stakeholders

| Stakeholder | Role | Engagement Level | Primary Concerns |
|-------------|------|-----------------|-----------------|
| Business Leadership | Sponsor / decision authority | Monthly status; escalation | Cost, timeline, vendor lock-in elimination |
| Product Owner | Requirements, acceptance, backlog | Daily; sprint ceremonies | Feature completeness, user needs, priority |
| Solution Architect | Technical decisions, ADRs | Sprint 0 intensive; advisory thereafter | Architectural integrity, ADR compliance |
| Backend Engineers (3) | Implementation | Daily | Technical clarity, unblocking, DoD |
| Frontend Engineer (0.5) | OAuth consent UI | Sprint-by-sprint as needed | Scope clarity for minimal frontend |
| QA Engineer | Test planning, execution, sign-off | Daily during active sprints | Test coverage, defect management |
| DevOps Engineer (0.5) | Infrastructure, CI/CD | Sprint 0, then on-demand | Environment stability, deployment automation |
| Security / Compliance Team | Pentest oversight, SOC2 evidence | Sprint 2+ (compliance consultant engaged) | Audit trail, GDPR, pentest findings |
| Tenant Administrators (pilot users) | UAT participants | Sprint 9 UAT | Onboarding ease, audit log access |
| Developers / API Consumers (pilot) | Integration testing | Sprint 9 UAT | API documentation quality, error clarity |
| External Penetration Tester | Security validation | Sprint 6/7 (contracted Sprint 0) | Full system access, finding remediation |
| SOC2 Compliance Consultant | Compliance guidance | Engaged by Sprint 2 | Evidence collection, control mapping |

## 1.8 PM Authority Level

| Decision Type | PM Authority |
|---------------|-------------|
| Sprint scope adjustments within approved backlog | Full authority (notify PO) |
| Resource reallocation within project team | Full authority (notify all parties) |
| Vendor selection (email, secrets manager) | Recommend; Sponsor approval required |
| Scope changes (add/remove user stories) | Requires Change Control Process; PO + PM approval |
| Budget variance ≤ 10% | PM authority with immediate notification to Sponsor |
| Budget variance > 10% | Requires Sponsor approval before proceeding |
| Timeline extension ≤ 1 sprint (2 weeks) | PM authority with Sponsor notification |
| Timeline extension > 1 sprint | Requires Sponsor approval |
| Cancellation or major pivot | Sponsor decision; PM facilitates |
| Architectural decisions | Solution Architect authority; PM coordinates |

---

---

# Section 2: Work Breakdown Structure (WBS)

The WBS is organized hierarchically. Level 1 = Phases, Level 2 = Work Packages, Level 3 = Tasks/Activities. Story point references are from the PRD backlog.

---

## 1.0 Project Management

### 1.1 Initiation
- 1.1.1 Project Charter finalization
- 1.1.2 Stakeholder identification and engagement plan
- 1.1.3 Project Management Plan production (this document)
- 1.1.4 Kickoff meeting facilitation

### 1.2 Planning (ongoing)
- 1.2.1 Sprint planning facilitation (all 10 sprints)
- 1.2.2 Backlog refinement sessions (bi-weekly)
- 1.2.3 Risk register maintenance (weekly)
- 1.2.4 Resource allocation monitoring (weekly)
- 1.2.5 Budget tracking and burn rate reporting (weekly)

### 1.3 Execution Oversight
- 1.3.1 Daily standup facilitation
- 1.3.2 Sprint review facilitation (all 9 delivery sprints)
- 1.3.3 Sprint retrospective facilitation (all 10 sprints)
- 1.3.4 Blocker escalation and resolution
- 1.3.5 Vendor coordination (pentest, compliance, email provider)

### 1.4 Monitoring and Control
- 1.4.1 Weekly status report to Sponsor
- 1.4.2 Sprint velocity tracking and forecast
- 1.4.3 Change request intake and processing
- 1.4.4 Scope change log maintenance
- 1.4.5 Issue log maintenance

### 1.5 Closure
- 1.5.1 Project closure report
- 1.5.2 Lessons learned facilitation
- 1.5.3 Handoff to operations / maintenance team

---

## 2.0 Architecture and Infrastructure

### 2.1 Sprint 0 — Architecture Spike
- 2.1.1 Entity-Relationship Diagram (ERD) — global registry + tenant schema template
- 2.1.2 Schema-routing middleware design document (Go)
- 2.1.3 Migration runner strategy design + working POC
- 2.1.4 API versioning strategy definition
- 2.1.5 Error response format standard
- 2.1.6 ADR documentation (ADR-001 through ADR-009 formally written)
- 2.1.7 Go monorepo structure setup (backend)
- 2.1.8 Next.js minimal shell setup (frontend)

### 2.2 Infrastructure Setup
- 2.2.1 Cloud environment provisioning (compute, networking, DNS)
- 2.2.2 PostgreSQL provisioning (staging + production-equivalent)
- 2.2.3 Redis provisioning (staging + production-equivalent)
- 2.2.4 Secrets manager setup and initial key seeding (ADR-009)
- 2.2.5 TLS certificate provisioning and HSTS configuration
- 2.2.6 Monitoring and alerting infrastructure (AVAIL-04 thresholds)
- 2.2.7 Log aggregation and retention setup (ADR-007)
- 2.2.8 Cold storage (S3/equivalent) for audit log archiving

### 2.3 CI/CD Pipeline
- 2.3.1 Code repository setup and branch strategy
- 2.3.2 CI pipeline: lint, unit tests, integration tests
- 2.3.3 CI pipeline: cross-tenant isolation test suite (US-08b) on every PR
- 2.3.4 CD pipeline: automated staging deployment
- 2.3.5 CD pipeline: production deployment with manual gate
- 2.3.6 Secrets injection pipeline (secrets manager → runtime)

---

## 3.0 Backend Development — Core Auth

### 3.1 User Registration and Email Verification (Sprint 1)
- 3.1.1 US-01: User registration endpoint (POST /auth/register) — 5 pts
- 3.1.2 US-02: Email verification token generation, send, verify, resend — 5 pts
- 3.1.3 Email service integration (async transactional email) — 3 pts
- 3.1.4 Password hashing with Argon2id (SEC-01)
- 3.1.5 Per-tenant password complexity config (global default for Sprint 1)

### 3.2 Authentication (Sprint 1–2)
- 3.2.1 US-03: Login endpoint — JWT (RS256) + opaque refresh token — 5 pts
- 3.2.2 US-06: Password reset flow (forgot + reset endpoints) — 3 pts
- 3.2.3 US-04: Session refresh with refresh token rotation — 3 pts
- 3.2.4 US-05: Logout (single-device + all-devices) — 5 pts

### 3.3 Security Baseline (Sprint 2)
- 3.3.1 US-14: Rate limiting + account lockout (Redis-backed) — 5 pts
- 3.3.2 M15: HTTPS enforcement + secure response headers (SEC-05, SEC-06) — 2 pts

### 3.4 Audit Log (Sprint 2)
- 3.4.1 US-15: Audit log append-only storage — 5 pts
- 3.4.2 Audit log read API (tenant-admin scoped)
- 3.4.3 All 15 defined event types wired to relevant endpoints

---

## 4.0 Backend Development — Multi-Tenancy

### 4.1 Tenant Provisioning (Sprint 3)
- 4.1.1 US-07a: Tenant provisioning API + PostgreSQL schema creation — 5 pts
- 4.1.2 US-07b: Per-tenant migration runner — 5 pts
- 4.1.3 US-07c: Tenant API credentials (client_id / client_secret) — 3 pts
- 4.1.4 Default role seeding (admin, user) on provisioning

### 4.2 Tenant Isolation (Sprint 3)
- 4.2.1 US-08a: Schema-routing middleware (search_path per request) — 5 pts
- 4.2.2 US-08b: Cross-tenant isolation automated test suite — 5 pts
- 4.2.3 CI integration of isolation suite (blocks merge on failure)

---

## 5.0 Backend Development — RBAC and Admin

### 5.1 RBAC (Sprint 4)
- 5.1.1 US-09: Role assignment / unassignment endpoint — 3 pts
- 5.1.2 US-10: Role + tenant claims in JWT — 2 pts

### 5.2 Admin API (Sprint 4)
- 5.2.1 M16: Admin user management API (invite, disable, delete) — 5 pts
- 5.2.2 M17: Token introspection endpoint (RFC 7662) — 3 pts
- 5.2.3 M18: JWKS endpoint (public key publication) — 2 pts

---

## 6.0 Backend Development — OAuth 2.0

### 6.1 OAuth Authorization Server (Sprint 5)
- 6.1.1 US-11a: OAuth client registration API — 3 pts
- 6.1.2 US-11b: /oauth/authorize endpoint (API-only, JSON response) — 3 pts
- 6.1.3 US-11c: /oauth/token — code exchange + PKCE S256 validation — 5 pts
- 6.1.4 OAuth library selection and integration (battle-tested, no from-scratch implementation)

### 6.2 OAuth M2M and Social Login (Sprint 6)
- 6.2.1 US-12: Client credentials grant (M2M) — 5 pts
- 6.2.2 US-13: Google social login (OIDC discovery, account linking) — 8 pts
- 6.2.3 Google Cloud project setup and credential configuration

---

## 7.0 Backend Development — Should Haves (Sprint 7)

### 7.1 MFA
- 7.1.1 S1: TOTP MFA setup (QR code provisioning, secret storage) — 8 pts
- 7.1.2 S1: TOTP MFA verification on login
- 7.1.3 S2: Per-tenant MFA enforcement (admin toggle) — 3 pts

### 7.2 User Profile
- 7.2.1 S5: Self-service user profile API (update name, email, password, MFA) — 5 pts

---

## 8.0 Frontend Development (Minimal)

### 8.1 OAuth Consent Mechanism (Sprint 5)
- 8.1.1 OAuth consent response handling (API-driven, JSON)
- 8.1.2 No hosted login page — API-only per ADR-003

---

## 9.0 Quality Assurance

### 9.1 Test Planning
- 9.1.1 Test strategy document
- 9.1.2 Test environment setup (QA tenant, test data)
- 9.1.3 Test cases authored per user story (Gherkin acceptance criteria from PRD)

### 9.2 Test Execution (Per Sprint)
- 9.2.1 Sprint 1: Core auth flows (register, verify, login, reset)
- 9.2.2 Sprint 2: Session, security, rate limiting, audit log
- 9.2.3 Sprint 3: Multi-tenancy, isolation, migration runner
- 9.2.4 Sprint 4: RBAC, admin API, token introspection, JWKS
- 9.2.5 Sprint 5: OAuth authorization code + PKCE end-to-end
- 9.2.6 Sprint 6: M2M tokens, Google social login, account linking
- 9.2.7 Sprint 7: TOTP MFA, MFA enforcement, user profile
- 9.2.8 Sprint 8: Security hardening review, performance tests, GDPR endpoint
- 9.2.9 Sprint 9: End-to-end regression, UAT support

### 9.3 Specialized Testing
- 9.3.1 Cross-tenant isolation test suite (automated, CI — authored Sprint 3)
- 9.3.2 Performance testing: PERF-01 through PERF-05 (Sprint 8)
- 9.3.3 OWASP Top 10 review (Sprint 8)
- 9.3.4 Pentest coordination and findings intake (Sprint 6/7 window)
- 9.3.5 Pentest remediation verification (Sprint 8)
- 9.3.6 Regression test suite maintenance (ongoing)

### 9.4 UAT (Sprint 9)
- 9.4.1 UAT participant identification and onboarding
- 9.4.2 UAT scenarios (tenant admin, developer / API consumer)
- 9.4.3 UAT issue triage and resolution
- 9.4.4 UAT sign-off

---

## 10.0 Security and Compliance

### 10.1 Security Controls
- 10.1.1 Secrets manager provisioning and key rotation policy
- 10.1.2 Secure headers implementation (Sprint 2)
- 10.1.3 PKCE S256 enforcement (Sprint 5)
- 10.1.4 Argon2id password hashing baseline (Sprint 1)
- 10.1.5 Refresh token family revocation logic (Sprint 2)

### 10.2 Penetration Testing
- 10.2.1 Vendor procurement and SOW definition (Sprint 0)
- 10.2.2 Pentest execution window (Sprint 6 end / Sprint 7)
- 10.2.3 Finding triage and severity classification
- 10.2.4 Critical/High finding remediation (Sprint 8)
- 10.2.5 Pentest remediation verification and sign-off

### 10.3 Compliance
- 10.3.1 GDPR right-to-erasure API endpoint (Sprint 8, COMP-01)
- 10.3.2 SOC2 Type II evidence collection framework (started Sprint 2, post-launch ongoing)
- 10.3.3 Compliance consultant engagement (Sprint 2)
- 10.3.4 Audit log completeness verification (100% of events)
- 10.3.5 Audit log archiving pipeline (cold storage)

---

## 11.0 Documentation

### 11.1 Technical Documentation
- 11.1.1 ADR-001 through ADR-009 formal write-up (Sprint 0)
- 11.1.2 ERD and schema documentation (Sprint 0)
- 11.1.3 API reference documentation (inline, updated each sprint)
- 11.1.4 Developer integration guide (Sprint 8)
- 11.1.5 Secrets manager configuration guide (Sprint 0)
- 11.1.6 Migration runner developer CLI guide (Sprint 3)

### 11.2 Operational Documentation
- 11.2.1 Incident response runbook (Sprint 9)
- 11.2.2 Monitoring and alerting guide (Sprint 8)
- 11.2.3 Tenant onboarding operator guide (Sprint 3/4)
- 11.2.4 Go/no-go criteria checklist (Sprint 9)

---

## 12.0 Launch

### 12.1 Pre-Launch Validation (Sprint 9)
- 12.1.1 Staging-to-production-equivalent environment validation
- 12.1.2 Smoke test suite on production environment
- 12.1.3 Go/no-go criteria review with all stakeholders
- 12.1.4 Rollback plan documented and tested

### 12.2 Launch Execution
- 12.2.1 Production deployment (gated on go/no-go approval)
- 12.2.2 Launch communication to stakeholders
- 12.2.3 Monitoring dashboard activated and watched
- 12.2.4 Hypercare period (first 2 weeks post-launch, team on standby)

---

---

# Section 3: RACI Chart

**Roles:**
- **PM** = Project Manager
- **PO** = Product Owner
- **SA** = Solution Architect
- **BE** = Backend Engineers (collective; lead noted where relevant)
- **FE** = Frontend Engineer
- **QA** = QA Engineer
- **DO** = DevOps Engineer
- **STK** = Business Stakeholders / Sponsor

**RACI Key:** R = Responsible (does the work), A = Accountable (owns the outcome), C = Consulted (input sought), I = Informed (kept updated)

---

| Work Category | PM | PO | SA | BE | FE | QA | DO | STK |
|--------------|----|----|----|----|----|----|-----|-----|
| **Project Charter and Plan** | R/A | C | C | I | I | I | I | C |
| **Stakeholder Management** | R/A | C | I | I | I | I | I | I |
| **Sprint Planning Facilitation** | R/A | C | C | C | C | C | C | I |
| **Sprint Review Facilitation** | R/A | C | C | C | C | C | C | C |
| **Sprint Retrospective Facilitation** | R/A | C | C | C | C | C | C | I |
| **Daily Standup** | R | I | I | R | R | R | R | — |
| **Backlog Refinement** | C | R/A | C | C | C | C | — | I |
| **Acceptance Criteria Sign-off** | I | R/A | C | C | — | C | — | C |
| **Budget Tracking** | R/A | I | I | — | — | — | I | I |
| **Scope Change Decisions** | A | R | C | C | — | C | — | C |
| **Risk Management** | R/A | C | C | C | C | C | C | I |
| **Architecture Decisions (ADRs)** | I | C | R/A | C | — | I | C | I |
| **ERD and Schema Design** | I | I | R/A | C | — | I | — | — |
| **Migration Runner Design** | I | I | R/A | R | — | I | — | — |
| **API Versioning and Error Format** | I | C | R/A | C | — | C | — | — |
| **Go Monorepo Structure** | I | I | C | R/A | — | — | C | — |
| **Infrastructure Provisioning** | I | — | C | I | — | I | R/A | — |
| **CI/CD Pipeline Setup** | I | — | C | C | — | I | R/A | — |
| **Secrets Manager Setup** | I | — | A | C | — | — | R | — |
| **TLS and Secure Headers** | I | I | C | R/A | — | C | C | — |
| **US-01: User Registration** | I | C | C | R/A | — | C | — | — |
| **US-02: Email Verification** | I | C | C | R/A | — | C | — | — |
| **US-03: User Login** | I | C | C | R/A | — | C | — | — |
| **US-04: Session Refresh** | I | C | C | R/A | — | C | — | — |
| **US-05: Logout** | I | C | C | R/A | — | C | — | — |
| **US-06: Password Reset** | I | C | C | R/A | — | C | — | — |
| **US-07a: Tenant Provisioning** | I | C | C | R/A | — | C | — | — |
| **US-07b: Migration Runner** | I | I | C | R/A | — | C | — | — |
| **US-07c: Tenant API Credentials** | I | C | C | R/A | — | C | — | — |
| **US-08a: Schema-Routing Middleware** | I | I | A | R | — | C | — | — |
| **US-08b: Isolation Test Suite** | I | I | C | R | — | R/A | — | — |
| **US-09: Role Assignment** | I | C | C | R/A | — | C | — | — |
| **US-10: JWT Claims** | I | C | C | R/A | — | C | — | — |
| **M16: Admin User Management API** | I | C | C | R/A | — | C | — | — |
| **M17: Token Introspection** | I | C | C | R/A | — | C | — | — |
| **M18: JWKS Endpoint** | I | C | C | R/A | — | C | — | — |
| **US-11a: OAuth Client Registration** | I | C | C | R/A | — | C | — | — |
| **US-11b: /oauth/authorize** | I | C | A | R | — | C | — | — |
| **US-11c: /oauth/token + PKCE** | I | C | A | R | — | C | — | — |
| **OAuth Consent Mechanism (minimal FE)** | I | C | C | C | R/A | C | — | — |
| **US-12: Client Credentials M2M** | I | C | C | R/A | — | C | — | — |
| **US-13: Google Social Login** | I | C | C | R/A | — | C | — | — |
| **S1: TOTP MFA** | I | C | C | R/A | — | C | — | — |
| **S2: MFA Enforcement** | I | C | C | R/A | — | C | — | — |
| **S5: User Profile API** | I | C | C | R/A | — | C | — | — |
| **Performance Testing** | I | I | C | C | — | R/A | C | — |
| **OWASP Top 10 Review** | I | I | C | C | — | R/A | — | — |
| **Penetration Test Vendor Selection** | R/A | I | C | — | — | C | — | C |
| **Penetration Test Execution** | A | I | C | C | — | R | — | I |
| **Pentest Finding Remediation** | A | I | C | R | — | C | — | — |
| **GDPR Right-to-Erasure Endpoint** | I | A | C | R | — | C | — | — |
| **SOC2 Evidence Collection** | C | A | C | C | — | C | C | R |
| **Compliance Consultant Engagement** | R/A | C | I | — | — | I | — | C |
| **Developer Integration Docs** | I | C | C | R/A | — | C | — | — |
| **Incident Response Runbook** | R/A | C | C | C | — | C | C | I |
| **UAT Planning** | R/A | C | C | C | — | R | — | C |
| **UAT Execution** | A | R | C | C | — | R | — | R |
| **UAT Sign-off** | A | R | I | I | — | C | — | C |
| **Go/No-Go Decision** | R | C | C | C | — | C | C | A |
| **Production Deployment** | A | I | C | C | — | C | R | I |
| **Launch Communication** | R/A | C | I | I | I | I | I | C |
| **Post-Launch Monitoring** | A | I | C | C | — | C | R | I |
| **Project Closure** | R/A | C | C | C | C | C | C | C |

---

---

# Section 4: Detailed Sprint Plan

**Sprint duration:** 2 weeks
**Team velocity:** 25–32 points/sprint (stabilizes Sprint 2 onward)
**All ceremonies are time-boxed as shown; adjust proportionally if team is remote**

---

## Sprint 0 — Architecture Spike
**Dates:** Week 1–2 (2026-03-02 to 2026-03-13)
**Story Points:** 0 (no production code committed)

### Sprint Goal
Establish the technical foundation — a working migration runner POC, complete infrastructure, all ADRs documented, and CI/CD green — so that Sprint 1 can begin feature delivery with zero ambiguity.

### Tasks and Assignees

| Task | Assignee | Priority |
|------|----------|----------|
| Design global registry schema + tenant schema template (ERD) | SA | P0 — blocking Sprint 3 |
| Design schema-routing middleware approach (Go, search_path) | BE Lead | P0 — blocking Sprint 3 |
| Migration runner strategy: design + working POC | BE Lead | P0 — blocking Sprint 3 |
| Provision PostgreSQL + Redis (staging) | DevOps | P0 — blocking Sprint 1 |
| Provision compute (staging) and networking | DevOps | P0 — blocking Sprint 1 |
| Set up secrets manager (ADR-009) — initial keys seeded | DevOps + SA | P0 — blocking Sprint 1 |
| Set up CI/CD pipeline — lint, test, deploy-to-staging | DevOps | P0 — blocking Sprint 1 |
| Set up branch strategy and PR template | DevOps + BE Lead | P0 |
| Go monorepo structure (backend service skeleton, no logic) | BE Lead | P0 — blocking Sprint 1 |
| Next.js minimal shell (no UI yet) | FE | P1 |
| Agree on API versioning strategy (e.g., /api/v1 prefix) | SA + BE Lead | P0 — blocking Sprint 1 |
| Define standard error response JSON format | SA + BE Lead | P0 — blocking Sprint 1 |
| Formally document ADR-001 through ADR-009 | SA | P0 |
| Identify and book external penetration test vendor (Sprint 6/7 window) | PM | P0 — long lead time |
| Identify SOC2 compliance consultant (engage by Sprint 2) | PM | P1 |
| Email service provider selection and API key setup | PM + DevOps | P1 — blocking Sprint 1 |
| Google Cloud project setup and OAuth credentials (needed Sprint 6) | PM + DevOps | P2 — needed Sprint 6 |
| Set up Jira (or equivalent) project tracking | PM | P0 |

### Dependencies (what must exist before this sprint)
- None — this is the first sprint. Team must be on-boarded before 2026-03-02.

### Definition of Ready (Entry Criteria)
- All team members have accounts in code repository, CI/CD system, and cloud environment
- Infrastructure vendor/cloud account is contracted and accessible
- PRD v1.1 is approved and distributed to all team members

### Definition of Done (Sprint 0 Exit Criteria)
- ERD formally documented and reviewed by SA + BE Lead
- Migration runner POC: creates a tenant schema, applies a migration, records in schema_migrations — demonstrated running
- CI pipeline: lint + unit test skeleton running green on main branch
- CD pipeline: deploys a skeleton Go service to staging on merge to main
- PostgreSQL and Redis accessible from staging compute
- Secrets manager running; one test secret retrievable by application
- All ADR-001 through ADR-009 written and committed to docs directory
- API versioning and error format agreed and documented
- Penetration tester contracted (or vendor shortlist with decision by Sprint 1 Day 1)
- Email provider contracted and test email sendable from CI
- Jira board set up with Sprint 1 backlog loaded

### Key Risks
- Infrastructure provisioning delays (cloud account setup, DNS, TLS) — mitigation: PM tracks daily; DevOps escalates blockers on Day 1
- Migration runner POC takes longer than estimated — mitigation: BE Lead time-boxes to 3 days; SA unblocks decisions immediately
- Penetration tester cannot be contracted in time — mitigation: start vendor outreach on Day 1 of Sprint 0; have two vendors in parallel
- Team ramp-up time if any member is unfamiliar with Go or the stack — mitigation: SA provides onboarding session Day 1

### Ceremonies
| Ceremony | When | Duration | Participants |
|----------|------|----------|-------------|
| Sprint 0 Kickoff | Day 1 AM | 2 hours | All |
| Daily Standup | Daily 9:00 AM | 15 min | All |
| Mid-sprint architecture review | Day 5 | 1 hour | SA, BE Lead, PM |
| Sprint 0 Review | Day 10 | 1 hour | All |
| Sprint 0 Retro | Day 10 | 45 min | All |

---

## Sprint 1 — Core Auth Foundation
**Dates:** Week 3–4 (2026-03-16 to 2026-03-27)
**Story Points:** 21

### Sprint Goal
Users can register with a password, verify their email, log in to receive a JWT and refresh token, and reset a forgotten password — all against a single-tenant baseline.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| US-01: User Registration (POST /auth/register) | 5 | BE-1 | Argon2id hash; 201 + unverified status; trigger email |
| US-02: Email Verification + Resend | 5 | BE-2 | Single-use token; 24h TTL; resend invalidates previous |
| US-03: User Login (POST /auth/login) | 5 | BE-1 | JWT RS256 15min + opaque refresh; audit log LOGIN_SUCCESS/FAILURE |
| US-06: Password Reset Flow | 3 | BE-3 | Anti-enumeration timing; single-use 1h token; revoke sessions on reset |
| Email service integration (async, transactional) | 3 | BE-2 + DevOps | Required for US-02; use provider from Sprint 0 |
| Unit tests for all above | included | All BE | ≥ 80% coverage on new code |
| QA test execution: Sprint 1 stories | — | QA | Gherkin scenarios from PRD Sec 7 |

### Dependencies
- Sprint 0 complete: CI/CD green, PostgreSQL staging available, secrets manager operational, email provider contracted
- Argon2id library selected and approved
- JWT signing key generated and stored in secrets manager (not hardcoded)
- Single-tenant schema exists in staging (migration runner POC output)
- Error response format finalized (Sprint 0)

### Definition of Ready (Entry Criteria)
- US-01, US-02, US-03, US-06 have full acceptance criteria in Jira (from PRD Section 7)
- Email service credentials loaded into secrets manager
- Database schema for users, sessions, verification_tokens, password_reset_tokens migrated to staging
- Sprint 0 DoD confirmed by PM

### Key Risks
- Email delivery reliability in staging (transactional email service setup incomplete) — mitigation: test email send on Day 1
- Argon2id hashing parameters need tuning (latency vs security trade-off for PERF-01) — mitigation: benchmark on Day 1; target 200–250ms hash time to stay under p95 300ms login
- Anti-enumeration timing on password reset (US-06 SEC requirement) — mitigation: SA reviews implementation before QA

### Ceremonies
| Ceremony | When | Duration | Participants |
|----------|------|----------|-------------|
| Sprint Planning | Day 1 AM | 2 hours | All |
| Daily Standup | Daily 9:00 AM | 15 min | All |
| Backlog Refinement (Sprint 2 preview) | Day 7 | 1 hour | PM, PO, BE Lead, QA |
| Sprint Review | Day 10 PM | 1 hour | All + Stakeholders |
| Sprint Retro | Day 10 PM | 45 min | All |

---

## Sprint 2 — Sessions, Security, and Audit
**Dates:** Week 5–6 (2026-03-30 to 2026-04-10)
**Story Points:** 20

### Sprint Goal
Sessions are fully secured with rotation and revocation, brute-force protection is active, secure headers are enforced, and every auth event is captured in the audit log.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| US-04: Session Refresh + Rotation (POST /auth/token/refresh) | 3 | BE-1 | Family revocation on reuse; SUSPICIOUS_TOKEN_REUSE audit event |
| US-05: Logout (single + all devices) | 5 | BE-2 | Opaque refresh token denylist; LOGOUT/LOGOUT_ALL audit events |
| US-14: Rate Limiting + Account Lockout (Redis) | 5 | BE-3 | Per-IP + per-user counters; configurable thresholds; ACCOUNT_LOCKED audit event |
| US-15: Audit Log (all 15 event types wired) | 5 | BE-1 | Append-only; tenant-admin read API; wire to all Sprint 1 endpoints retroactively |
| M15: HTTPS enforcement + secure headers | 2 | BE-2 | HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy |
| Unit + integration tests for all above | included | All BE | Cross-tenant isolation suite not yet running — added Sprint 3 |
| QA test execution: Sprint 2 stories | — | QA | Include negative test: rate limit bypass attempts |
| Engage SOC2 compliance consultant | — | PM | ADR-008 — must be engaged this sprint |

### Dependencies
- Sprint 1 complete and deployed to staging: US-01, US-02, US-03, US-06
- Redis available and accessible from application in staging
- Default lockout thresholds agreed with PO before sprint starts (threshold: N attempts, lockout duration — get PO sign-off in Sprint 1 Review or Refinement)
- Audit log DB schema migrated to staging

### Definition of Ready (Entry Criteria)
- Redis confirmed operational in staging
- Default rate limit and lockout thresholds are documented and PO-approved
- All 15 audit event types formally listed and accepted in Jira tickets (from PRD US-15 definition)
- Sprint 1 retrospective complete; carryover items assessed

### Key Risks
- Redis connectivity or performance issues in staging — mitigation: DevOps validates Redis ping from app tier on Day 1
- Audit log retroactive wiring to Sprint 1 endpoints takes longer than estimated — mitigation: allocate dedicated 0.5 day per BE for wiring; BE Lead coordinates
- Compliance consultant not yet identified — mitigation: PM escalates to Sponsor if no candidates identified by Sprint 1 end

### Ceremonies
Same cadence as Sprint 1. Add: compliance consultant intro meeting if engaged.

---

## Sprint 3 — Multi-Tenancy Foundation
**Dates:** Week 7–8 (2026-04-13 to 2026-04-24)
**Story Points:** 23
**THIS IS THE HIGHEST-RISK SPRINT IN THE PROJECT.**

### Sprint Goal
Tenant isolation is architecturally implemented and automatically tested — every tenant's data is provably isolated from every other tenant, verified by a CI test suite that blocks merges on failure.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| US-07a: Tenant Provisioning API + Schema Creation | 5 | BE-1 | POST /admin/tenants; creates schema; default roles; invitation email; 409 on duplicate |
| US-07b: Per-Tenant Migration Runner | 5 | BE-2 | Runs migrations against all tenant schemas on deploy; idempotent; transactional; schema_migrations table |
| US-07c: Tenant API Credentials | 3 | BE-3 | client_id + client_secret (show once, store hashed); rotation endpoint |
| US-08a: Schema-Routing Middleware | 5 | BE-1 | Extracts tenant_id from JWT; sets search_path; rejects unresolvable tenant |
| US-08b: Cross-Tenant Isolation Test Suite | 5 | QA + BE-2 | Automated; covers users/roles/audit/OAuth clients/sessions; CI gate — blocks merge |
| Migration runner developer CLI documentation | — | BE-2 | Developer guide: "run migrations for tenant X" |
| Integration test: full end-to-end (register → login → provision tenant → login to tenant) | — | QA | End-to-end smoke across Sprints 1–3 |
| QA test execution: Sprint 3 stories | — | QA | US-07a acceptance criteria from PRD |

### Dependencies
- Sprint 0 migration runner POC must be complete and reviewed — if incomplete, this sprint WILL slip
- Sprint 2 complete: sessions, security, audit fully operational
- SA-designed ERD available (tenant schema template, global registry schema)
- Schema-routing middleware design document from Sprint 0

### Definition of Ready (Entry Criteria)
- Migration runner POC from Sprint 0 reviewed by SA and BE Lead: confirmed viable approach
- Global registry schema and tenant schema template finalized by SA
- US-07a, US-07b, US-07c, US-08a, US-08b have acceptance criteria in Jira
- Super-admin user role defined and seeded in global schema
- QA has begun drafting isolation test scenarios (prep during Sprint 2)

### Key Risks
- **HIGHEST RISK:** Migration runner complexity exceeds Sprint 0 POC scope (e.g., concurrent migration of 100+ schemas, rollback handling) — mitigation: Sprint 0 POC must prove the critical path; BE Lead owns escalation Day 1 if blockers surface
- Schema-routing middleware has edge cases at unauthenticated endpoints — mitigation: SA reviews middleware design before implementation begins
- US-08b (isolation test suite) may be underestimated — mitigation: QA starts drafting scenarios in Sprint 2; do NOT defer — this is non-negotiable DoD

**PM Note:** If this sprint cannot complete US-08b, do NOT proceed to Sprint 4. US-08b is the safety net for all subsequent data-access work. Escalate to Sponsor immediately if this is at risk.

### Ceremonies
Same cadence. Add: mid-sprint architecture checkpoint Day 5 (SA reviews migration runner and middleware implementation before merge).

---

## Sprint 4 — RBAC, Claims, and Admin API
**Dates:** Week 9–10 (2026-04-27 to 2026-05-08)
**Story Points:** 15
**Intentionally lighter sprint — buffer for Sprint 3 overflow + integration testing**

### Sprint Goal
Tenant admins can assign roles to users, JWT tokens carry correct tenant and role claims, admins can manage users via API, and resource servers can validate tokens via introspection and JWKS.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| US-09: Assign / Unassign Roles | 3 | BE-3 | Role audit events; tenant-scoped |
| US-10: Role + Tenant Claims in JWT | 2 | BE-1 | sub, iss, aud, exp, iat, tenant_id, roles[]; flag > 20 roles |
| M16: Admin User Management API (invite, disable, delete) | 5 | BE-2 | USER_INVITED, USER_DISABLED audit events |
| M17: Token Introspection Endpoint (RFC 7662) | 3 | BE-1 | PERF-03: p95 < 50ms |
| M18: JWKS Endpoint | 2 | BE-3 | Public key publication; supports RS256/ES256 key rotation |
| End-to-end integration: auth → multi-tenant → RBAC full flow | — | QA | Covers Sprints 1–4 |
| QA test execution: Sprint 4 stories | — | QA | Token claims test: verify JWT payload matches expected claims |

### Dependencies
- Sprint 3 complete: tenant provisioning, schema-routing, isolation test suite operational
- US-10 depends on US-09 (roles must exist to be embedded in JWT)
- JWKS endpoint requires signing key material available in secrets manager (Sprint 0)
- Per-tenant password complexity config should be added now (Sprint 1 used global default)

### Definition of Ready (Entry Criteria)
- Sprint 3 DoD confirmed; isolation test suite running in CI
- RFC 7662 (Token Introspection) reviewed by BE Lead and SA
- JWKS format confirmed (JSON Web Key Set, RFC 7517)

### Key Risks
- Sprint 3 overflow carrying into Sprint 4 (likely given 23-point Sprint 3) — mitigation: 15-point sprint provides explicit buffer; PM monitors velocity
- JWT size growth with many roles (US-10 flag > 20 roles) — mitigation: implement flag as structured log + alert; do not silently truncate

### Ceremonies
Same cadence. No additional meetings.

---

## Sprint 5 — OAuth 2.0 Authorization Server
**Dates:** Week 11–12 (2026-05-11 to 2026-05-22)
**Story Points:** 11

### Sprint Goal
Applications can register as OAuth clients and complete the Authorization Code flow with mandatory PKCE S256, receiving valid JWT access tokens.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| OAuth 2.0 library evaluation + integration (battle-tested, no from-scratch) | — | SA + BE Lead | Decision: library selected, integrated, and tested; timebox 2 days |
| US-11a: OAuth Client Registration API | 3 | BE-2 | client_id, client_secret (hashed), redirect_uris[], scopes[]; strict URI matching |
| US-11b: /oauth/authorize endpoint | 3 | BE-1 | Validates client_id, redirect_uri, state, code_challenge; returns JSON (no UI); code 10min TTL |
| US-11c: /oauth/token — Code Exchange + PKCE | 5 | BE-1 | S256 mandatory; single-use code; code reuse = revoke issued tokens; plain method = 400 |
| OAuth consent mechanism (minimal FE response) | — | FE | API response format; no HTML page per ADR-003 |
| QA test execution: Sprint 5 stories (incl. PKCE edge cases) | — | QA | Full Gherkin suite from PRD US-11c; PKCE negative tests mandatory |
| Sprint 5 end: integration test full OAuth Authorization Code flow | — | QA | End-to-end: client registration → authorize → token exchange → introspect |

### Dependencies
- Sprint 4 complete: RBAC and JWT claims in place
- OAuth 2.0 library decided before Sprint 5 Day 1 (can pre-evaluate during Sprint 4)
- FE available for minimal consent mechanism work

### Definition of Ready (Entry Criteria)
- OAuth library shortlist ready for Day 1 decision
- US-11a, US-11b, US-11c acceptance criteria in Jira
- Plain PKCE method rejection behavior agreed with SA (return 400, not silently downgrade)
- State parameter mandatory behavior confirmed

### Key Risks
- **HIGH:** OAuth 2.0 edge cases are a consistent source of overrun — mitigation: battle-tested library is non-negotiable; SA must approve library choice; strict timebox
- Code reuse detection and family revocation logic underestimated — mitigation: design this explicitly before coding; QA tests reuse scenario Day 1 of test phase
- PKCE S256 validation correctness — mitigation: QA runs negative PKCE tests before sprint review

### Ceremonies
Same cadence. Add: OAuth library decision meeting on Day 1 (SA, BE Lead, PM, 30 min).

---

## Sprint 6 — OAuth M2M and Google Social Login
**Dates:** Week 13–14 (2026-05-25 to 2026-06-05)
**Story Points:** 13

### Sprint Goal
Backend services can obtain tokens without user context via Client Credentials, and end users can log in with their Google account with proper account linking.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| US-12: Client Credentials Grant (M2M) | 5 | BE-3 | No sub/user roles; scope-bound; client_id + client_secret (hashed) |
| US-13: Social Login via Google (OIDC) | 8 | BE-1 + BE-2 | OIDC discovery; state mandatory; verify-password-before-link (ADR-006); new user auto-verified |
| QA test execution: Sprint 6 stories | — | QA | Account linking test: Google email matches existing password account |
| Pentest window: external penetration tester begins assessment | — | PM + QA + BE | Tester given staging access; PM coordinates; team available for questions |

### Dependencies
- Sprint 5 complete: OAuth Authorization Server operational
- **Google Cloud project configured with OAuth 2.0 credentials — MUST be ready before Sprint 6 Day 1** (provisioned in Sprint 0, verified in Sprint 5)
- Penetration testing vendor contracted (Sprint 0); access credentials and scope agreed (Sprint 5)
- Staging environment is complete (all Sprint 1–5 features deployed and stable)

### Definition of Ready (Entry Criteria)
- Google Cloud OAuth 2.0 client ID and client secret loaded into secrets manager
- OIDC discovery URL for Google confirmed (https://accounts.google.com/.well-known/openid-configuration)
- Pentest scope of work agreed and signed; tester has staging environment access
- Per-tenant Google client ID/secret configuration mechanism defined by SA

### Key Risks
- Google OAuth API changes / Google credential setup delays — mitigation: verify credentials in Sprint 5 refinement; do not start US-13 without working credentials
- Account linking logic (ADR-006) is complex — password verification before social account link is a security-critical flow; mitigation: SA reviews design; QA tests linking and non-linking paths
- Pentest findings may arrive during Sprint 7 and cause Sprint 8 scope to expand — mitigation: PM tracks finding ETA; preliminaries fed to team as they arrive
- Memorial Day (2026-05-25) — US public holiday on Sprint 6 Day 1: adjust sprint planning to Day 2 if US team

### Ceremonies
Same cadence. Add: pentest kickoff call with vendor (Day 1 or 2).

---

## Sprint 7 — Should Haves: MFA and User Profile
**Dates:** Week 15–16 (2026-06-08 to 2026-06-19)
**Story Points:** 16

### Sprint Goal
Users can enroll in TOTP MFA and use it for login, tenant admins can enforce MFA for their organization, and users can manage their own profile.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| S1: TOTP MFA — Setup (enroll, QR code, secret) | 4 | BE-2 | TOTP RFC 6238; backup codes generated at enrollment |
| S1: TOTP MFA — Verification on Login | 4 | BE-1 | Step-up after password verification; 30-second window; clock drift tolerance ±1 window |
| S2: Per-Tenant MFA Enforcement | 3 | BE-3 | Admin toggle; users without MFA enrolled are redirected to enroll; audit log event |
| S5: Self-Service User Profile API | 5 | BE-2 | Update name, email (re-verify), password (verify current), MFA enrollment/unenrollment |
| QA test execution: Sprint 7 stories | — | QA | TOTP: test valid code, expired code, reused code, wrong code |
| Pentest findings intake: preliminary review | — | QA + BE Lead | Findings arriving from Sprint 6; triage severity; feed High/Critical to Sprint 8 |

### Dependencies
- Sprint 6 complete: OAuth M2M and social login operational
- TOTP library selected and approved (recommend standard RFC 6238 implementation)
- Pentest vendor has had access since Sprint 6 start; preliminary findings should be available by Day 5

### Definition of Ready (Entry Criteria)
- S1, S2, S5 acceptance criteria in Jira
- TOTP library decision made (pre-Sprint 7 refinement)
- Clock drift tolerance policy agreed with SA (±1 window = ±30 seconds)
- Pentest findings triage process agreed: PM receives report; SA + QA review; PM updates risk register

### Key Risks
- MFA backup codes add scope creep to S1 — mitigation: backup codes are included in 8-point estimate; do not reduce to single-code only (security risk)
- TOTP implementation clock drift issues in CI (CI server time vs user device time) — mitigation: use mocked time in TOTP unit tests
- Pentest findings may require Sprint 7 engineers to pivot to remediation — mitigation: if Critical findings arrive, PM triggers change control to re-prioritize; MFA deferred before pentest remediation deferred

### Ceremonies
Same cadence. Add: pentest findings review meeting Day 7 (SA, QA, BE Lead, PM — 1 hour).

---

## Sprint 8 — Hardening and Compliance
**Dates:** Week 17–18 (2026-06-22 to 2026-07-03)
**Story Points:** 25
**Note: This sprint has the highest point total. The 8 points for pentest remediation is an estimate — actual may vary.**

### Sprint Goal
The system is production-hardened: all Critical and High pentest findings are remediated, GDPR right-to-erasure is implemented, performance meets SLA targets, monitoring is operational, and developer documentation is ready.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| GDPR Right-to-Erasure API endpoint | 3 | BE-3 | DELETE /users/me or /admin/users/{id}; deletes PII across tenant schema; anonymizes audit log entries (replace user ID with tombstone) |
| Pentest finding remediation (Critical + High) | 8 | All BE | 8 pts is an estimate — actual scope depends on findings; PM monitors weekly from Sprint 6 |
| OWASP Top 10 review + gap remediation | 3 | QA + BE Lead | Structured walkthrough; document findings; fix gaps |
| Performance testing: PERF-01 through PERF-05 | 5 | QA + BE | Login p95 < 300ms at 1000 concurrent; token issuance p95 < 100ms; introspection p95 < 50ms; rate limit check p95 < 5ms |
| Performance remediation (indexes, query optimization) | included in 5 | BE | PERF-04: ensure indexes on email, tenant_id, refresh_token_hash |
| Monitoring, alerting, dashboards (AVAIL-04) | 3 | DevOps + BE | Error rate > 1%, p95 > 500ms, failed login spike > 10x — all alerting |
| Developer integration documentation | 3 | BE Lead + SA | API reference; integration guide; error codes; authentication flows; sandbox tenant setup |
| Pentest remediation verification | — | QA + Pentest vendor | Re-test of Critical/High findings after remediation |

### Dependencies
- Sprint 7 complete; all Must Have + primary Should Have features deployed
- Pentest findings report fully received (all findings, not just preliminary)
- GDPR erasure: compliance consultant has reviewed the erasure approach by Sprint 7 end
- Performance test environment must match production spec (load testing requires representative hardware)
- Monitoring infrastructure provisioned (Sprint 0 / Sprint 2 baseline, validated now)

### Definition of Ready (Entry Criteria)
- Full pentest findings report received from vendor
- Critical and High findings triaged with SA — remediation approach agreed
- Performance test tooling selected (e.g., k6, Locust, Vegeta)
- GDPR erasure requirements confirmed with compliance consultant

### Key Risks
- **HIGH:** Pentest findings exceed 8-point estimate — if Critical architectural findings require re-architecture, this sprint extends. PM must have contingency ready (see Risk Register R-04)
- Performance testing reveals bottlenecks requiring significant refactoring — mitigation: flag indexes and query hot paths in earlier sprints; index review at Sprint 4
- Sprint 8 has 25 story points — highest total. Monitor velocity daily; escalate if > 20% behind by Day 5
- July 4th (2026-07-03) — US public holiday on Sprint 8 last day; adjust if needed

### Ceremonies
Same cadence. Add: security review retrospective (SA + QA + BE Lead, 1 hour, post-pentest remediation).

---

## Sprint 9 — Launch Preparation and UAT
**Dates:** Week 19–20 (2026-07-06 to 2026-07-17)
**Story Points:** 15

### Sprint Goal
All Must Have stories are validated end-to-end in a production-equivalent environment, UAT participants have signed off, and the team has a documented go/no-go decision with a tested rollback plan.

### Stories and Tasks

| Story / Task | Points | Assignee | Notes |
|-------------|--------|----------|-------|
| End-to-end integration testing: all Must Have flows | 5 | QA + All | Full regression: 20 user stories tested end-to-end |
| Staging → production-equivalent environment validation | 3 | DevOps + QA | Environment parity check; secrets loaded; SSL certs; DNS |
| Incident response runbook | 2 | PM + SA + DevOps | Covers: auth service down, Redis down, DB failover, pentest incident |
| Go/no-go criteria definition and validation | 2 | PM + PO + SA | Checklist form; all stakeholders sign off |
| Smoke tests on production environment | 3 | QA + DevOps | Happy path: register → verify → login → refresh → logout per tenant |
| UAT: tenant admin scenario (Carlos) | included | QA + PO | Onboard tenant, assign roles, view audit log — < 5 min flow |
| UAT: developer scenario (Priya) | included | QA + PO | Integrate new service with OAuth flow — < half-day flow |
| UAT sign-off | — | PO + Stakeholders | Formal sign-off from PO and at least 1 tenant admin participant |
| Rollback plan documented and tested | — | DevOps + PM | Database rollback; service rollback; DNS cutover reversal |

### Dependencies
- Sprint 8 complete and all Critical/High pentest findings closed
- Production environment provisioned and accessible (DevOps)
- UAT participants (tenant admin, developer) identified and available Week 19
- All ADRs and developer documentation finalized
- Monitoring and alerting active on production-equivalent environment

### Definition of Ready (Entry Criteria)
- Sprint 8 DoD confirmed; zero open Critical/High security findings
- Performance test results passing all PERF thresholds
- UAT scenarios documented and distributed to participants
- Production environment available for smoke testing

### Key Risks
- UAT participant availability — mitigation: PM confirms UAT participants by Sprint 7 end
- Production environment differences from staging surface late issues — mitigation: environment parity audit in Sprint 8 prep
- Go/no-go decision delayed by stakeholder availability — mitigation: PM blocks calendar 2 weeks in advance

### Ceremonies
Same cadence. Add:
- UAT kickoff: Day 3 (PO, QA, PM, UAT participants — 1 hour)
- Go/no-go meeting: Day 9 (all stakeholders — 1 hour)
- Launch celebration: Day 10 (team)

---

## Post-Sprint 9: Hypercare Period
**Dates:** Week 21–22 (2026-07-20 to 2026-07-31)
**No story points; team on standby**

Production deployment target: 2026-07-20 (start of Week 21, contingent on Sprint 9 go/no-go)

Activities:
- Monitor production dashboards 24/7 for first 48 hours (DevOps on-call rotation)
- Triage any production incidents using incident response runbook
- Address any Severity 1 or 2 issues immediately
- Weekly status report to Sponsor for 4 weeks post-launch
- Compliance consultant begins SOC2 evidence collection framework
- Developer satisfaction survey distributed at 4 weeks post-launch

---

---

# Section 5: Project Timeline / Gantt-style Milestones

## 5.1 Master Timeline

Start date: 2026-03-02 (Week 1)
Target launch: 2026-07-20 (Week 21, production deployment)

| Week | Dates | Sprint | Key Milestone |
|------|-------|--------|--------------|
| 1–2 | 2026-03-02 to 2026-03-13 | Sprint 0 | **M0: Architecture Spike Complete** — ERD, migration runner POC, CI/CD green, staging ready |
| 3–4 | 2026-03-16 to 2026-03-27 | Sprint 1 | **M1: Core Auth Operational** — register, verify, login, reset working in staging |
| 5–6 | 2026-03-30 to 2026-04-10 | Sprint 2 | **M2: Sessions and Security Baseline** — refresh, logout, rate limiting, audit log live |
| 7–8 | 2026-04-13 to 2026-04-24 | Sprint 3 | **M3: Multi-Tenancy Live** — tenant provisioning, schema isolation, isolation tests in CI |
| 9–10 | 2026-04-27 to 2026-05-08 | Sprint 4 | **M4: RBAC and Admin API Complete** — roles, JWT claims, user management, JWKS, introspection |
| 11–12 | 2026-05-11 to 2026-05-22 | Sprint 5 | **M5: OAuth Authorization Server Live** — Authorization Code + PKCE operational |
| 13–14 | 2026-05-25 to 2026-06-05 | Sprint 6 | **M6: Full OAuth Suite** — M2M + Google social login; **PENTEST BEGINS** |
| 15–16 | 2026-06-08 to 2026-06-19 | Sprint 7 | **M7: MFA and User Profile Complete**; pentest findings intake begins |
| 17–18 | 2026-06-22 to 2026-07-03 | Sprint 8 | **M8: Production Hardened** — pentest remediated, performance validated, docs ready |
| 19–20 | 2026-07-06 to 2026-07-17 | Sprint 9 | **M9: UAT Signed Off** — go/no-go approved, production environment validated |
| 21 | 2026-07-20 | Launch | **M10: PRODUCTION LAUNCH** |

## 5.2 Critical Path

The critical path runs through:

```
M0 (Sprint 0 complete)
  → M3 (Multi-tenancy — Sprint 3 is the highest-risk; any slip here cascades)
    → M5 (OAuth Authorization Server — Sprint 5 depends on RBAC from Sprint 4)
      → M6 (Sprint 6 — Pentest begins; any slip delays pentest = delays Sprint 8 remediation)
        → M8 (Sprint 8 — Pentest remediation; this sprint must not be shortened)
          → M10 (Launch)
```

**Critical Path items that cannot slip without impacting launch:**
1. Sprint 0: Migration runner POC (blocks Sprint 3 entirely)
2. Sprint 3: US-08b isolation test suite (non-negotiable; blocks safe progression)
3. Sprint 6: Pentest start (late pentest = compressed remediation time)
4. Sprint 8: Pentest remediation (unresolved Critical/High = no-go for launch)

## 5.3 Key Contractual / External Deadlines

| Deadline | Date | Consequence if Missed |
|----------|------|----------------------|
| Book penetration tester | 2026-03-06 (Sprint 0, Day 5) | Sprint 6/7 window unavailable; pentest pushed to Sprint 8+; launch at risk |
| Engage compliance consultant | By 2026-04-10 (Sprint 2 end) | SOC2 evidence collection framework delayed; post-launch compliance work harder |
| Google Cloud credentials ready | 2026-05-11 (Sprint 5 start) | US-13 (Google login) cannot start; Sprint 6 delay |
| Pentest execution window | 2026-05-25 to 2026-06-19 (Sprint 6/7) | Late findings = insufficient remediation time in Sprint 8 |
| UAT participants confirmed | By 2026-06-19 (Sprint 7 end) | UAT cannot begin on time in Sprint 9 |

## 5.4 Milestone Summary (Visual)

```
WEEK:  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21
       |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |
S0:    [======]
S1:          [======]
S2:                [======]
S3:                      [======]   << CRITICAL
S4:                            [======]
S5:                                  [======]
S6:                                        [======]   << PENTEST
S7:                                              [======]
S8:                                                    [======]
S9:                                                          [======]
LAUNCH:                                                              *

M0: W2    M1: W4    M2: W6    M3: W8*   M4: W10   M5: W12
M6: W14   M7: W16   M8: W18   M9: W20   M10: W21
```
*Sprint 3 is on the critical path — any slip propagates.

---

---

# Section 6: Resource Allocation Plan

## 6.1 Team Roster

| Role | Allocation | Sprint 0 | Sprint 1–2 | Sprint 3 | Sprint 4–7 | Sprint 8 | Sprint 9 |
|------|-----------|---------|-----------|---------|-----------|---------|---------|
| Backend Engineer 1 | 100% | Repo + skeleton | Core auth (US-01, 03) | Tenant provisioning (US-07a, 08a) | RBAC, OAuth | Pentest remediation | E2E testing |
| Backend Engineer 2 | 100% | Monorepo + migration POC | Email, US-02 | Migration runner (US-07b, 08b collab) | Admin API, OAuth consent | Hardening | E2E testing |
| Backend Engineer 3 | 100% | Spike research | Password reset, US-06 | Tenant credentials (US-07c) | JWKS, M2M | GDPR endpoint | E2E testing |
| Frontend Engineer | 50% | Next.js shell | — | — | OAuth consent (Sprint 5 only) | Docs support | — |
| QA Engineer | 100% | Test strategy, env setup | Sprint 1 test execution | Isolation test suite (US-08b), Sprint 3 QA | Sprint 4–7 QA | Perf tests, OWASP, pentest verify | UAT, regression |
| DevOps Engineer | 50% | Infra provisioning (full 100% this sprint) | On-demand | On-demand | On-demand | Monitoring/alerts | Prod env, smoke tests |
| Solution Architect | 100% Sprint 0, then advisory | Design, ADRs, ERD, middleware design | Advisory (sprint reviews) | Sprint 3 arch checkpoint | Advisory | Review remediation | Sign-off |
| Product Owner | 50% (ceremonies) | Sprint 0 kick-off | Planning, review, refinement | Planning, review, refinement | Planning, review, refinement | Review | UAT sign-off |
| Project Manager | 100% | Plan, vendor booking, setup | Facilitation, tracking | Sprint 3 daily risk monitoring | Facilitation, tracking | Escalation, budget | Go/no-go, launch |

## 6.2 Sprint-by-Sprint Resource Allocation

### Sprint 0 — All Hands
- DevOps: **100%** (elevated — all infra must be ready by end of Sprint 0)
- SA: **100%** (elevated — Sprint 0 is SA's primary sprint)
- BE Lead: **100%** on migration runner POC + repo setup
- BE-2, BE-3: Research and skeleton work
- QA: Test strategy + test environment setup
- FE: Next.js shell

**Bottleneck alert:** DevOps is normally 50% but Sprint 0 requires full-time availability. Confirm shared resource can be 100% in Sprint 0 with their manager. If not, infrastructure delivery may slip.

### Sprint 1–2 — Backend Heavy
- Three BE engineers at 100%, all on core auth and security features
- QA at 100%, building test cases and executing
- DevOps on-demand (staging support)
- FE idle (no frontend work until Sprint 5)
- SA advisory only

**Bottleneck alert:** Email service integration in Sprint 1 requires coordination between BE-2 and DevOps for credential injection. PM must verify this is non-blocking.

### Sprint 3 — Highest Risk, Highest Coordination
- BE-1: Tenant provisioning + schema-routing middleware — these are tightly coupled; allocate full sprint
- BE-2: Migration runner + collaboration on US-08b — migration runner is the riskiest workstream
- BE-3: Tenant API credentials (lighter, can assist BE-1 or BE-2 if needed)
- QA: Cross-tenant isolation test suite (US-08b) — QA co-authors with BE-2; this is a significant QA effort
- SA: Mid-sprint review (Day 5) — mandatory checkpoint; SA must be available

**Bottleneck alert:** BE-2 owns both Migration Runner (US-07b) and co-owns Isolation Test Suite (US-08b). These are both 5-point stories. PM should monitor BE-2 utilization daily; pull BE-3 in if BE-2 is behind.

### Sprint 4 — Buffer Sprint
- BE-1: US-10 (JWT claims) + M17 (introspection) — token-related work, logically grouped
- BE-2: M16 (admin user management) — separate concern
- BE-3: US-09 (roles) + M18 (JWKS)
- QA: Sprint 4 testing + Sprint 3 regression

**No bottleneck expected.** This sprint is intentionally 15 points to absorb Sprint 3 carryover.

### Sprint 5 — OAuth Sprint
- BE-1 + BE-2: US-11b and US-11c (authorize + token exchange) — most complex OAuth work; pair programming recommended
- BE-3: US-11a (client registration — lighter)
- FE: **Activated this sprint** (0.5 FE) — OAuth consent mechanism
- SA: Available for OAuth library decision on Day 1

**Bottleneck alert:** OAuth 2.0 complexity is high. BE-1 and BE-2 should pair. SA must be reachable for edge case questions throughout the sprint, not just at review.

### Sprint 6 — External Integrations + Pentest
- BE-1 + BE-2: US-13 (Google social login) — 8 points, two engineers
- BE-3: US-12 (M2M client credentials) — 5 points, one engineer
- QA: Sprint 6 testing + pentest coordination support
- PM: Pentest coordination (daily check-in with vendor)

**Bottleneck alert:** Pentest runs concurrently. BE team may be pulled for questions or clarifications by the tester. Allocate 10% overhead to BE team for pentest support.

### Sprint 7 — MFA and Pentest Findings
- BE-1: TOTP login verification (S1, second half)
- BE-2: TOTP setup (S1, first half) + User profile (S5)
- BE-3: MFA enforcement (S2)
- QA: Sprint 7 testing + pentest findings intake/triage

**Bottleneck alert:** If Critical pentest findings arrive mid-Sprint 7, PM may need to redirect BE resources. Have a pre-agreed priority: Critical pentest findings > MFA enforcement > User profile API.

### Sprint 8 — Hardening (All Hands)
- All three BE: Pentest remediation (8 pts estimated — may need all three engineers)
- BE-3: GDPR erasure endpoint
- QA: Performance testing, OWASP review, pentest re-verification
- DevOps: Monitoring, alerting, log aggregation
- BE Lead + SA: Developer documentation

**Bottleneck alert:** This is the sprint most likely to need additional time. PM should track pentest finding count from Sprint 6 and update capacity estimate accordingly. If > 15 Critical/High findings, escalate to Sponsor for timeline discussion at Sprint 7 review.

### Sprint 9 — QA and Launch Prep
- QA: 100% on end-to-end testing, UAT support
- DevOps: Production environment validation, smoke tests
- BE: Available for defect fixes from QA/UAT
- PM: Go/no-go facilitation, launch coordination

**No new feature work in Sprint 9.** Any BE time not consumed by defect fixes can be used for technical debt or documentation polish.

## 6.3 Allocation Summary Table

| Sprint | BE-1 | BE-2 | BE-3 | FE | QA | DevOps | SA |
|--------|------|------|------|----|----|--------|----|
| 0 | 100% | 100% | 100% | 50% | 100% | **100%** | **100%** |
| 1 | 100% | 100% | 100% | — | 100% | 30% | 20% |
| 2 | 100% | 100% | 100% | — | 100% | 20% | 20% |
| 3 | 100% | 100% | 100% | — | 100% | 20% | **40%** |
| 4 | 100% | 100% | 100% | — | 100% | 20% | 20% |
| 5 | 100% | 100% | 100% | **50%** | 100% | 20% | **30%** |
| 6 | 100% | 100% | 100% | — | 100% | 20% | 20% |
| 7 | 100% | 100% | 100% | — | 100% | 20% | 20% |
| 8 | 100% | 100% | 100% | — | 100% | **50%** | **30%** |
| 9 | 70% | 70% | 70% | — | 100% | **50%** | 20% |

Bolded entries indicate elevated allocation vs. standard.

## 6.4 Overload and Bottleneck Risk Summary

| Risk | Sprint | Affected Role | Mitigation |
|------|--------|--------------|------------|
| DevOps at 100% required in Sprint 0 but is a 50% shared resource | Sprint 0 | DevOps | Confirm 100% availability with DevOps manager before project start |
| BE-2 owns two 5-pt stories in Sprint 3 (migration runner + isolation suite) | Sprint 3 | BE-2 | Daily monitoring; BE-3 on standby to assist |
| QA handles both isolation test suite authoring and regular test execution in Sprint 3 | Sprint 3 | QA | QA starts drafting isolation tests during Sprint 2 (prep work) |
| All three BE working on pentest remediation in Sprint 8 may leave no bandwidth for other stories | Sprint 8 | All BE | Accept risk; 8 pt estimate is placeholder; update after Sprint 6 pentest kickoff |
| FE resource (0.5) only engaged one sprint (Sprint 5) — minimal context; may be unfamiliar | Sprint 5 | FE | SA and BE Lead provide FE with API spec before Sprint 5 starts |

---

---

# Section 7: Risk Register

**Scoring:** Probability (1=Low, 2=Medium, 3=High) × Impact (1=Low, 2=Medium, 3=High, 4=Critical) = Score

| ID | Risk | Category | Probability | Impact | Score | Mitigation | Owner | Status | Trigger |
|----|------|----------|------------|--------|-------|-----------|-------|--------|---------|
| R-01 | Migration runner complexity exceeds Sprint 0 POC scope — multi-schema concurrent migration, rollback handling | Technical | 2 | 3 | **6** | Sprint 0 POC must prove the critical path end-to-end. BE Lead escalates Day 1 if blockers. If POC not ready by Sprint 0 Day 8, PM escalates to SA and re-plans Sprint 3. | BE Lead | Open | POC not demo-able by Sprint 0 Day 8 |
| R-02 | OAuth 2.0 edge cases cause Sprint 5 overrun | Technical | 3 | 2 | **6** | Battle-tested OAuth library is non-negotiable. SA approves library choice. Strict timebox. If sprint overruns by > 2 pts, defer lower-priority OAuth edge cases to Sprint 6 buffer. | SA + BE Lead | Open | Sprint 5 velocity < 70% by Day 7 |
| R-03 | Cross-tenant data leak discovered in test or production | Security | 1 | 4 | **4** | US-08b isolation test suite is non-negotiable. Runs in CI on every PR. Any failure blocks merge. SA reviews schema-routing middleware before Sprint 3 merge. | SA + QA | Open | Isolation test fails on any PR |
| R-04 | Penetration test uncovers Critical architectural findings requiring re-architecture | Security | 2 | 4 | **8** | Book pentest early (Sprint 0). Execute Sprint 6/7 window. If Critical architectural finding: PM convenes emergency architecture review within 24h. Timeline extension requested from Sponsor. | PM + SA | Open | Pentest report contains ≥1 Critical finding with CVSS ≥ 9.0 |
| R-05 | "Can we add SAML?" scope creep request | Process | 3 | 2 | **6** | PRD explicitly marks SAML as out of scope (OS-01). Change Control Process applies. PM declines informally first, then formally if escalated. Estimate: SAML = +3 sprints minimum. | PM + PO | Open | Formal or informal SAML request from any stakeholder |
| R-06 | Google OAuth API changes or credential setup delays | External | 1 | 2 | **2** | Pin to OIDC discovery endpoint. Monitor Google OAuth changelog. Provision Google credentials in Sprint 0. Verify working in Sprint 5 (before Sprint 6 depends on them). | BE Lead + DevOps | Open | Google credential validation fails in Sprint 5 |
| R-07 | Refresh token rotation edge cases underestimated — family revocation complexity | Technical | 2 | 3 | **6** | Designate BE-1 as auth security champion. Required reading: RFC 6749, 7009, 7519, 7636. QA runs specific reuse-attack scenarios. SA reviews design before implementation. | BE-1 + SA | Open | Reuse attack test fails in Sprint 2 QA |
| R-08 | Schema-per-tenant migration tooling underestimated — Sprint 3 slip cascades to all subsequent sprints | Technical | 2 | 3 | **6** | Sprint 0 POC is mandatory. SA + BE Lead review POC. If Sprint 3 looks at risk by Day 5, pull BE-3 from US-07c to support migration runner. Do not proceed to Sprint 4 until US-08b is complete. | BE Lead + PM | Open | Sprint 3 velocity < 60% by Day 5 |
| R-09 | DevOps shared resource (0.5) unavailable in Sprint 0 at required 100% | Resource | 2 | 3 | **6** | Confirm 100% DevOps availability in Sprint 0 before project start. Have fallback: SA can guide BE Lead on secrets manager setup. Cloud provider documentation used for self-service provisioning where possible. | PM | Open | DevOps confirms < 80% availability in Sprint 0 before kickoff |
| R-10 | Penetration test vendor cannot be booked in time for Sprint 6/7 window | External | 2 | 3 | **6** | Start vendor outreach on Sprint 0 Day 1. Contact two vendors in parallel. If no vendor available for Sprint 6/7, PM evaluates: (a) delay launch, or (b) use internal red team as interim with external pentest post-launch. | PM | Open | No vendor confirmed by Sprint 0 Day 10 |
| R-11 | Team member extended absence (illness, departure) | Resource | 2 | 3 | **6** | Knowledge sharing in sprints: no single point of knowledge. Code review ensures two engineers understand every area. If one BE leaves: PM re-plans immediately; Sprint 3 has highest bus-factor risk (migration runner). | PM | Open | Any team member absent > 3 consecutive days without replacement |
| R-12 | Compliance consultant not identified or engaged by Sprint 2 | External | 2 | 2 | **4** | PM begins search in Sprint 0 in parallel with pentest vendor. Compliance consulting firms typically have 2–4 week lead time. If no consultant by Sprint 2, PM escalates to Sponsor for direct network referral. | PM | Open | No consultant identified by Sprint 1 Week 2 |
| R-13 | Email service delivery issues in staging delay Sprint 1 testing | Technical | 2 | 2 | **4** | Test email send from CI as Sprint 0 exit criterion. If provider unreliable in staging, use verified email mock/stub for testing; real provider for staging integration test only. | BE-2 + DevOps | Open | Email delivery test fails at Sprint 0 exit |
| R-14 | Audit log retroactive wiring to Sprint 1 endpoints adds unplanned work in Sprint 2 | Process | 2 | 2 | **4** | Design audit log event emission interface in Sprint 1 (even if no storage yet). Wiring in Sprint 2 then becomes configuration, not redesign. BE Lead responsible for this design upfront. | BE Lead | Open | US-15 estimate increases > 2 pts in Sprint 2 planning |
| R-15 | Sprint 8 pentest remediation significantly underestimated (> 15 Critical/High findings) | External | 2 | 4 | **8** | PM monitors pentest findings as they arrive from Sprint 6 onward. If finding count trajectory suggests > 8 pts remediation: update estimate in Sprint 7 planning; request timeline extension or scope reduction from Sponsor. | PM | Open | Pentest preliminary findings show > 8 Critical/High by Sprint 7 Day 5 |
| R-16 | Argon2id hashing latency exceeds budget — login p95 approaches or exceeds 300ms | Technical | 2 | 2 | **4** | Benchmark Argon2id parameters on target hardware in Sprint 0/1. Tune memory/iterations to achieve ~200–250ms hash time. If cannot achieve < 300ms p95, discuss bcrypt as fallback with SA (cost ≥ 12). | BE-1 + SA | Open | Sprint 1 login latency benchmark > 250ms |
| R-17 | Scope creep from Should Have items S3, S4, S8, S9, S10 being added mid-project | Process | 2 | 2 | **4** | Change Control Process applies to all Should Have additions. PM maintains scope boundary. PO must justify business case and trade-off (what is removed or timeline extended). | PM + PO | Open | PO requests addition of any Should Have beyond S1, S2, S5 |
| R-18 | PostgreSQL schema count growth — performance degradation at scale during pilot | Technical | 1 | 2 | **2** | Monitor schema count and per-schema query plan cost. Index usage reviewed in Sprint 4 (PERF-04). Performance testing in Sprint 8 covers realistic tenant count. ADR-001 documents scale limitation. | SA + BE Lead | Open | PostgreSQL schema count > 500 in staging performance test |

### Risk Register Summary

| Score | Count | Risks |
|-------|-------|-------|
| 8 (Critical) | 2 | R-04 (Pentest architectural finding), R-15 (Pentest finding volume) |
| 6 (High) | 6 | R-01, R-02, R-05, R-07, R-08, R-09, R-10, R-11 |
| 4 (Medium) | 5 | R-03, R-12, R-13, R-14, R-16, R-17 |
| 2 (Low) | 2 | R-06, R-18 |

**Top 3 risks requiring immediate PM action at project start:**
1. R-04 / R-15: Book penetration tester Sprint 0 Day 1
2. R-09: Confirm DevOps 100% availability in Sprint 0 before kickoff
3. R-01 / R-08: Validate migration runner POC is achievable with SA and BE Lead before Sprint 0 Day 1

---

---

# Section 8: Budget Estimate

## 8.1 Team Cost Estimate

**Assumptions:**
- Market rates are for experienced engineers in a mid-tier technology market (e.g., US remote, mid-level to senior)
- All rates are fully-loaded (salary + benefits + overhead at approximately 1.4x base)
- Project duration: 22 weeks (20 delivery + 2 hypercare), starting 2026-03-02

| Role | Allocation | Monthly Rate (fully-loaded) | 5-month Cost | Notes |
|------|-----------|----------------------------|-------------|-------|
| Backend Engineer 1 | 100% | $16,000/month | $80,000 | Senior Go engineer |
| Backend Engineer 2 | 100% | $16,000/month | $80,000 | Senior Go engineer |
| Backend Engineer 3 | 100% | $14,000/month | $70,000 | Mid-senior Go engineer |
| Frontend Engineer | 50% | $7,000/month | $35,000 | Mid-level, 0.5 FTE for 5 months |
| QA Engineer | 100% | $11,000/month | $55,000 | Senior QA, automation-capable |
| DevOps Engineer | 50% | $8,000/month | $40,000 | 0.5 FTE, shared resource |
| Solution Architect | 100% Sprint 0 + 25% thereafter | ~$7,000/month blended | $35,000 | 2 weeks full + 18 weeks advisory |
| Product Owner | 50% | $8,000/month | $40,000 | PO time for ceremonies + decisions |
| Project Manager | 100% | $13,000/month | $65,000 | Senior PM |
| **Team Subtotal** | | | **$500,000** | |

## 8.2 Infrastructure Costs

| Item | Monthly Cost | 5-Month Cost | Notes |
|------|-------------|-------------|-------|
| Cloud compute (staging + production) | $800 | $4,000 | 4–6 mid-tier VMs (backend, DB replicas, Redis, monitoring) |
| PostgreSQL managed service | $600 | $3,000 | Primary + read replica; managed service preferred (e.g., RDS, Cloud SQL) |
| Redis managed service | $200 | $1,000 | Cache-class instance; HA optional for staging |
| Secrets manager | $150 | $750 | HashiCorp Vault Cloud or cloud-native (AWS Secrets Manager) |
| Object storage / S3 (audit log cold archive) | $50 | $250 | Low volume in pilot; priced for 1-year retention |
| TLS certificates | $0–50 | $50 | Let's Encrypt (free) or managed cert ($50 one-time) |
| DNS | $20 | $100 | Managed DNS for staging and production domains |
| Monitoring / observability | $300 | $1,500 | Datadog, Grafana Cloud, or equivalent |
| Log aggregation | $200 | $1,000 | CloudWatch, Loki, or equivalent |
| Load testing infrastructure (Sprint 8) | $100 | $100 | One-time; k6 Cloud or equivalent for Sprint 8 only |
| **Infrastructure Subtotal** | **~$2,420/month** | **$11,750** | |

## 8.3 External Services and Vendors

| Item | Cost | When | Notes |
|------|------|------|-------|
| External penetration test | $15,000–$25,000 | Sprint 6/7 | 1–2 week engagement; web application pentest (OWASP methodology); API focus. Budget at $20,000. |
| SOC2 compliance consultant | $8,000–$15,000 | Ongoing from Sprint 2 | Engagement to Sprint 8; evidence collection framework setup. Budget at $12,000. |
| Transactional email service | $50–200/month | From Sprint 1 | SendGrid, Postmark, AWS SES; budget at $100/month × 5 months = $500 |
| Google Cloud project (OAuth) | $0 | Sprint 0 | OAuth 2.0 with Google is free; only costs if Google Workspace APIs are used |
| OAuth 2.0 library | $0 | Sprint 5 | Open source library (e.g., fosite, ory/hydra components) |
| **External Services Subtotal** | **$33,000–$50,000** | | Budget midpoint: $43,000 |

## 8.4 Contingency

| Item | Amount |
|------|--------|
| Technical contingency (15% of team cost) | $75,000 |
| Scope contingency (10% of team cost) | $50,000 |
| Infrastructure over-provisioning buffer | $5,000 |
| **Contingency Subtotal** | **$130,000** |

Note: A 15% technical contingency is appropriate given Sprint 3 and Sprint 8 risks. The pentest finding remediation estimate has high variance.

## 8.5 Total Budget Summary

| Category | Amount |
|---------|--------|
| Team (people) | $500,000 |
| Infrastructure | $11,750 |
| External services (pentest, compliance, email) | $43,000 |
| Contingency | $130,000 |
| **Total Project Budget** | **$684,750** |

## 8.6 Monthly Burn Rate

| Month | Dates | Phase | Estimated Burn |
|-------|-------|-------|---------------|
| Month 1 | March 2026 | Sprint 0 + Sprint 1 | $115,000 (team + infra setup) |
| Month 2 | April 2026 | Sprint 2 + Sprint 3 | $110,000 |
| Month 3 | May 2026 | Sprint 4 + Sprint 5 | $108,000 |
| Month 4 | June 2026 | Sprint 6 + Sprint 7 + pentest | $138,000 (pentest $20K) |
| Month 5 | July 2026 | Sprint 8 + Sprint 9 + launch | $120,000 (compliance consultant closeout) |
| **Total** | | | **$591,000** (within budget with contingency remaining) |

**Budget monitoring:** PM produces budget actuals vs plan in weekly status report. Alert threshold: actual spend > plan by 10% in any single month.

---

---

# Section 9: Communication Plan

## 9.1 Meeting Cadence

| Meeting | Frequency | Duration | Participants | Facilitator | Output |
|---------|-----------|----------|-------------|-------------|--------|
| Daily Standup | Daily (Mon–Fri) | 15 min | All engineers, QA, PM | PM | Blocker log updated |
| Sprint Planning | Start of each sprint | 2 hours | All team | PM | Sprint backlog committed |
| Backlog Refinement | Bi-weekly (mid-sprint) | 1 hour | PM, PO, BE Lead, QA | PM | Next sprint stories refined and estimated |
| Sprint Review | End of each sprint | 1 hour | All team + Stakeholders | PM | Demo of completed stories; PO acceptance |
| Sprint Retrospective | End of each sprint | 45 min | All team | PM | Action items for next sprint |
| Architecture Review | Sprint 0 (intensive) + Sprint 3 Day 5 + on-demand | 1 hour | SA, BE Lead, PM | SA | Architecture decisions documented |
| PM–PO Weekly Sync | Weekly (Wednesday) | 30 min | PM, PO | PM | Priority, scope, blockers |
| Stakeholder Status Update | Monthly | 30 min | PM, PO, Sponsor, Stakeholders | PM | Status report reviewed |
| Risk Review | Bi-weekly (end of sprint planning) | 20 min | PM, SA, QA | PM | Risk register updated |
| Vendor Check-in | Weekly from Sprint 6 (pentest) | 15 min | PM, QA, Pentest vendor | PM | Finding status, ETA, severity |
| Go/No-Go Meeting | Sprint 9 Day 9 | 1 hour | PM, PO, SA, Sponsor, QA, DevOps | PM | Go/No-Go decision recorded |

## 9.2 Status Reporting

### Weekly Status Report (PM to Sponsor + all stakeholders)
Distributed every Friday by 5:00 PM.

**Template:**

```
Authentication System — Weekly Status Report
Week [N] | [Date Range] | Sprint [N]

STATUS: [GREEN / AMBER / RED]
  GREEN = On track (timeline, scope, budget)
  AMBER = At risk — one dimension behind with mitigation in place
  RED = Off track — requires Sponsor decision

THIS WEEK:
- Completed: [stories/tasks completed]
- In Progress: [stories/tasks in flight]
- Blocked: [blockers with owner and ETA]

NEXT WEEK:
- Planned: [stories/tasks planned for next week]

METRICS:
- Sprint velocity: [actual] vs [planned] pts
- Stories completed this sprint: [N/total]
- Budget actuals: $[X] of $[plan]; variance: [+/-]%
- Open risks: [count]; new risks: [count]
- Open defects: [Severity 1: N | Severity 2: N | Total: N]

RISKS AND ISSUES:
[List any AMBER/RED items with owner and mitigation]

DECISIONS NEEDED:
[List any decisions that require Sponsor/PO input this week]
```

### Sprint Review Summary
Distributed after each sprint review to all stakeholders.

Contents:
- Stories accepted vs planned (velocity)
- Demo highlights
- Defects found in sprint
- Next sprint preview

### Monthly Executive Summary
Distributed to Sponsor at end of Months 1–5.

Contents:
- Milestone status (on track / at risk)
- Budget actual vs plan
- Top 3 risks and mitigations
- Decisions made by Sponsor this month
- Upcoming decisions required next month

## 9.3 Communication Channels

| Channel | Purpose | Audience | Response SLA |
|---------|---------|---------|-------------|
| Jira | Story tracking, defect management, sprint boards | All team | Real-time |
| Slack / Teams (project channel) | Daily async communication, blocker flags | All team | 2 hours during business hours |
| Slack / Teams (PM–PO channel) | Priority, scope, quick decisions | PM, PO | 1 hour |
| Email | Formal status reports, vendor communication, contracts | All stakeholders | 24 hours |
| Confluence / Docs | Architecture docs, ADRs, this PMP, meeting notes | All | Read-only; updates on merge |
| Video call (Zoom/Meet) | Standups, planning, review, retrospective | Per meeting invite | — |

## 9.4 Escalation Path

| Issue Type | First Escalation | Second Escalation | Timeline |
|-----------|-----------------|------------------|----------|
| Technical blocker (code, architecture) | SA → BE Lead resolves | PM escalates to Sponsor if > 2 sprint days unresolved | Same day |
| Resource conflict (DevOps shared) | PM negotiates with DevOps manager | Sponsor resolves | 24 hours |
| Scope change request | PM → PO decision (Change Control Process) | Sponsor if PO and PM disagree | 3 business days |
| Budget variance > 10% | PM to Sponsor (immediate) | Board if > 20% | Same day |
| Security incident in staging | QA + SA + PM notified immediately | Sponsor notified within 4 hours | Immediate |
| Pentest Critical finding | SA + PM convene within 24h | Sponsor notification within 48h | 24 hours |
| Team member departure | PM to Sponsor within 24h; re-planning begins immediately | HR and recruiting engaged | 24 hours |
| Timeline slip ≥ 1 sprint | PM to Sponsor immediately | Board approval for > 2 sprint slip | Same day |

---

---

# Section 10: Change Control Process

## 10.1 Principles

1. The PRD v1.1 with its 20 Must Have stories and the defined out-of-scope list constitute the approved baseline.
2. All changes to scope, timeline, or budget above the PM authority threshold require formal change control.
3. No change is implemented until it is approved through this process — regardless of verbal agreement.
4. Every approved change is logged, dated, and linked to the sprint or milestone it affects.
5. Rejected changes are also logged, with rationale, to prevent re-submission without new information.

## 10.2 Change Categories and Approval Authority

| Category | Definition | Approver | Turnaround |
|----------|-----------|---------|-----------|
| Minor Adjustment | Story subtask added, removed, or reordered within an approved sprint; no scope impact; no timeline impact | PM authority | Same day |
| Story-Level Change | Adding or removing a user story within approved Must Have list; no net scope or timeline change (trade-off) | PM + PO | 1 business day |
| Scope Addition | Adding any new user story or feature not in the approved PRD backlog | PO + PM + Sponsor | 3 business days |
| Scope Removal | Removing a Must Have story from the delivery plan | PO + PM | 2 business days |
| Timeline Extension | Extending project timeline by 1 sprint (2 weeks) | PM + Sponsor | 2 business days |
| Timeline Extension (Major) | Extending project timeline by > 1 sprint | Sponsor + Board | 5 business days |
| Budget Increase | Any increase to approved budget | Sponsor | 3 business days |
| Architectural Change | Any change to ADR-001 through ADR-009 | SA + PM + Sponsor | 3 business days |
| Out-of-Scope Feature | Any feature in the PRD Won't Do list (e.g., SAML) | Sponsor (after full impact assessment) | 5 business days |

## 10.3 Change Request Process

**Step 1: Request Submission**
Any team member or stakeholder may submit a change request. The PM provides a standardized Change Request Form (see 10.4).

**Step 2: Initial Assessment (PM)**
PM reviews within 1 business day:
- Is this already in scope? (If yes, it is a clarification, not a change)
- Which category does it fall into?
- What is the preliminary impact (scope, timeline, budget)?
- Should it be escalated to SA for architectural impact assessment?

**Step 3: Impact Analysis**
- PM and SA assess: story point estimate, sprint impact, resource impact, dependency changes
- If architectural: SA produces a one-page impact memo
- Risk Register updated if new risks introduced

**Step 4: Decision**
- Approver reviews full impact analysis
- Decision: Approved / Rejected / Deferred
- If approved: sprint plan updated, PO accepts change into backlog

**Step 5: Communication**
- PM communicates decision to all stakeholders within 24 hours of decision
- Change log updated
- If approved: Jira updated, sprint board updated

## 10.4 Change Request Form Template

```
CHANGE REQUEST FORM — Authentication System Pilot

CR Number: [PM assigns; format CR-YYYY-NNN]
Date Submitted:
Submitted By:
Priority: [Low / Medium / High / Urgent]

DESCRIPTION OF CHANGE:
[What is being requested? Be specific.]

REASON / BUSINESS JUSTIFICATION:
[Why is this change needed? What problem does it solve?]

CURRENT BASELINE IMPACT:
- Scope impact: [None / Minor / Significant]
- Timeline impact: [None / +N days / +N sprints]
- Budget impact: [None / +$N]
- Risk introduced: [None / Low / Medium / High]
- ADRs affected: [List ADR numbers or "None"]

STORIES / TASKS AFFECTED:
[List US IDs or task names]

DEPENDENCIES INTRODUCED OR CHANGED:
[Any new dependencies this creates]

PROPOSED TRADE-OFF (if applicable):
[If adding scope: what is being removed to accommodate?]

ATTACHMENTS:
[Designs, specs, SA impact memo if applicable]

--- PM ASSESSMENT ---
Category: [Minor / Story / Scope Addition / Scope Removal / Timeline / Budget / Architectural]
Preliminary estimate: [N story points / N days / $N]
SA review required: [Yes / No]
Decision authority: [PM / PM+PO / PM+PO+Sponsor / Sponsor+Board]

--- DECISION ---
Date of Decision:
Decision: [APPROVED / REJECTED / DEFERRED]
Decision Made By:
Conditions (if any):
Sprint / Release affected:

--- COMMUNICATION ---
Date communicated to team:
Jira ticket updated: [Yes / No]
```

## 10.5 Change Log

The PM maintains a running Change Log in the project wiki. Fields: CR Number | Date Submitted | Description | Category | Submitter | Status | Approved By | Date Decided | Sprint Impact.

## 10.6 SAML Scope Creep — Pre-Authorized Response

Given the HIGH probability of a SAML request (PRD Risk Register), the following is the pre-authorized PM response:

*"SAML 2.0 is explicitly out of scope for this pilot per OS-01 in the approved PRD. SAML implementation requires a minimum of 3 additional sprints (+6 weeks) and significant additional budget. If business requirements have changed such that SAML is now required for launch, this must be raised as a formal change request via the Change Control Process, with Sponsor approval, and a full re-planning exercise. We will not proceed informally."*

---

---

# Section 11: Vendor and External Dependencies

## 11.1 Google OAuth 2.0 Credentials

| Field | Detail |
|-------|--------|
| Purpose | Social login via Google (US-13); OIDC discovery for user authentication |
| Type | Google Cloud OAuth 2.0 application credentials (client_id + client_secret) |
| Owner | DevOps + PM (procurement); BE Lead (integration) |
| When Needed | Credentials must be in secrets manager before Sprint 6 Day 1 (2026-05-25) |
| Action Required | Create Google Cloud project in Sprint 0; configure OAuth consent screen; add authorized redirect URIs for staging and production |
| Storage | Stored in secrets manager (ADR-009); never in code or env vars |
| Risk | R-06: Google API changes or setup delays |
| Mitigation | Set up and test credentials in Sprint 0. Verify working end-to-end in Sprint 5. Do not wait until Sprint 6. |
| Responsible | PM initiates; DevOps provisions; BE Lead validates |

## 11.2 Transactional Email Service Provider

| Field | Detail |
|-------|--------|
| Purpose | Email verification (US-02), password reset (US-06), user invitation (US-07a, M16) |
| Candidates | SendGrid, Postmark, AWS SES, Mailgun |
| Selection Criteria | (1) Reliable delivery rates; (2) Simple API; (3) GDPR-compliant data handling; (4) Cost at pilot scale (< $100/month) |
| Owner | PM (procurement); DevOps (API key management); BE-2 (integration) |
| When Needed | Provider contracted and test email sendable before Sprint 1 Day 1 (2026-03-16) |
| Action Required | Select provider in Sprint 0. Create account. Add API key to secrets manager. Send test email from CI. |
| Storage | API key in secrets manager (ADR-009) |
| Risk | R-13: Email delivery issues delay Sprint 1 testing |
| Mitigation | Test email delivery as Sprint 0 exit criterion |
| Responsible | PM contracts; DevOps provisions key; BE-2 integrates |

## 11.3 External Penetration Testing Vendor

| Field | Detail |
|-------|--------|
| Purpose | Independent security assessment of the authentication API before production launch |
| Scope of Work | Web application / API penetration test; OWASP Top 10 methodology; authentication-specific vectors (token forgery, PKCE bypass, session fixation, cross-tenant access, brute force, enumeration); staging environment access |
| Expected Duration | 1–2 weeks (Sprint 6 start to Sprint 7 midpoint) |
| Finding Remediation Window | Sprint 8 (2026-06-22 to 2026-07-03) |
| Deliverable | Written findings report: finding ID, severity (CVSS score), description, reproduction steps, recommendation. Re-test of Critical/High after remediation. |
| Budget | $15,000–$25,000; PM authority to approve up to $25,000 |
| When to Contract | Sprint 0 (contract signed by Week 2 end, 2026-03-13) |
| Long Lead Time Warning | Quality pentest vendors are often booked 6–8 weeks out. Outreach must begin Sprint 0 Day 1. |
| Vendor Requirements | (1) Experience with Go API security; (2) Experience with OAuth 2.0 and JWT security; (3) Can sign NDA; (4) Available Sprint 6/7 window; (5) Provides CVSS-scored findings |
| Responsible | PM (procurement, coordination); QA (technical liaison); SA (scope definition) |
| Risk | R-04, R-10, R-15 |

## 11.4 SOC2 Compliance Consultant

| Field | Detail |
|-------|--------|
| Purpose | Guide SOC2 Type II evidence collection framework; map ADRs to SOC2 Trust Service Criteria; prepare for eventual audit |
| Scope | Advisory engagement: (1) review architecture for SOC2 gaps; (2) define evidence collection procedures; (3) identify controls to document; (4) prepare evidence binder structure |
| When to Engage | By Sprint 2 end (2026-04-10) per ADR-008 |
| Engagement Duration | Ongoing advisory from Sprint 2 through launch; retainer or fixed-fee |
| Budget | $8,000–$15,000 for pilot phase advisory; budget at $12,000 |
| Key Deliverable | SOC2 evidence collection framework; controls mapping document; gap assessment report |
| Responsible | PM (procurement, coordination); PO (requirements); QA (evidence procedures) |
| Risk | R-12 |

## 11.5 Secrets Manager

| Field | Detail |
|-------|--------|
| Purpose | Store all signing keys, client secrets, API keys, database credentials (ADR-009) |
| Candidates | HashiCorp Vault (self-hosted or Cloud), AWS Secrets Manager, Azure Key Vault, GCP Secret Manager |
| Selection Criteria | (1) Supports dynamic secrets / key rotation; (2) Audit log of secret access; (3) Integration with Go (vault-client or cloud SDK); (4) Cost at pilot scale |
| When Needed | Operational before Sprint 1 Day 1; set up in Sprint 0 |
| Action Required | DevOps provisions secrets manager in Sprint 0. SA defines initial secrets namespaces. BE Lead integrates client library. |
| Storage | Signing keys (RS256/ES256), refresh token HMAC key (if used), DB credentials, email API key, Google OAuth credentials |
| Key Rotation | Must support key rotation without service restart (SEC-02) |
| Responsible | DevOps (provisioning); SA (design); BE Lead (integration) |

## 11.6 Cloud Hosting Provider

| Field | Detail |
|-------|--------|
| Purpose | Compute, managed PostgreSQL, managed Redis, object storage, networking |
| Assumed Provider | AWS, GCP, or Azure (selection by DevOps + PM in Sprint 0 if not pre-determined) |
| Components | Compute instances (backend service), managed PostgreSQL (RDS / Cloud SQL), managed Redis (ElastiCache / Memorystore), S3/GCS (audit log cold archive), VPC, load balancer, CDN (optional) |
| Environments | Staging (Sprint 0 onward) + Production-equivalent (Sprint 9) + Production (Launch) |
| Cost | See Budget Section (Section 8) |
| Data Residency | Must be single region (ADR, OS-06); region selection should account for data residency requirements of expected tenants |
| Responsible | DevOps (provisioning, configuration); SA (architecture guidance); PM (cost monitoring) |

## 11.7 Dependency Summary and Timeline

| Dependency | Owner | Required By | Status |
|-----------|-------|------------|--------|
| Penetration test vendor contracted | PM | 2026-03-13 (Sprint 0 end) | Not started |
| Email provider contracted + API key in secrets manager | PM + DevOps | 2026-03-16 (Sprint 1 Day 1) | Not started |
| Staging infrastructure fully operational | DevOps | 2026-03-16 (Sprint 1 Day 1) | Not started |
| Secrets manager operational | DevOps + SA | 2026-03-16 (Sprint 1 Day 1) | Not started |
| SOC2 compliance consultant engaged | PM | 2026-04-10 (Sprint 2 end) | Not started |
| Google Cloud OAuth credentials in secrets manager | PM + DevOps + BE Lead | 2026-05-11 (Sprint 5 start — tested) | Not started |
| Pentest vendor staging access granted | DevOps + PM | 2026-05-25 (Sprint 6 Day 1) | Not started |
| Pentest findings report (full) received | PM | 2026-06-19 (Sprint 7 end, at latest) | Not started |
| UAT participants confirmed | PM + PO | 2026-06-19 (Sprint 7 end) | Not started |
| Production environment provisioned | DevOps | 2026-07-06 (Sprint 9 Day 1) | Not started |

---

---

# Section 12: Project Closure Criteria

The Authentication System Pilot project is considered **closed** when ALL of the following criteria are verified and documented.

## 12.1 Feature Delivery Criteria

| Criterion | Verification Method | Owner |
|-----------|-------------------|-------|
| All 20 Must Have user stories (US-01 through US-15 plus M15–M18) are delivered, accepted, and deployed to production | PO formal acceptance in sprint reviews; production deployment confirmed | PO + PM |
| Should Have stories S1 (TOTP MFA), S2 (MFA enforcement), S5 (User Profile API) are delivered and accepted | PO formal acceptance | PO + PM |
| All user stories meet the Global Definition of Done (code review, unit tests ≥ 80%, integration tests, QA sign-off, docs updated) | QA DoD checklist per story | QA |

## 12.2 Quality and Security Criteria

| Criterion | Verification Method | Owner |
|-----------|-------------------|-------|
| External penetration test completed with zero unresolved Critical findings and zero unresolved High findings | Pentest vendor sign-off; re-test report | QA + SA |
| Cross-tenant isolation test suite passes on 100% of CI runs in production branch | CI pipeline green (last 30 runs minimum) | QA + DevOps |
| Performance targets met: login p95 < 300ms at 1,000 concurrent, token issuance p95 < 100ms, introspection p95 < 50ms | Performance test results (Sprint 8) | QA |
| OWASP Top 10 review complete; all findings remediated or risk-accepted with documentation | OWASP review report | QA + SA |
| All 15 audit event types are logged, verifiable, and append-only | QA audit log completeness test | QA |
| 100% of secrets stored in secrets manager; zero secrets in code, env vars, or committed config | SA sign-off + DevOps verification | SA |

## 12.3 Compliance Criteria

| Criterion | Verification Method | Owner |
|-----------|-------------------|-------|
| GDPR right-to-erasure endpoint functional: deletes user PII, anonymizes audit log entries | QA test + compliance consultant review | QA + Compliance |
| Audit log retention configured: 1-year hot + cold archive pipeline operational | DevOps configuration verified; test archive executed | DevOps |
| SOC2 evidence collection framework established; first evidence collected | Compliance consultant deliverable | PM + Compliance |
| ADR-001 through ADR-009 formally documented and accessible | Docs directory in repository | SA |

## 12.4 Infrastructure and Operational Criteria

| Criterion | Verification Method | Owner |
|-----------|-------------------|-------|
| Production environment deployed with all components operational: backend, PostgreSQL, Redis, secrets manager, monitoring | DevOps production checklist | DevOps |
| Monitoring and alerting active for all AVAIL-04 thresholds (error rate > 1%, p95 > 500ms, failed login spike > 10x) | Alert firing test | DevOps + QA |
| CI/CD pipeline deployed to production: pipeline deploys on merge to main with manual gate | Deployment test run | DevOps |
| Rollback plan documented and tested | Rollback test execution log | DevOps + PM |
| TLS 1.2+ enforced; HSTS configured; all secure headers active | Security headers scan (e.g., SecurityHeaders.io) | DevOps + QA |

## 12.5 Documentation Criteria

| Criterion | Verification Method | Owner |
|-----------|-------------------|-------|
| Developer integration guide published: covers all API endpoints, authentication flows, error codes, sandbox tenant setup | Guide reviewed by one external developer (pilot UAT participant) | BE Lead + SA |
| Incident response runbook complete and approved by SA | SA sign-off | PM + SA |
| Tenant onboarding operator guide available | PM review | BE Lead |
| Migration runner developer CLI guide available | PM review | BE-2 |

## 12.6 Acceptance and Sign-off Criteria

| Criterion | Verification Method | Owner |
|-----------|-------------------|-------|
| Product Owner formal sign-off on all Must Have stories | Written PO acceptance (email or Jira closure) | PO |
| UAT sign-off from at least one tenant admin participant and one developer/API consumer participant | Signed UAT acceptance form | PO + PM |
| Sponsor formal project closure approval | Signed closure document | PM + Sponsor |
| Go/no-go checklist completed and all items in GREEN | Go/no-go meeting minutes | PM |

## 12.7 Project Closure Activities

Upon meeting all criteria above, the PM will execute the following closure activities:

1. Produce the Project Closure Report:
   - Objectives met vs planned
   - Final velocity and story point delivery
   - Budget actuals vs plan
   - Risk outcomes (materialized risks, effectiveness of mitigations)
   - Timeline actuals vs plan
   - Lessons learned (from retrospectives)

2. Facilitate Lessons Learned Workshop (2 hours, all team):
   - What went well
   - What could have been better
   - Recommendations for the next authentication system iteration

3. Hand off to operations / on-call team:
   - Incident response runbook
   - Monitoring and alerting guide
   - On-call rotation (first 2 weeks PM-coordinated hypercare, then steady-state)

4. Archive all project artifacts:
   - PRD v1.1
   - This Project Management Plan
   - Sprint planning documents and meeting notes
   - Risk register (final state)
   - Change log
   - Pentest findings and remediation report
   - Performance test results
   - UAT sign-off forms
   - Compliance consultant deliverables

5. Sponsor closure sign-off and project formally closed in Jira.

---

## Post-Launch Success Tracking (6-Month Follow-up)

The PM or designated operational owner will track the following metrics for 6 months post-launch and report to Sponsor:

| Metric | Target | Measurement |
|--------|--------|-------------|
| New service integration time | ≤ 2 business days | Tracked per integration request |
| Login latency (p95) | < 300ms | Monitoring dashboards |
| Authentication error rate | < 0.1% | Monitoring dashboards |
| MFA adoption (admin users) | ≥ 60% | Analytics query on MFA enrollment |
| Security incidents from auth layer | 0 | Incident log |
| Audit log completeness | 100% | Spot audit quarterly |
| Tenant onboarding time | < 30 minutes | Timing logged in tenant provisioning |
| Developer satisfaction (1–5 survey) | ≥ 4.2 | Survey distributed at 90 days post-launch |

---

*End of Document — Authentication System Pilot: Project Management Plan v1.0*
*Produced by: Project Manager Agent | 2026-02-27*
*Input: PRD v1.1 (Product Owner Agent, 2026-02-27)*
*Intended for: Solution Architect, Developer, Tester agents and all project stakeholders*
