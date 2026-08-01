#!/usr/bin/env node
/**
 * scripts/verify-claims.mjs
 *
 * R006 claim-to-vault gate for the Hourglass README (M001/S04). Exits 0 only
 * when every contract check passes. Zero-dependency plain Node; CI-ready.
 *
 * Contract (docs/README-claim-trace.md is the machine-check contract — a
 * claim in README.md without a trace row, or a trace row whose vault source
 * does not exist, is a defect):
 *
 *   1. Claim table: exactly 43 rows, 4 content cells per row
 *   2. Flag column: every cell is exactly 'v0.1' or 'vision' — 22 v0.1 + 21
 *      vision rows (vision rows are claims that must not be misread as
 *      shipping today)
 *   3. Vault source existence: all 15 backticked hourglass-vault/... sources
 *      named in the trace exist on disk, and the claim table references only
 *      sources covered by the existence list
 *   4. README overpromise anchors: the What-is-coming block marks V-features
 *      as 'not in v0.1'; the Roadmap marks the vision path as 'direction, not
 *      current scope'
 *   5. F-doc Implemented-marker guard: every v0.1 claim's F-doc (F05-F13)
 *      must carry an Implemented marker (US-row '✅ Implemented' or a
 *      '**Status:** Implemented' line) — except F10, the documented carve-out
 *      (activities ship big-bang pre-deploy with the P-007 migration)
 *   6. Negative-path proof: --self-test injects a bad flag, a missing vault
 *      file, removed anchors, a stripped F-doc marker, and malformed tables
 *      and asserts each produces named FAILs (exit path is failures.length
 *      > 0 -> exit 1); --trace/--readme overrides allow external mutation
 *      testing without touching the tracked inputs
 *
 * Run: node scripts/verify-claims.mjs             # normal gate
 *      node scripts/verify-claims.mjs --self-test # negative-path proof
 *      node scripts/verify-claims.mjs --trace=/tmp/bad.md --readme=... # mutation demo
 */

import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

// ---------------------------------------------------------------------------
// Contract constants
// ---------------------------------------------------------------------------
const EXPECTED_ROWS = 43; // 22 v0.1 + 21 vision
const EXPECTED_V0_1 = 22;
const EXPECTED_VISION = 21;
const EXPECTED_EXISTENCE_SOURCES = 15;
// Documented carve-out (docs/README-claim-trace.md "Mapping note on F10"):
// activities and working groups are v0.1 structure, but F10 ships big-bang
// pre-deploy with the P-007 migration, so it carries 'In progress' today.
const F10_CARVE_OUT = 'F10-Activities.md';

// ---------------------------------------------------------------------------
// CLI overrides (injectability for negative-path proof; defaults are canonical)
// ---------------------------------------------------------------------------
function argValue(flag, fallback) {
  const hit = process.argv.find((a) => a.startsWith(`${flag}=`));
  return hit ? hit.slice(flag.length + 1) : fallback;
}
const tracePath = resolve(root, argValue('--trace', 'docs/README-claim-trace.md'));
const readmePath = resolve(root, argValue('--readme', 'README.md'));
const selfTest = process.argv.includes('--self-test');

// ---------------------------------------------------------------------------
// Shared reporting helpers (main-mode and self-test-mode both report through
// the same check()/summarize() pair so exit-code semantics stay identical)
// ---------------------------------------------------------------------------
const failures = [];
const results = [];

function check(name, ok, detail = '') {
  results.push({ name, ok });
  if (!ok) failures.push(`${name}${detail ? ` — ${detail}` : ''}`);
}

function loadFile(path, label) {
  try {
    return readFileSync(path, 'utf8');
  } catch (err) {
    check(`${label} readable`, false, err.code || err.message);
    return null;
  }
}

// ---------------------------------------------------------------------------
// Pure evaluation — inputs are injected so --self-test can prove the negative
// paths without touching the tracked README/trace/vault files.
// ---------------------------------------------------------------------------
function evaluate({ traceText, readmeText, vaultExists, readVault }) {
  const f = [];
  const r = [];
  const ck = (name, ok, detail = '') => {
    r.push({ name, ok });
    if (!ok) f.push(`${name}${detail ? ` — ${detail}` : ''}`);
  };

  const traceStart = traceText.indexOf('## Claim table');
  const traceEnd = traceText.indexOf('## Source file existence check');
  ck(
    'claim table section located',
    traceStart !== -1 && traceEnd !== -1 && traceEnd > traceStart
  );
  if (traceStart === -1 || traceEnd === -1 || traceEnd <= traceStart) {
    return { failures: f, results: r };
  }

  // -------------------------------------------------------------------------
  // 1. Claim-table integrity: 43 rows, 4 cells each
  // -------------------------------------------------------------------------
  const claimSection = traceText.slice(traceStart, traceEnd);
  const linesBefore = traceText.slice(0, traceStart).split('\n').length - 1;
  const rows = [];
  claimSection.split('\n').forEach((line, i) => {
    const t = line.trim();
    if (!t.startsWith('|')) return;
    if (/^\|[\s\-:|]+\|$/.test(t)) return; // separator line
    const cells = t.replace(/^\|/, '').replace(/\|$/, '').split('|').map((c) => c.trim());
    if (cells.length === 4 && /^claim$/i.test(cells[0])) return; // header row
    rows.push({
      idx: rows.length + 1,
      lineNo: linesBefore + 1 + i,
      cells,
      flag: cells[3] ?? '(missing)',
    });
  });

  ck('claim table has exactly 43 rows', rows.length === EXPECTED_ROWS, `got ${rows.length}`);

  const badCellRows = rows.filter((row) => row.cells.length !== 4);
  ck(
    'every claim row has exactly 4 cells',
    badCellRows.length === 0,
    badCellRows.map((row) => `row ${row.idx} (line ${row.lineNo}): ${row.cells.length} cells`).join('; ')
  );

  const badFlagRows = rows.filter((row) => !['v0.1', 'vision'].includes(row.flag));
  ck(
    'every claim row flag is v0.1 or vision',
    badFlagRows.length === 0,
    badFlagRows.map((row) => `row ${row.idx}: flag '${row.flag}'`).join('; ')
  );

  const v01Count = rows.filter((row) => row.flag === 'v0.1').length;
  const visionCount = rows.filter((row) => row.flag === 'vision').length;
  ck('v0.1 rows = 22', v01Count === EXPECTED_V0_1, `got ${v01Count}`);
  ck('vision rows = 21', visionCount === EXPECTED_VISION, `got ${visionCount}`);

  // -------------------------------------------------------------------------
  // 2. Vault source existence: 15/15 backticked sources + coverage
  // -------------------------------------------------------------------------
  const existenceSection = traceText.slice(traceEnd);
  const existencePaths = [
    ...new Set(
      [...existenceSection.matchAll(/`(hourglass-vault\/[^`]+)`/g)].map((m) => m[1])
    ),
  ];
  ck(
    'existence-check list has 15 distinct backticked sources',
    existencePaths.length === EXPECTED_EXISTENCE_SOURCES,
    `got ${existencePaths.length}`
  );
  for (const path of existencePaths) {
    ck(`vault source exists: ${path}`, vaultExists(path));
  }

  const claimPaths = [
    ...new Set([...claimSection.matchAll(/`(hourglass-vault\/[^`]+)`/g)].map((m) => m[1])),
  ];
  const uncovered = claimPaths.filter((p) => !existencePaths.includes(p));
  ck(
    'claim-table vault paths covered by existence list',
    uncovered.length === 0,
    `not listed: ${uncovered.join(', ')}`
  );

  // -------------------------------------------------------------------------
  // 3. README overpromise anchors
  // -------------------------------------------------------------------------
  const comingIdx = readmeText.indexOf('### What is coming');
  const nextHeadingAfterComing = readmeText.indexOf('\n## ', comingIdx);
  const comingBlock =
    comingIdx === -1
      ? ''
      : nextHeadingAfterComing === -1
        ? readmeText.slice(comingIdx)
        : readmeText.slice(comingIdx, nextHeadingAfterComing);
  ck(
    'What-is-coming block marks V-features as not in v0.1',
    /not in v0\.1/.test(comingBlock),
    comingIdx === -1 ? "'### What is coming' not found" : "anchor phrase 'not in v0.1' missing from block"
  );

  const roadIdx = readmeText.indexOf('## Roadmap');
  const nextHeadingAfterRoad = readmeText.indexOf('\n## ', roadIdx + 1);
  const roadBlock =
    roadIdx === -1
      ? ''
      : nextHeadingAfterRoad === -1
        ? readmeText.slice(roadIdx)
        : readmeText.slice(roadIdx, nextHeadingAfterRoad);
  ck(
    'Roadmap marks vision path as direction, not current scope',
    /direction, not current scope/.test(roadBlock),
    roadIdx === -1 ? "'## Roadmap' not found" : "anchor phrase 'direction, not current scope' missing from roadmap"
  );

  // -------------------------------------------------------------------------
  // 4. F-doc Implemented-marker guard (F10 carve-out applies)
  // -------------------------------------------------------------------------
  function isImplemented(ref) {
    try {
      const content = readVault(ref);
      return content.includes('✅ Implemented') || /-\s*\*\*Status:\*\*\s*Implemented/.test(content);
    } catch {
      return false; // unreadable — existence check above already flags it
    }
  }

  // F-docs referenced by v0.1 claims only (vision rows may cite ADRs/VISION)
  const fDocRefs = [
    ...new Set(
      rows
        .filter((row) => row.flag === 'v0.1')
        .flatMap((row) => [
          ...row.cells[1].matchAll(/hourglass-vault\/01-Features\/F\d\d-[^`]+\.md/g),
        ].map((m) => m[0]))
    ),
  ];

  for (const ref of fDocRefs) {
    const base = ref.split('/').pop();
    const carveOut = base === F10_CARVE_OUT;
    ck(
      `F-doc Implemented marker: ${base}`,
      carveOut || isImplemented(ref),
      carveOut ? 'F10 carve-out exempts this file' : 'no Implemented marker found'
    );
  }

  const missingOther = fDocRefs.filter(
    (ref) => !isImplemented(ref) && ref.split('/').pop() !== F10_CARVE_OUT
  );
  ck(
    'F-doc Implemented markers present (F10 carve-out applied)',
    missingOther.length === 0,
    missingOther.length ? `missing: ${missingOther.map((p) => p.split('/').pop()).join(', ')}` : ''
  );

  let f10Documented = false;
  try {
    const f10 = readVault(`hourglass-vault/01-Features/${F10_CARVE_OUT}`);
    f10Documented = /-\s*\*\*Status:\*\*\s*In progress \(.*P-007/.test(f10);
  } catch {
    // unreadable — existence check above already flags it
  }
  ck('F10 carve-out documented as In progress (P-007)', f10Documented);

  return { failures: f, results: r };
}

// ---------------------------------------------------------------------------
// Negative-path proof (--self-test)
// ---------------------------------------------------------------------------
function runSelfTest(traceText, readmeText) {
  const vaultExists = (p) => existsSync(resolve(root, p));
  const readVault = (p) => readFileSync(resolve(root, p), 'utf8');
  const f05Path = 'hourglass-vault/01-Features/F05-Org-Bootstrap.md';
  const f05Content = readVault(f05Path);

  const base = evaluate({ traceText, readmeText, vaultExists, readVault });
  check('self-test: base inputs produce zero failures', base.failures.length === 0, base.failures.join('; '));

  const cases = [
    {
      name: 'injected bad flag yields named FAIL + exit 1',
      mutateTrace: (t) => t.replace('| vision |', '| future |'),
      expect: 'future',
    },
    {
      name: 'injected missing vault file yields named FAIL + exit 1',
      mutateTrace: (t) => t.replace('hourglass-vault/VISION.md', 'hourglass-vault/NOPE.md'),
      expect: 'NOPE.md',
    },
    {
      name: 'removed What-is-coming anchor yields named FAIL + exit 1',
      mutateReadme: (m) => m.replace('not in v0.1', 'not included yet'),
      expect: 'What-is-coming',
    },
    {
      name: 'removed Roadmap anchor yields named FAIL + exit 1',
      mutateReadme: (m) => m.replace('direction, not current scope', 'future direction'),
      expect: 'Roadmap',
    },
    {
      name: 'stripped F05 Implemented marker yields named FAIL + exit 1',
      readVaultOverride: (p) => (p === f05Path ? f05Content.replace(/✅ Implemented/g, '') : readVault(p)),
      expect: 'F05-Org-Bootstrap.md',
    },
    {
      name: 'removed claim row (42 rows) yields named FAIL + exit 1',
      mutateTrace: (t) =>
        t
          .split('\n')
          .filter((l) => !l.startsWith('| Roadmap v0.1:'))
          .join('\n'),
      expect: '43',
    },
    {
      name: '3-cell claim row yields named FAIL + exit 1',
      mutateTrace: (t) => t.replace('| vision |', ''),
      expect: 'cells',
    },
    {
      name: 'missing claim table section yields named FAIL + exit 1',
      mutateTrace: (t) => t.replace('## Claim table', '## Claims'),
      expect: 'claim table section',
    },
  ];

  for (const c of cases) {
    let trace = traceText;
    let readme = readmeText;
    if (c.mutateTrace) trace = c.mutateTrace(trace);
    if (c.mutateReadme) readme = c.mutateReadme(readme);
    const rv = c.readVaultOverride ? c.readVaultOverride : readVault;
    const res = evaluate({ traceText: trace, readmeText: readme, vaultExists, readVault: rv });
    const joined = res.failures.join(' | ');
    const hit = res.failures.length > 0 && joined.includes(c.expect);
    check(
      `self-test: ${c.name}`,
      hit,
      res.failures.length === 0
        ? `expected a named FAIL containing '${c.expect}', got zero failures`
        : `expected '${c.expect}' in failures, got: ${joined}`
    );
  }
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------
const traceText = loadFile(tracePath, tracePath);
const readmeText = loadFile(readmePath, readmePath);

if (traceText === null || readmeText === null) {
  for (const { name, ok } of results) console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`);
  process.exit(failures.length > 0 ? 1 : 0);
}

if (selfTest) {
  runSelfTest(traceText, readmeText);
  for (const { name, ok } of results) console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`);
  if (failures.length > 0) {
    console.error(`\nverify-claims --self-test: ${failures.length} case(s) failed:`);
    for (const f of failures) console.error(`  - ${f}`);
    process.exit(1);
  }
  console.log(
    `\nverify-claims --self-test: all ${results.length} cases passed (negative paths proven; exit-1 path is failures.length > 0 -> exit 1)`
  );
  process.exit(0);
}

const vaultExists = (p) => existsSync(resolve(root, p));
const readVault = (p) => readFileSync(resolve(root, p), 'utf8');
const gate = evaluate({ traceText, readmeText, vaultExists, readVault });

for (const { name, ok } of gate.results) console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`);
if (gate.failures.length > 0) {
  console.error(`\nverify-claims: ${gate.failures.length} check(s) failed:`);
  for (const f of gate.failures) console.error(`  - ${f}`);
  process.exit(1);
}
console.log(`\nverify-claims: all ${gate.results.length} checks passed`);
process.exit(0);
