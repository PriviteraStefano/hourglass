#!/usr/bin/env node
/**
 * scripts/verify-dev-parity.mjs
 *
 * R004 content-parity guard for Hourglass M001/S02. The README technical half
 * was reshaped (287 -> 208 lines) in T02, relocating the heavy developer
 * reference onto wiki/Developer.md (T01). This script verifies nothing was
 * lost in the move: every env var, make target, testing command, domain-model
 * term, stack row, structure entry, prerequisite, and quickstart command from
 * the OLD technical half must still exist somewhere the reader can find it.
 *
 * Inventory is a STATIC snapshot of the pre-T02 README technical half
 * (commit cae317c, "## For developers" through "Domain model reference"),
 * grouped by category. Each term is checked against:
 *
 *   - wiki-required categories (env-vars, make-targets, testing,
 *     domain-model): the reference moved to wiki/Developer.md, so the term
 *     MUST be present there (README presence alone does not prove the
 *     relocated reference survived).
 *   - either-or categories (stack-rows, structure, prerequisites,
 *     quickstart): the term may live in README.md OR wiki/Developer.md,
 *     since the README kept the stack table, condensed tree, and quickstart.
 *
 * Known intentional exclusions (regenerated from code in T01, not loss):
 *   - cmd/schema was in the old README tree but does not exist in the repo
 *     (only cmd/server and cmd/migrate); the wiki tree omits it.
 *   - Old README "Testing" said eslint; the repo uses oxlint (web/package.json)
 *     and the wiki documents oxlint. The command (`bun run lint`) is checked.
 *   - Expense categories in the old README (4) were stale; the wiki lists the
 *     real 9. The 4 old terms are checked; the 5 added ones are extras.
 *
 * Plain Node, no dependencies (mirrors scripts/verify-readme.mjs).
 * Run: node scripts/verify-dev-parity.mjs
 */

import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const readmePath = resolve(root, 'README.md');
const devPath = resolve(root, 'wiki', 'Developer.md');

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
const dev = loadFile(devPath, 'wiki/Developer.md');

if (readme === null || dev === null) {
  report();
}

const readmeLines = readme.split('\n');
const readmeLineCount = readmeLines.length - (readme.endsWith('\n') ? 1 : 0);
const devLines = dev.split('\n');
const devLineCount = devLines.length - (dev.endsWith('\n') ? 1 : 0);

// ---------------------------------------------------------------------------
// Structural guards
// ---------------------------------------------------------------------------
check('wiki/Developer.md exists', existsSync(devPath));
check('wiki/Developer.md non-empty', dev.trim().length > 0, `${devLineCount} lines`);
check('wiki/Developer.md has no hourglass-vault references', !dev.includes('hourglass-vault'));
check(
  'wiki/Developer.md footered with source + date (S03 convention)',
  /_Source:.*\d{4}-\d{2}-\d{2}/.test(dev)
);

// ---------------------------------------------------------------------------
// Static inventory of the old README technical half (pre-T02, cae317c)
// ---------------------------------------------------------------------------

// MUST live on wiki/Developer.md (relocated reference)
const ENV_VARS = [
  'DATABASE_URL',
  'JWT_SECRET',
  'ALLOWED_ORIGINS',
  'PORT',
  'VITE_API_URL',
];

// Old README table listed 11 targets (build…clean); db-init was missing from
// the stale table and is the 12th real Makefile target regenerated in T01.
// Targets are matched with the `make ` prefix as presented in both tables.
const MAKE_TARGETS = [
  'build',
  'run',
  'test',
  'setup',
  'migrate-up',
  'migrate-down',
  'migrate-all',
  'docker-build',
  'docker-up',
  'docker-down',
  'clean',
  'db-init',
];

const TESTING_COMMANDS = [
  'make test',
  'go test -v ./...',
  'bun run test',
  'bun run lint',
  'bunx playwright test',
];

const DOMAIN_TERMS = [
  // Roles
  'employee',
  'manager',
  'finance',
  'customer',
  // Entry status flow
  'draft',
  'submitted',
  'pending_manager',
  'pending_finance',
  'approved',
  'rejected',
  // Approval actions
  'submit',
  'approve',
  'reject',
  'edit_approve',
  'edit_return',
  'partial_approve',
  'delegate',
  // Governance models
  'creator_controlled',
  'unanimous',
  'majority',
  // Project types
  'billable',
  'internal',
  // Expense categories (the 4 the old README carried; wiki regenerated 9)
  'mileage',
  'meal',
  'accommodation',
  'other',
  // Time entry model
  'TimeEntry',
  'TimeEntryItem',
];

// MAY live in README.md OR wiki/Developer.md
const STACK_ROWS = [
  'Go 1.26.1',
  'net/http',
  'hexagonal',
  'React 19',
  'TanStack Router v1',
  'TanStack React Query v5',
  'Vite',
  'TypeScript',
  'Tailwind CSS v4',
  'shadcn/ui',
  'PostgreSQL 15',
  'JWT',
  'golang-jwt/jwt/v5',
  'bcrypt',
  'golang.org/x/crypto',
  'stretchr/testify',
  'testcontainers-go',
  'Vitest',
  'Playwright',
  'multi-stage',
  'docker-compose',
];

// Canonical paths of the old README tree. Trees render leaf branches relative
// to their parent (e.g. `auth/` indented under `internal/`), so matching also
// accepts the branch token (name + '/') when the full path is absent — this is
// how the entries actually appeared in BOTH the old README and the wiki tree.
const STRUCTURE_ENTRIES = [
  'cmd/server',
  'cmd/migrate',
  'internal/core',
  'internal/adapters',
  'primary/http',
  'secondary/postgres',
  'internal/auth',
  'internal/cookies',
  'internal/db',
  'internal/handlers',
  'internal/middleware',
  'internal/models',
  'pkg/api',
  'migrations',
  'web',
  'Dockerfile',
  'docker-compose.yml',
  'Makefile',
  'go.mod',
];

// Last path segment, e.g. 'internal/auth' -> 'auth'
const branchOf = (path) => path.slice(path.lastIndexOf('/') + 1);

const PREREQUISITES = [
  '1.26.1', // Go >= 1.26.1 (bolded in both files, so check the version token)
  'Node.js',
  'Bun',
  'PostgreSQL 15',
  'Docker',
  'docker-compose',
];

const QUICKSTART_COMMANDS = [
  'git clone https://github.com/PriviteraStefano/hourglass.git',
  'cd hourglass',
  'make docker-up',
  'make docker-down',
  'make setup',
  'make run',
  'cd web',
  'bun install',
  'bun run dev',
  'http://localhost:8080',
  'http://localhost:3000',
];

// ---------------------------------------------------------------------------
// Category checks
// ---------------------------------------------------------------------------
function runCategory(label, terms, { wikiOnly, treeBranch }) {
  for (const term of terms) {
    const readmeHit = readme.includes(term);
    const wikiHit = dev.includes(term);
    // Tree-rendered entries may appear as a branch token (name + '/') rather
    // than the canonical full path, e.g. `auth/` under `internal/`.
    const branchHit =
      treeBranch &&
      (readme.includes(`${branchOf(term)}/`) || dev.includes(`${branchOf(term)}/`));
    const ok = wikiOnly ? wikiHit : readmeHit || wikiHit || branchHit;
    const where = [readmeHit ? 'readme' : null, wikiHit ? 'wiki' : null]
      .filter(Boolean)
      .join('+') || 'missing';
    check(`${label}: "${term}"`, ok, `found in: ${where}`);
  }
}

runCategory('env-var', ENV_VARS, { wikiOnly: true });
runCategory('make target', MAKE_TARGETS.map((t) => `make ${t}`), { wikiOnly: true });
runCategory('testing command', TESTING_COMMANDS, { wikiOnly: true });
runCategory('domain-model term', DOMAIN_TERMS, { wikiOnly: true });
runCategory('stack row', STACK_ROWS, { wikiOnly: false });
runCategory('structure entry', STRUCTURE_ENTRIES, { wikiOnly: false, treeBranch: true });
runCategory('prerequisite', PREREQUISITES, { wikiOnly: false });
runCategory('quickstart command', QUICKSTART_COMMANDS, { wikiOnly: false });

// ---------------------------------------------------------------------------
// README still under the 300-line gate (R004 / harness cross-check)
// ---------------------------------------------------------------------------
check('README under 300 lines', readmeLineCount < 300, `${readmeLineCount} lines`);

report();

// ---------------------------------------------------------------------------
function report() {
  for (const { name, ok } of results) {
    console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}`);
  }
  if (failures.length > 0) {
    console.error(`\nverify-dev-parity: ${failures.length} check(s) failed:`);
    for (const f of failures) console.error(`  - ${f}`);
    process.exit(1);
  }
  console.log(
    `\nverify-dev-parity: all ${results.length} checks passed ` +
      `(README ${readmeLineCount} lines, wiki ${devLineCount} lines)`
  );
  process.exit(0);
}
