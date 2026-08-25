---
status: complete
quick_id: 260825-export-streaming
date: 2026-08-25
---

# Summary — 260825-export-streaming

**Concern:** CONCERNS.md #7 Export endpoints buffer all rows in memory.

**Changes:**
- Export range is now capped at `maxExportRange` (731 days ≈ 2 years) and validated in
  `parseExportRange`; over-wide or inverted ranges return 400. This is the primary memory bound —
  an org's full history can no longer be pulled into one export.
- CSV export streams rows directly through `csv.Writer` (removed the `[]csvRow` intermediate slice).
- XLSX export uses excelize's `StreamWriter`, so the workbook model no longer holds every cell at
  once. `xlsxSheet` now carries `[]ports.ExportRow` + a per-row converter.
- `Combined` no longer runs an in-memory `sort.Slice`; it merges the repo's already date-ordered
  slices via `mergeExportRowsDesc` (two-pointer, O(n)).

**Residual:** The service still materializes `[]ports.ExportRow` from the repository (true DB-cursor
streaming would require a streaming repository method). With the range cap that slice is bounded,
so the unbounded-memory failure mode is closed.

**Verification:** `go build ./...`, `go vet`, and both package test suites pass.
