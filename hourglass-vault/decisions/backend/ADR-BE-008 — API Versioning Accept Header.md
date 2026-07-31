# ADR-BE-008 — API Versioning via Accept Header

---
tags: ["adr", "backend", "api", "versioning"]

---

# ADR-BE-008 — API Versioning via Accept Header

**Status:** Accepted
**Date:** 2026-07-28
**Code:** `internal/middleware/version.go`

## Context

The API will eventually need breaking changes without killing existing clients. Versioning strategy needed to be chosen before v0.1 goes online so the first public contract starts on a deliberate scheme.

## Decision

* **Mechanism:** Accept-header media-type versioning: `Accept: application/vnd.hourglass+json version=1`.
* **Middleware** (`APIVersion`) parses the header and records the requested version in context; default/absent = latest stable.
* **Current state:** only **v1** exists. The middleware is in place so v2 can be introduced by routing on the parsed version, not by retrofitting URL prefixes.
* **Not URL-versioned.** Paths stay clean (`/time-entries`, not `/v1/time-entries`); version is a representation concern, carried in the header.

## Consequences

* First public clients are born on v1 with a defined upgrade path.
* Breaking changes later mean negotiating v2 via header — old clients keep v1.
* ⚠️ No actual version negotiation logic exists yet beyond parsing; when v2 arrives, routing-on-version is new work this ADR sanctions but does not implement.

## Related

* [[ADR-BE-006 — Middleware Composition]] (middleware order)
* [[S04-API-Contracts]] (the v1 contract this versions)
