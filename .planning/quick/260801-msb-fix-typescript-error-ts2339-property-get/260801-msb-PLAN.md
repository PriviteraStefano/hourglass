---
slug: 260801-msb-fix-typescript-error-ts2339-property-get
created: 2026-08-01
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: [web/src/lib/__tests__/api.test.ts]
autonomous: true
requirements: [MSB-260801-TS2339-HEADERS-NEVER]
must_haves:
  truths:
    - "`npx tsc -b` in web/ completes with zero errors"
    - "The Content-Type header assertion test still passes at runtime (vitest)"
  artifacts:
    - path: "web/src/lib/__tests__/api.test.ts"
      provides: "TS2339-free header-capture test"
      contains: "captured.headers"
  key_links:
    - from: "msw resolver ({ request }) =>"
      to: "outer capture object"
      via: "property assignment inside closure (object-property capture)"
      pattern: "captured\\.headers = request\\.headers"
---

# Fix TS2339: Property 'get' does not exist on type 'never' in api.test.ts:84

## Objective

**Problem:** `web/src/lib/__tests__/api.test.ts` line 84 fails `npx tsc -b` (web/) with
`TS2339: Property 'get' does not exist on type 'never'` in the "sets Content-Type:
application/json header" test.

**Root cause (empirically verified — do NOT re-investigate, see Evidence below):** This is
**NOT a type-annotation mismatch**. MSW's resolver receives `request: Request` where the
global `Request` (under `msw/node` + `@types/node` 24.13.3) is undici's; `request.headers`
is undici `Headers`, which is structurally assignable to the DOM `Headers` used in the
variable annotation (a cast compiles without error). The `never` is produced by TS 7 (Go
port) **control-flow analysis**: any `let x: T | null = null` that is assigned inside a
nested closure and read after an `await` narrows to `never` (the intersection of the `null`
init and the closure-assigned type is empty). Every variant that keeps this
closure-assigned `let`-with-null pattern fails identically — including undici-type
annotations, `Request["headers"]` indexing, `as Headers` casts, and `new
Headers(request.headers)`.

**Fix (verified):** replace the bare `let capturedHeaders: Headers | null = null` with an
**object-property capture** — `const captured: { headers: Headers | null } = { headers:
null }` and assign `captured.headers = request.headers`. Property writes do not trigger the
variable-CFA bug; the post-`await` read type-checks as `Headers | null`. This is a 3-line
minimal diff that keeps the DOM `Headers` type, needs no `!`, `any`, or casts.

**Purpose:** Restore `npx tsc -b` green in web/ without changing test semantics.

**Output:** Fixed `web/src/lib/__tests__/api.test.ts` (one test's body, ~3 lines changed).

## Evidence (investigation results — verified, bind the executor)

Probed with scratch files type-checked by `npx tsc -b` in web/:

| Variant | Result |
|---------|--------|
| `let capturedHeaders: Headers | null = null` (baseline, DOM Headers) | FAILS — never |
| `let capturedHeaders: UndiciHeaders | null` (undici-types import) | FAILS — never |
| `let capturedHeaders: Request["headers"] | null` | FAILS — never |
| `capturedHeaders = request.headers as Headers` (cast) | FAILS — never |
| `capturedHeaders = new Headers(request.headers)` | FAILS — never |
| `let contentType: string | null` (capture header value only) | FAILS — never |
| `let capturedRequest: Request | null` (capture request itself) | FAILS — never |
| `let getHeaders: (() => Headers) | null` (function capture) | FAILS — never |
| `let capturedHeaders!: Headers` (definite assignment, no null) | PASSES |
| `const captured: { headers: Headers | null } = { headers: null }` (object property) | PASSES |
| read `.get()` inside the closure immediately after assignment | PASSES |

The object-property capture was additionally verified against the **real** test file:
`npx tsc -b` clean (0 errors) and `npx vitest run src/lib/__tests__/api.test.ts` → 6/6
passed. The file is currently back at its original (broken) state; apply the fix fresh.

<context>
@web/src/lib/__tests__/api.test.ts
@web/src/lib/api.ts
@.planning/STATE.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Switch header capture to object-property pattern</name>
  <files>web/src/lib/__tests__/api.test.ts</files>
  <action>In `web/src/lib/__tests__/api.test.ts`, inside the `it("sets Content-Type: application/json header", ...)` block (currently lines 74-85), apply the object-property capture fix. Exact edit — three substitutions in that one block only:

  1. Line 75 — replace the declaration
     `let capturedHeaders: Headers | null = null;`
     with
     `const captured: { headers: Headers | null } = { headers: null };`
  2. Line 78 — inside the resolver callback, replace
     `capturedHeaders = request.headers;`
     with
     `captured.headers = request.headers;`
  3. Line 84 — replace
     `expect(capturedHeaders?.get("Content-Type")).toBe("application/json");`
     with
     `expect(captured.headers?.get("Content-Type")).toBe("application/json");`

  Do NOT try any other fix shape: the evidence table above proves annotation changes, casts, `new Headers(...)`, string-value capture, and request-object capture all still produce `never` under TS 7 CFA. Do NOT touch the other five tests in this file, and do NOT modify `web/src/lib/api.ts` — the client behavior is correct; only the test's capture pattern is at fault. Do NOT add `let capturedHeaders!: Headers` — it compiles, but `!` suppresses a real read-before-write case (resolver never runs ⇒ runtime `undefined`), whereas the object-property form keeps an explicit `null` default.</action>
  <verify>
    <automated>cd web && npx tsc -b && npx vitest run src/lib/__tests__/api.test.ts</automated>
  </verify>
  <done>`npx tsc -b` in web/ exits with zero errors (no TS2339 anywhere); `vitest run src/lib/__tests__/api.test.ts` reports 6/6 passing, including the Content-Type header assertion.</done>
</task>

</tasks>

<verification>
- `npx tsc -b` (web/) — zero TypeScript errors.
- `npx vitest run src/lib/__tests__/api.test.ts` — 6 tests pass.
- `git diff --stat web/src/lib/__tests__/api.test.ts` — diff limited to the one test block (~3 changed lines).
</verification>

<success_criteria>
- The TS2339 `never` error is gone from `npx tsc -b` output.
- The Content-Type header test still asserts `request.headers.get("Content-Type") === "application/json"` against the real captured request headers (no test weakening).
- No other files modified.
</success_criteria>

<output>
Create `.planning/quick/260801-msb-fix-typescript-error-ts2339-property-get/260801-msb-SUMMARY.md` when done
</output>
