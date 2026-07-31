import { execSync } from "node:child_process";
import type { BrowserContext } from "@playwright/test";

/**
 * Shared e2e helpers (Phase 08 patterns).
 *
 * Rate-limit budgeting: the backend caps anonymous register/login at 5/min per
 * IP and the outer route limiter at 20/min (defaults). E2E suites register +
 * login ONCE via the API in `beforeAll` and inject the session cookies into
 * each test's browser context. Start the backend for e2e with:
 *
 *   RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500 go run ./cmd/server
 *
 * Seeding: direct-Postgres inserts via `docker exec hourglass-postgres` — the
 * same mechanism the customers suite uses for role promotion. The public API
 * cannot produce rows in all six workflow states (or expense receipt/mileage
 * shapes) in one call, and `POST /expenses` is still broken backend-side
 * (deferred-items.md), so list-view suites seed their datasets.
 */

/** Run SQL against the local dockerized Postgres. Returns the first output row (psql -t -A). */
export function psql(sql: string): string {
  const out = execSync(
    `docker exec hourglass-postgres psql -U hourglass -d hourglass -t -A -c ${JSON.stringify(sql)}`,
    { stdio: "pipe" }
  )
    .toString()
    .trim();
  // psql may append the command-status line (e.g. "INSERT 0 1") to the row
  // output when using RETURNING; only the first line is query data.
  return out.split("\n")[0].trim();
}

export async function registerUser(
  request: import("@playwright/test").APIRequestContext,
  prefix: string
): Promise<{ email: string; password: string }> {
  const stamp = `${prefix}_${Date.now()}`;
  const email = `${stamp}@test.com`;
  const password = "Password123!";
  const res = await request.post("http://localhost:8080/auth/register", {
    data: {
      email,
      username: stamp,
      password,
      firstname: "Test",
      lastname: "User",
      organization_name: `${stamp}_org`,
    },
  });
  if (!res.ok()) {
    throw new Error(`register failed: ${res.status()} ${await res.text()}`);
  }
  return { email, password };
}

export function fetchIds(email: string): { userId: string; orgId: string } {
  const row = psql(
    `SELECT u.id || '|' || m.organization_id FROM users u JOIN organization_memberships m ON m.user_id = u.id WHERE u.email = '${email}' LIMIT 1`
  );
  const [userId, orgId] = row.split("|");
  if (!userId || !orgId) {
    throw new Error(`could not resolve user/org ids for ${email}`);
  }
  return { userId, orgId };
}

export function promoteToFinance(email: string) {
  psql(
    `UPDATE organization_memberships SET role='finance' FROM users WHERE users.id = organization_memberships.user_id AND users.email='${email}'`
  );
}

/** Login via the API and return the session Set-Cookie pairs (no rate-limit burn). */
export async function loginOnce(
  request: import("@playwright/test").APIRequestContext,
  email: string,
  password: string
): Promise<Array<{ name: string; value: string }>> {
  const login = await request.post("http://localhost:8080/auth/login", {
    data: { identifier: email, password },
  });
  if (!login.ok()) {
    throw new Error(`login failed: ${login.status()} ${await login.text()}`);
  }
  const cookies: Array<{ name: string; value: string }> = [];
  for (const h of login.headersArray()) {
    if (h.name.toLowerCase() === "set-cookie") {
      const [pair] = h.value.split(";");
      const eq = pair.indexOf("=");
      if (eq > 0) {
        cookies.push({ name: pair.slice(0, eq), value: pair.slice(eq + 1) });
      }
    }
  }
  return cookies;
}

export async function useSession(
  context: BrowserContext,
  cookies: Array<{ name: string; value: string }>
) {
  await context.addCookies(
    cookies.map((c) => ({ name: c.name, value: c.value, url: "http://localhost:3000" }))
  );
}

/**
 * Seed the org-scoped entities the time_entries FK chain requires
 * (unit -> project -> subproject -> working group). Returns their ids.
 */
export function seedBaseEntities(
  orgId: string,
  userId: string
): { unitId: string; projectId: string; subprojectId: string } {
  const unitId = psql(
    `INSERT INTO units (org_id, name) VALUES ('${orgId}', 'Default Unit') RETURNING id`
  );
  const projectId = psql(
    `INSERT INTO projects (org_id, name, project_type, type, governance_model, created_by_org_id) VALUES ('${orgId}', 'E2E Seed Project', 'billable', 'billable', 'creator_controlled', '${orgId}') RETURNING id`
  );
  const subprojectId = psql(
    `INSERT INTO subprojects (project_id, name) VALUES ('${projectId}', 'E2E Seed Subproject') RETURNING id`
  );
  psql(
    `INSERT INTO working_groups (org_id, subproject_id, name, manager_id) VALUES ('${orgId}', '${subprojectId}', 'E2E Seed WG', '${userId}')`
  );
  return { unitId, projectId, subprojectId };
}

/** Seed one time entry per workflow state, all inside the current month. */
export function seedTimeEntries(
  orgId: string,
  userId: string,
  base: { unitId: string; projectId: string; subprojectId: string },
  prefix: string
) {
  const wgId = psql(
    `SELECT id FROM working_groups WHERE org_id='${orgId}' LIMIT 1`
  );
  const rows = [
    { status: "draft", date: "2026-07-15", hours: 2.5, desc: `seeded-draft-${prefix}` },
    { status: "submitted", date: "2026-07-16", hours: 3, desc: `seeded-submitted-${prefix}` },
    { status: "pending_manager", date: "2026-07-17", hours: 4, desc: `seeded-pending-manager-${prefix}` },
    { status: "pending_finance", date: "2026-07-18", hours: 5, desc: `seeded-pending-finance-${prefix}` },
    { status: "approved", date: "2026-07-19", hours: 6, desc: `seeded-approved-${prefix}` },
    { status: "rejected", date: "2026-07-20", hours: 1, desc: `seeded-rejected-${prefix}` },
  ];
  for (const r of rows) {
    psql(
      `INSERT INTO time_entries (org_id, user_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, status) VALUES ('${orgId}', '${userId}', '${base.projectId}', '${base.subprojectId}', '${wgId}', '${base.unitId}', ${r.hours}, '${r.desc}', '${r.date} 00:00:00+00', '${r.status}')`
    );
  }
}

/**
 * Seed expenses across categories/statuses, including a receipt-backed row
 * (receipt_url) and a mileage row with km_distance — the two list indicators.
 */
export function seedExpenses(
  orgId: string,
  userId: string,
  base: { unitId: string; projectId: string },
  prefix: string
) {
  const rows = [
    { status: "approved", category: "meal", amount: 42.5, date: "2026-07-10", desc: `seeded-meal-approved-${prefix}`, km: null, receipt: null },
    { status: "draft", category: "meal", amount: 18.0, date: "2026-07-11", desc: `seeded-meal-draft-${prefix}`, km: null, receipt: null },
    { status: "submitted", category: "mileage", amount: 0.36, date: "2026-07-12", desc: `seeded-mileage-${prefix}`, km: 12.5, receipt: null },
    { status: "draft", category: "accommodation", amount: 120.0, date: "2026-07-13", desc: `seeded-receipt-${prefix}`, km: null, receipt: "https://example.com/receipt.pdf" },
    { status: "rejected", category: "other", amount: 15.0, date: "2026-07-14", desc: `seeded-other-${prefix}`, km: null, receipt: null },
  ];
  for (const r of rows) {
    const km = r.km == null ? "NULL" : String(r.km);
    const receipt = r.receipt == null ? "NULL" : `'${r.receipt}'`;
    psql(
      `INSERT INTO expenses (org_id, user_id, project_id, unit_id, category, amount, km_distance, description, expense_date, status, receipt_url) VALUES ('${orgId}', '${userId}', '${base.projectId}', '${base.unitId}', '${r.category}', ${r.amount}, ${km}, '${r.desc}', '${r.date} 00:00:00+00', '${r.status}', ${receipt})`
    );
  }
}

/** Seed external + internal customers for the P0-3 list/search/detail coverage. */
export function seedCustomers(orgId: string, prefix: string) {
  psql(
    `INSERT INTO customers (org_id, name, email, is_internal) VALUES ('${orgId}', 'Alpha Industries', 'alpha@${prefix}.com', false)`
  );
  psql(
    `INSERT INTO customers (org_id, name, email, is_internal) VALUES ('${orgId}', 'Zeta Labs', 'zeta@${prefix}.com', false)`
  );
  psql(
    `INSERT INTO customers (org_id, name, email, is_internal) VALUES ('${orgId}', 'Internal Ops', 'internal@${prefix}.com', true)`
  );
}
