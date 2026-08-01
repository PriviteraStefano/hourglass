#!/usr/bin/env node
/**
 * scripts/verify-readme.mjs
 *
 * Reusable README verification harness for Hourglass M001/S01 (and the S04
 * machine-check contract). Exits 0 only when every contract check passes.
 *
 * Contract checks:
 *   1. README.md total line count < 300
 *   2. The three questions appear verbatim
 *   3. 'Hourglass for adopters' and 'Hourglass for employees' headings present
 *   4. Adopters half has the three sub-blocks, incl. non-goals for chat,
 *      task boards, and payroll/HR machinery
 *   5. Employees half has the daily loop and one worked example
 *   6. No blocklisted jargon ('steering test', 'stakeholder map', 'ADR') and
 *      no links/references into hourglass-vault/
 *   7. Code fences are balanced
 *   8. '## Tech stack', '## License', '## Changelog' headings present
 *   9. docs/README-claim-trace.md exists and is non-empty
 *
 * Plain Node (node:assert-style); no dependencies. Run: node scripts/verify-readme.mjs
 */

import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const readmePath = resolve(root, 'README.md');
const claimTracePath = resolve(root, 'docs', 'README-claim-trace.md');

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
// Inputs
// ---------------------------------------------------------------------------
const readme = loadFile(readmePath, 'README.md');
const claimTrace = loadFile(claimTracePath, 'docs/README-claim-trace.md');

if (readme === null || claimTrace === null) {
  report();
}

const readmeLines = readme.split('\n');
// wc -l compatible: a trailing newline does not count as an extra line
const lineCount = readmeLines.length - (readme.endsWith('\n') ? 1 : 0);

// ---------------------------------------------------------------------------
// 1. Line count ceiling
// ---------------------------------------------------------------------------
check('README under 300 lines', lineCount < 300, `${lineCount} lines`);

// ---------------------------------------------------------------------------
// 2. Three questions verbatim
// ---------------------------------------------------------------------------
const QUESTIONS = [
  'What should I be working on?',
  'Is the work on track?',
  'What does the work cost and earn?',
];
for (const q of QUESTIONS) {
  check(`verbatim question present: "${q}"`, readme.includes(q));
}

// ---------------------------------------------------------------------------
// 3. Audience headings
// ---------------------------------------------------------------------------
check("'Hourglass for adopters' heading present", readme.includes('Hourglass for adopters'));
check("'Hourglass for employees' heading present", readme.includes('Hourglass for employees'));

// ---------------------------------------------------------------------------
// 4. Adopters half: sub-blocks + non-goals
// ---------------------------------------------------------------------------
const adoptersStart = readme.indexOf('## Hourglass for adopters');
const employeesStart = readme.indexOf('## Hourglass for employees');
const adoptersHalf =
  adoptersStart !== -1 && employeesStart !== -1
    ? readme.slice(adoptersStart, employeesStart)
    : '';

check('adopters half located', adoptersHalf.length > 0);
check("adopters 'What you get' sub-block present", adoptersHalf.includes('### What you get'));
check("adopters 'What it is not' sub-block present", adoptersHalf.includes('### What it is not'));
check("adopters 'What is coming' sub-block present", adoptersHalf.includes('### What is coming'));
check('adopters states no-chat non-goal', /\bchat\b/i.test(adoptersHalf));
check('adopters states no-task-board non-goal', /task\s*board/i.test(adoptersHalf));
check('adopters states no-payroll/HR non-goal', /payroll|HR\s*machinery/i.test(adoptersHalf));

// ---------------------------------------------------------------------------
// 5. Employees half: daily loop + worked example
// ---------------------------------------------------------------------------
const employeesHalf = employeesStart !== -1 ? readme.slice(employeesStart) : '';
check("employees 'The daily loop' present", employeesHalf.includes('The daily loop'));
check("employees 'One worked example' present", employeesHalf.includes('One worked example'));

// ---------------------------------------------------------------------------
// 6. Jargon blocklist + no vault links
// ---------------------------------------------------------------------------
const BLOCKLIST = ['steering test', 'stakeholder map', 'ADR'];
for (const term of BLOCKLIST) {
  check(`blocklisted jargon absent: "${term}"`, !readme.includes(term));
}
check('no hourglass-vault links or references', !readme.includes('hourglass-vault'));

// ---------------------------------------------------------------------------
// 7. Code fence balance
// ---------------------------------------------------------------------------
let inFence = false;
let fencePairs = 0;
for (const line of readmeLines) {
  if (/^\s*```/.test(line)) {
    if (!inFence) fencePairs += 1;
    inFence = !inFence;
  }
}
check('code fences balanced', !inFence, `${fencePairs} open/close pairs`);

// ---------------------------------------------------------------------------
// 8. Technical-half headings intact
// ---------------------------------------------------------------------------
for (const h of ['## Tech stack', '## License', '## Changelog']) {
  check(`heading present: ${h}`, readme.includes(h));
}

// ---------------------------------------------------------------------------
// 9. Claim trace exists and is non-empty
// ---------------------------------------------------------------------------
check('docs/README-claim-trace.md exists', existsSync(claimTracePath));
check('docs/README-claim-trace.md non-empty', claimTrace !== null && claimTrace.trim().length > 0);

report();

// ---------------------------------------------------------------------------
function report() {
  for (const { name, ok } of results) {
    console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`);
  }
  if (failures.length > 0) {
    console.error(`\nverify-readme: ${failures.length} check(s) failed:`);
    for (const f of failures) console.error(`  - ${f}`);
    process.exit(1);
  }
  console.log(
    `\nverify-readme: all ${results.length} checks passed (README ${lineCount} lines)`
  );
  process.exit(0);
}
