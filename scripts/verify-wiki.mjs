#!/usr/bin/env node
/**
 * scripts/verify-wiki.mjs
 *
 * Wiki page-set verification harness for Hourglass M001/S03 (and the S04
 * machine-check contract). Exits 0 only when every contract check passes.
 *
 * The wiki is a 7-page flat GFM set in repo-root wiki/: Adopters.md,
 * Developer.md, Employees.md, FAQ.md, Getting-Started.md, Home.md, Vision.md.
 *
 * Contract checks:
 *   1. The wiki directory holds exactly the 7-page contract set
 *   2. Per page (all 7): non-empty, footered '_Source: ... YYYY-MM-DD'
 *      (S03/D004 convention), zero 'hourglass-vault' strings, balanced code
 *      fences
 *   3. Jargon blocklist ('steering test', 'stakeholder map', 'ADR', standalone
 *      'edges') applies to the 6 customer pages only — wiki/Developer.md is
 *      the technical reference and is exempt per R005 (same exemption shape as
 *      verify-readme.mjs section 6)
 *   4. Cross-page links (D003): every '](Xxx.md)' target exists in wiki/
 *      (https:// blob URLs and ../README.md are excluded — Developer.md links
 *      back to the repo README that way); Home.md links to all 6 other pages;
 *      each customer page links back to Home.md (Developer.md exempt per
 *      MEM009 parity semantics — its content is not touched by the wiki seed)
 *
 * Plain Node, no dependencies (mirrors scripts/verify-readme.mjs and
 * scripts/verify-dev-parity.mjs).
 * Run: node scripts/verify-wiki.mjs
 */

import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const wikiDir = resolve(root, 'wiki');

// The 7-page contract set (S03 slice goal: "a 7-page wiki set in the flat
// wiki/ namespace"). Sorted for set comparison.
const EXPECTED_PAGES = [
  'Adopters.md',
  'Developer.md',
  'Employees.md',
  'FAQ.md',
  'Getting-Started.md',
  'Home.md',
  'Vision.md',
];

// wiki/Developer.md is the S02 technical reference: the jargon blocklist
// applies to the 6 S03 customer pages only (Developer.md exempt, R005). The
// back-to-Home link requirement applies to the 5 deep pages only — Home.md is
// the index, and Developer.md is exempt (MEM009: the wiki seed never modifies
// it, and it navigates back via ../README.md instead).
const CUSTOMER_PAGES = EXPECTED_PAGES.filter((p) => p !== 'Developer.md');
const DEEP_PAGES = EXPECTED_PAGES.filter((p) => p !== 'Developer.md' && p !== 'Home.md');

const JARGON_TERMS = ['steering test', 'stakeholder map', 'ADR'];

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

function countLines(text) {
  const lines = text.split('\n');
  return lines.length - (text.endsWith('\n') ? 1 : 0);
}

function fencesBalanced(text) {
  let inFence = false;
  for (const line of text.split('\n')) {
    if (/^\s*```/.test(line)) inFence = !inFence;
  }
  return !inFence;
}

// ---------------------------------------------------------------------------
// Inputs
// ---------------------------------------------------------------------------
let pageFiles = [];
try {
  pageFiles = readdirSync(wikiDir).filter((f) => f.endsWith('.md')).sort();
} catch (err) {
  check('wiki directory readable', false, err.code || err.message);
  report();
}

// ---------------------------------------------------------------------------
// 1. Page set contract
// ---------------------------------------------------------------------------
{
  const missing = EXPECTED_PAGES.filter((p) => !pageFiles.includes(p));
  const extra = pageFiles.filter((p) => !EXPECTED_PAGES.includes(p));
  const ok = missing.length === 0 && extra.length === 0;
  check(
    'wiki holds exactly the 7-page contract set',
    ok,
    [missing.length ? `missing: ${missing.join(', ')}` : '', extra.length ? `unexpected: ${extra.join(', ')}` : '']
      .filter(Boolean)
      .join('; ')
  );
}

// ---------------------------------------------------------------------------
// 2. Per-page checks (all 7 pages)
// ---------------------------------------------------------------------------
const pages = {};
for (const page of EXPECTED_PAGES) {
  pages[page] = loadFile(resolve(wikiDir, page), `wiki/${page}`);
}

for (const page of EXPECTED_PAGES) {
  const content = pages[page];
  if (content === null) continue;
  check(`wiki/${page} non-empty`, content.trim().length > 0, `${countLines(content)} lines`);
  check(
    `wiki/${page} footered with source + date (S03 convention)`,
    /_Source:.*\d{4}-\d{2}-\d{2}/.test(content)
  );
  check(`wiki/${page} has no hourglass-vault references`, !content.includes('hourglass-vault'));
  check(`wiki/${page} code fences balanced`, fencesBalanced(content));
}

// ---------------------------------------------------------------------------
// 3. Jargon blocklist — 6 customer pages only (Developer.md exempt, R005)
// ---------------------------------------------------------------------------
for (const page of CUSTOMER_PAGES) {
  const content = pages[page] ?? '';
  for (const term of JARGON_TERMS) {
    check(`wiki/${page} jargon absent: "${term}"`, !content.includes(term));
  }
  check(`wiki/${page} jargon absent: standalone "edges"`, !/\bedges\b/i.test(content));
}

// ---------------------------------------------------------------------------
// 4. Cross-page links (D003)
// ---------------------------------------------------------------------------
// Collect every '](target)' where the target is a .md link. https:// blob
// URLs and ../README.md are excluded (Developer.md links to the repo README
// that way; the wiki is a flat namespace with no parent traversal).
function collectMdLinks(content) {
  const links = [];
  const re = /\]\(([^)\s]+\.md)\)/g;
  let m;
  while ((m = re.exec(content)) !== null) {
    let target = m[1];
    const hash = target.indexOf('#');
    if (hash !== -1) target = target.slice(0, hash);
    links.push(target);
  }
  return links;
}

for (const page of EXPECTED_PAGES) {
  const content = pages[page] ?? '';
  const broken = collectMdLinks(content)
    .filter((target) => !target.startsWith('http') && target !== '../README.md')
    .filter((target) => !existsSync(resolve(wikiDir, target)));
  check(
    `wiki/${page} internal .md links resolve`,
    broken.length === 0,
    broken.length ? `broken targets: ${broken.join(', ')}` : ''
  );
}

// Home.md must index all 6 other pages.
{
  const home = pages['Home.md'] ?? '';
  const missing = EXPECTED_PAGES.filter((p) => p !== 'Home.md' && !home.includes(`](${p})`));
  check(
    'wiki/Home.md links to all 6 other pages',
    missing.length === 0,
    missing.length ? `missing links: ${missing.join(', ')}` : ''
  );
}

// Each deep page links back to Home.md. Developer.md is exempt (MEM009: the
// S03 seed does not modify Developer.md, and it predates the wiki seed's
// back-navigation convention — it links back via ../README.md instead).
for (const page of DEEP_PAGES) {
  const content = pages[page] ?? '';
  check(`wiki/${page} links back to Home.md`, content.includes('](Home.md)'));
}

report();

// ---------------------------------------------------------------------------
function report() {
  for (const { name, ok } of results) {
    console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`);
  }
  if (failures.length > 0) {
    console.error(`\nverify-wiki: ${failures.length} check(s) failed:`);
    for (const f of failures) console.error(`  - ${f}`);
    process.exit(1);
  }
  console.log(`\nverify-wiki: all ${results.length} checks passed (${pageFiles.length} wiki pages)`);
  process.exit(0);
}
