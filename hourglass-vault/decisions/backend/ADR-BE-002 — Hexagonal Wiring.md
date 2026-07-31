# ADR-BE-002 — Hexagonal Wiring: Ports, Services, Adapters

---
tags: ["adr", "backend", "architecture", "hexagonal", "promote"]

---

# ADR-BE-002 — Hexagonal Wiring: Ports, Services, Adapters

**Status:** Accepted · **(promote)** — candidate for the future global Go ADR set
**Date:** 2026-07-28
**Code:** `internal/core/{domain,ports,services}/`, `internal/adapters/{primary,secondary}/`, `cmd/server/main.go`

## Context

Hourglass is hexagonal (ports & adapters). The pattern is only worth its file count if the dependency rule and the struct/constructor conventions are applied identically everywhere — otherwise adapters leak into services and the "swap the database" promise is fiction. (It already held once: SurrealDB → PostgreSQL changed only the secondary layer.)

## Decision

**Dependency rule (acyclic, enforced):**

```
domain  ←  ports  ←  services  ←  primary adapters (HTTP)
            ↑
    secondary adapters (PostgreSQL) implement ports
```

* `internal/core/domain/28/` — pure entities, value objects, sentinel errors. Zero external imports (std lib + `uuid` only).
* `internal/core/ports/` — interfaces the core needs from the world (`*Repository`, `TokenService`, `PasswordHasher`, `UserFinder`).
* `internal/core/services/55/` — business logic. Depends on domain + ports only. Never imports `net/http` or `pgx`.
* `internal/adapters/primary/http/` — thin handlers: parse → call service → map error → format. No business logic.
* `internal/adapters/secondary/postgres/` — port implementations on `pgxpool`.

**Struct/constructor conventions:**

| Layer      | Struct                               | Constructor                       | Methods                                     |
| ---------- | ------------------------------------ | --------------------------------- | ------------------------------------------- |
| Service    | `Service`                            | `NewService(deps...) *Service`    | `(s *Service) Verb(ctx, req) (resp, error)` |
| Handler    | `*Handler` (e.g. `TimeEntryHandler`) | `NewHandler(svc) *Handler`        | `(h *Handler) Verb(w, r)`                   |
| Repository | `*Repository`                        | `NewRepository(pool) *Repository` | match the port interface                    |

**Wiring** happens once, in `cmd/server/main.go`: build pool → build secondary adapters → build services (inject ports) → build handlers → register routes. No service locators, no global state except the pool singleton ([[ADR-BE-003 — Data Access pgxpool No ORM]]).

## Consequences

* Business logic is testable without HTTP or a database (mock the ports — see `internal/core/services/testdata/`).
* Database/framework changes stay inside the secondary/primary layers.
* New features follow the 6-step path in `.planning/codebase/STRUCTURE.md` ("Where to Add New Code").
* ⚠️ Legacy remnants (`internal/handlers/health_handler.go`, `internal/models/`) predate the pattern — tolerated as glue, but new code must not be added there.

## Related

* [[ADR-BE-001 — Error Handling Sentinel Errors]] (error flow across these layers)
* [[ADR-BE-003 — Data Access pgxpool No ORM]], [[ADR-BE-009 — Testing testcontainers testify]]
* `.planning/codebase/ARCHITECTURE.md`
