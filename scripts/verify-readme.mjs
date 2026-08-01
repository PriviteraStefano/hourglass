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
 *  10. Link target resolution (S04): every ](...) target outside code fences
 *      resolves — file targets exist (7 wiki/*.md pages, ./LICENSE,
 *      ./CHANGELOG.md), intra-doc '#anchor' targets slugify against a real
 *      heading (GitHub anchor algorithm), external http(s)/mailto: skipped.
 *  11. Structural GFM (S04): no [[ Obsidian wiki links, no § section-ref
 *      leaks, every table row column-consistent, valid ^#{1,6} ATX heading
 *      syntax.
 *
 * --readme=PATH override points the harness at a mutated copy for external
 * negative-path testing without touching the tracked README.
 *
 * Plain Node (node:assert-style); no dependencies. Run: node scripts/verify-readme.mjs
 */

import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
// --readme=PATH override: run the harness against a mutated copy for external
// negative-path proof without mutating the tracked README (S04/T02).
const readmeOverride = process.argv.find((a) => a.startsWith('--readme='));
const readmePath = readmeOverride
  ? resolve(readmeOverride.slice('--readme='.length))
  : resolve(root, 'README.md');
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

// ---------------------------------------------------------------------------
// 10. Link target resolution + structural GFM checks (S04/T02)
// ---------------------------------------------------------------------------

// Track code-fence state; only non-fence lines are scanned for links/tables/
// headings so URLs inside code blocks never register as checkable targets.
function nonFenceLines(lines) {
  const out = [];
  let inFence = false;
  for (const line of lines) {
    if (/^\s*```/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (!inFence) out.push(line);
  }
  return out;
}

// GitHub anchor slugification (cmark-gfm github-extension semantics):
// lowercase, keep word chars/dashes/spaces, spaces -> '-', drop the rest,
// trim trailing hyphens. '## Getting started' -> 'getting-started'.
function ghSlug(text) {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+$/g, '');
}

// Extract {level, text, slug} for every ATX heading in a document.
function collectHeadings(text) {
  const headings = [];
  for (const line of text.split('\n')) {
    const m = /^(#{1,6})\s+(.+)$/.exec(line);
    if (m) headings.push({ level: m[1].length, text: m[2], slug: ghSlug(m[2]) });
  }
  return headings;
}

// Extract every ](target) inline-link target (tolerates "title" suffixes).
function collectLinks(text) {
  const links = [];
  const re = /\]\(([^\s)]+)(?:\s+[^)]*)?\)/g;
  let m;
  while ((m = re.exec(text)) !== null) links.push(m[1]);
  return links;
}

// Split 'path#anchor' into {path, anchor}; intra-doc '#foo' has empty path.
function splitTarget(target) {
  const hash = target.indexOf('#');
  if (hash === -1) return { path: target, anchor: '' };
  return { path: target.slice(0, hash), anchor: target.slice(hash + 1) };
}

// Trim a GFM table row into its cell list (leading/trailing pipe optional).
function tableCells(line) {
  const s = line.trim();
  const inner = s.startsWith('|') ? s.slice(1) : s;
  return (inner.endsWith('|') ? inner.slice(0, -1) : inner)
    .split('|')
    .map((c) => c.trim());
}

// Maximal runs of consecutive | -starting lines (outside fences) whose second
// line is a well-formed GFM delimiter row; returns the raw run lines per table.
function findTables(lines) {
  const out = [];
  const nf = nonFenceLines(lines);
  let i = 0;
  while (i < nf.length) {
    if (nf[i].trim().startsWith('|')) {
      const run = [];
      while (i < nf.length && nf[i].trim().startsWith('|')) run.push(nf[i++]);
      const delimOk =
        run.length >= 2 && tableCells(run[1]).every((c) => /^:?-+:?$/.test(c));
      if (delimOk) out.push(run);
    } else {
      i += 1;
    }
  }
  return out;
}

const nonFenceBody = nonFenceLines(readmeLines).join('\n');

// 10a. Every ](...) target outside code fences resolves: file targets must
// exist on disk; '#anchor' targets must slugify against a real heading of the
// README (intra-doc) or of the linked file. External schemes are skipped — a
// batch CI gate cannot resolve them offline.
{
  const EXTERNAL = /^(https?|mailto|tel|ftp):/i;
  const broken = [];
  for (const target of collectLinks(nonFenceBody)) {
    if (EXTERNAL.test(target)) continue;
    const { path, anchor } = splitTarget(target);
    if (path && !existsSync(resolve(root, path))) {
      broken.push(target);
      continue;
    }
    if (anchor) {
      const src =
        path && existsSync(resolve(root, path))
          ? (loadFile(resolve(root, path), path) ?? '')
          : readme;
      const ok =
        src !== '' && collectHeadings(src).some((h) => h.slug === anchor);
      if (!ok) broken.push(target);
    }
  }
  check(
    'every ](...) target resolves (file exists or anchor slugifies)',
    broken.length === 0,
    broken.length ? `broken targets: ${broken.join(', ')}` : ''
  );
}

// 10b. Named category checks: the 7 wiki pages + ./LICENSE + ./CHANGELOG.md +
// the intra-doc #getting-started anchor (must link AND slugify to the heading).
{
  const WIKI_LINKS = [
    'wiki/Home.md',
    'wiki/Getting-Started.md',
    'wiki/Adopters.md',
    'wiki/Employees.md',
    'wiki/Vision.md',
    'wiki/FAQ.md',
    'wiki/Developer.md',
  ];
  const missing = WIKI_LINKS.filter(
    (w) => !readme.includes(`](${w})`) || !existsSync(resolve(root, w))
  );
  check(
    'all 7 wiki/*.md links present and targets exist',
    missing.length === 0,
    missing.length ? `missing/broken: ${missing.join(', ')}` : ''
  );
  check(
    'README links ./LICENSE and it exists',
    readme.includes('](./LICENSE)') && existsSync(resolve(root, 'LICENSE'))
  );
  check(
    'README links ./CHANGELOG.md and it exists',
    readme.includes('](./CHANGELOG.md)') && existsSync(resolve(root, 'CHANGELOG.md'))
  );
  const gs = collectHeadings(readme).some((h) => h.slug === 'getting-started');
  check(
    "intra-doc anchor #getting-started resolves to '## Getting started'",
    gs && /\]\(#getting-started\)/.test(nonFenceBody),
    gs ? '' : "no heading slugs to 'getting-started'"
  );
}

// 10c. No Obsidian wiki links / section-ref leaks (checked outside fences).
check('no [[ Obsidian wiki links', !nonFenceBody.includes('[['));
check('no § section-ref leaks', !nonFenceBody.includes('§'));

// 10d. Every GFM table row must match its header column count (delimiter row
// included). Details name the table ordinal and the offending row.
{
  const problems = [];
  for (const [ti, run] of findTables(readmeLines).entries()) {
    const cols = tableCells(run[0]).length;
    const bad = [];
    run.slice(1).forEach((row, ri) => {
      const got = tableCells(row).length;
      if (got !== cols) bad.push(`row ${ri + 2}: expected ${cols} cells, got ${got} (${row.trim()})`);
    });
    if (bad.length) problems.push(`table ${ti + 1}: ${bad.join('; ')}`);
  }
  check('tables: every row column-consistent', problems.length === 0, problems.join('; '));
}

// 10e. ATX heading syntax: lines starting with '#' must be ^#{1,6} + space
// (or a bare 1-6 hash run). 7+ hashes or missing space = failed heading.
{
  const bad = [];
  for (const line of nonFenceLines(readmeLines)) {
    const m = /^#+/.exec(line);
    if (!m) continue;
    if (m[0].length > 6) {
      bad.push(`"${line}" (${m[0].length} hashes)`);
    } else if (line.length > m[0].length && !/^#+\s/.test(line)) {
      bad.push(`"${line}" (missing space after #)`);
    }
  }
  check('headings: valid ATX syntax (^#{1,6} + space)', bad.length === 0, bad.join('; '));
}

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
