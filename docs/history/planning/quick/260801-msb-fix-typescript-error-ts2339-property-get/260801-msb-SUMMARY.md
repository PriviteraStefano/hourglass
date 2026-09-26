---
quick_id: 260801-msb
status: complete
date: 2026-08-01
requirements: [MSB-260801-TS2339-HEADERS-NEVER]
---

# Quick Task 260801-msb — Summary

**Task:** Fix TS2339 "Property 'get' does not exist on type 'never'" in `web/src/lib/__tests__/api.test.ts` — restore `npx tsc -b` green without changing test semantics.

## What was done

In the `it("sets Content-Type: application/json header", ...)` block (formerly lines 74–85), replaced the bare `let capturedHeaders: Headers | null = null` closure-capture pattern with an **object-property capture**:

* `let capturedHeaders: Headers | null = null;` → `const captured: { headers: Headers | null } = { headers: null };`
* `capturedHeaders = request.headers;` → `captured.headers = request.headers;` (inside the MSW resolver)
* `expect(capturedHeaders?.get("Content-Type")).toBe("application/json");` → `expect(captured.headers?.get("Content-Type")).toBe("application/json");`

Rationale (empirically verified in the plan's evidence table): under TS 7 (Go port) control-flow analysis, any `let x: T | null = null` assigned inside a nested closure and read after an `await` narrows to `never`. Property writes (`captured.headers = ...`) do not trigger the variable-CFA bug, so the post-`await` read type-checks as `Headers | null`. Keeps the DOM `Headers` type — no `!`, `any`, or casts.

## Files modified

* `web/src/lib/__tests__/api.test.ts` — one test body, 3 insertions / 3 deletions (diff confined to the Content-Type header block; the other five tests untouched).

## Verification

* `cd web && npx tsc -b` → **exit 0, zero TypeScript errors** (no TS2339 anywhere).
* `npx vitest run src/lib/__tests__/api.test.ts` → **6/6 tests passed**, including the Content-Type header assertion.
* `git diff --stat web/src/lib/__tests__/api.test.ts` → `3 insertions(+), 3 deletions(-)` — diff limited to the one test block.
* Test semantics unweakened: still asserts `request.headers.get("Content-Type") === "application/json"` against the real captured request headers.

## Commit

* `b557a8e` — `fix(quick-260801-msb): fix TS2339 never type in api.test.ts header capture`

## Notes / follow-ups

* No deviations from plan — executed exactly as written (single `auto` task, applied fresh to the original broken state).
* Pre-existing unrelated WIP in six other files (`web/src/routes/__root.tsx`, auth/bootstrap, invite, password-reset, org-hierarchy `unit-detail-panel.tsx`) was left untouched and not staged, per scope boundary.
* `web/src/lib/api.ts` intentionally not modified — client behavior was already correct.
