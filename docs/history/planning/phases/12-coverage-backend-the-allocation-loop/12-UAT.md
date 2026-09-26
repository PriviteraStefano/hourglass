---
status: testing
phase: 12-coverage-backend-the-allocation-loop
source: [12-VERIFICATION.md]
started: 2026-08-08T10:54:04Z
updated: 2026-08-08T10:54:04Z
---

## Current Test

number: 1
name: Manual smoke: two PUTs of ≥2-row allocation sets for two different entries against a live DB (CR-01 fix)
expected: |
  Both replace-sets commit; every stored allocation row has a unique non-nil id; the table-wide PK never collides
awaiting: user response

## Tests

### 1. Manual smoke: two PUTs of ≥2-row allocation sets for two different entries against a live DB (CR-01 fix)
expected: Both replace-sets commit; every stored allocation row has a unique non-nil id; the table-wide PK never collides
result: [pending]

### 2. Manual concurrent-close test: two goroutines POST /coverage/close for the same period (WR-03 fix)
expected: Exactly one 201 and one 409; never two snapshots for one period
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
