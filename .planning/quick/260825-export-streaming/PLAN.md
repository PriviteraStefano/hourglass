---
quick_id: 260825-export-streaming
description: Bound export memory — range cap, CSV/XLSX streaming, drop in-memory sort (CONCERNS #7)
date: 2026-08-25
status: complete
---

# Quick Task 260825-export-streaming

## Plan

Address CONCERNS.md #7 "Export endpoints buffer all rows in memory".

Tasks:
1. `internal/adapters/primary/http/export.go`: add `maxExportRange` (731 days) enforced in
   `parseExportRange`; handlers return 400 on an over-wide or inverted range. This bounds memory
   for every export format regardless of streaming.
2. `writeCSV` rewritten to stream rows via `csv.Writer` one at a time (no intermediate `[]csvRow`
   buffer); converters become single-row (`timeSheetRow`/`expenseRow`/`combinedRow`).
3. `writeXLSX` rewritten to use excelize `StreamWriter` (rows written one at a time, no full
   in-memory workbook model). `xlsxSheet` now carries `[]ports.ExportRow` + a converter.
4. `internal/core/services/export/export.go`: replace the O(n log n) `sort.Slice` in `Combined`
   with `mergeExportRowsDesc`, a two-pointer merge of the two date-ascending slices the repository
   already returns. Preserves nil when both inputs are nil.

## Verify

- `go build ./...` passes
- `go vet ./internal/adapters/primary/http ./internal/core/services/export` passes
- `go test ./internal/core/services/export/... ./internal/adapters/primary/http/...` passes
