import { test, expect } from '@playwright/test';
import {
  registerUser,
  fetchIds,
  psql,
  loginOnce,
  useSession,
  seedBaseEntities,
} from './helpers';

test.describe.configure({ mode: 'serial' });

// Approvals queue (Plan 10-05): one page, stage-filtered Manager/Finance
// queues. Register three users and move the employee + finance memberships
// into the manager's org via psql (registration always creates a fresh org —
// same mechanism as customers.spec's promoteToFinance). The manager is also
// the WG manager (seedBaseEntities anchors a WG with manager_id = manager),
// so entries on the child activity route to the manager stage and appear in
// the manager's WG-scoped queue (T-10-05-3 admission verified end-to-end).

const PREFIX = `apr_${Date.now()}`;
const PASSWORD = 'Password123!';

let managerEmail: string;
let employeeEmail: string;
let financeEmail: string;
let orgId: string;
let managerId: string;
let employeeId: string;
let financeId: string;
let unitId: string;
let childActivityId: string;
let managerCookies: Array<{ name: string; value: string }> = [];
let employeeCookies: Array<{ name: string; value: string }> = [];
let financeCookies: Array<{ name: string; value: string }> = [];

// Create + submit a time entry via the API (employee-owned) and return its id.
async function createAndSubmitEntry(
  request: import('@playwright/test').APIRequestContext,
  cookies: Array<{ name: string; value: string }>,
  hours: number,
  desc: string,
  date: string
): Promise<string> {
  const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ');
  const created = await request.post('http://localhost:8080/time-entries', {
    headers: { cookie: cookieHeader },
    data: {
      activity_id: childActivityId,
      unit_id: unitId,
      hours,
      description: desc,
      date,
    },
  });
  if (!created.ok()) {
    throw new Error(`create entry failed: ${created.status()} ${await created.text()}`);
  }
  const createdBody = await created.json();
  const id = createdBody.data?.id;
  if (!id) throw new Error(`create entry response missing id: ${JSON.stringify(createdBody)}`);
  const submitted = await request.post(`http://localhost:8080/time-entries/${id}/submit`, {
    headers: { cookie: cookieHeader },
  });
  if (!submitted.ok()) {
    throw new Error(`submit failed: ${submitted.status()} ${await submitted.text()}`);
  }
  return id;
}

async function entryStatus(
  request: import('@playwright/test').APIRequestContext,
  cookies: Array<{ name: string; value: string }>,
  id: string
): Promise<string> {
  const res = await request.get(`http://localhost:8080/time-entries/${id}`, {
    headers: { cookie: cookies.map((c) => `${c.name}=${c.value}`).join('; ') },
  });
  if (!res.ok()) {
    throw new Error(`get entry failed: ${res.status()} ${await res.text()}`);
  }
  return (await res.json()).data?.status as string;
}

test.beforeAll(async ({ request }) => {
  // Manager (org creator — role manager) + seed the org's base entities.
  const manager = await registerUser(request, `aprm`);
  managerEmail = manager.email;
  const mIds = fetchIds(managerEmail);
  orgId = mIds.orgId;
  managerId = mIds.userId;
  const base = seedBaseEntities(orgId, managerId);
  unitId = base.unitId;
  childActivityId = base.childActivityId;

  // Employee — register (fresh org), then move membership into manager's org
  // with role employee.
  const employee = await registerUser(request, `apre`);
  employeeEmail = employee.email;
  employeeId = fetchIds(employeeEmail).userId;
  psql(
    `UPDATE organization_memberships SET organization_id='${orgId}', role='employee' WHERE user_id='${employeeId}'`
  );

  // Finance — same move, role finance.
  const finance = await registerUser(request, `aprf`);
  financeEmail = finance.email;
  financeId = fetchIds(financeEmail).userId;
  psql(
    `UPDATE organization_memberships SET organization_id='${orgId}', role='finance' WHERE user_id='${financeId}'`
  );

  // Session cookies AFTER the membership moves so the JWT carries the right
  // org + role.
  managerCookies = await loginOnce(request, managerEmail, PASSWORD);
  employeeCookies = await loginOnce(request, employeeEmail, PASSWORD);
  financeCookies = await loginOnce(request, financeEmail, PASSWORD);
});

test.describe('Approvals queue e2e (10-05)', () => {
  test('employee visibility: no Review group, /approvals shows the muted notice', async ({
    browser,
    page,
  }) => {
    const context = await browser.newContext();
    await useSession(context, employeeCookies);
    const empPage = await context.newPage();

    // Direct URL access → locked muted state (no pending queries fired, so no
    // 403 spam).
    await empPage.goto('/approvals');
    await expect(
      empPage.getByText('Approvals are for manager and finance stages.')
    ).toBeVisible({ timeout: 10000 });

    // Sidebar: no Review group → no Approvals nav link for the employee.
    await empPage.goto('/');
    await expect(empPage.getByRole('heading', { name: 'Today' })).toBeVisible({
      timeout: 10000,
    });
    await expect(empPage.getByRole('link', { name: 'Approvals' })).toHaveCount(0);

    await context.close();
  });

  test('manager approves an entry and it leaves the queue', async ({
    browser,
    page,
    request,
  }) => {
    const entryId = await createAndSubmitEntry(request, employeeCookies, 4, `approve-me-${PREFIX}`, '2026-08-01');

    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const mgrPage = await context.newPage();

    await mgrPage.goto('/approvals');
    // Manager queue row appears (hours 4, single stage → no tab bar).
    const row = mgrPage.locator('li').filter({ hasText: '2026-08-01' }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await expect(row.getByRole('button', { name: 'Approve' })).toBeVisible();

    await row.getByRole('button', { name: 'Approve' }).click();
    await expect(mgrPage.getByText('Entry approved')).toBeVisible({ timeout: 10000 });

    // Approving advances the two-stage chain: submitted → pending_finance.
    await expect(row).not.toBeVisible({ timeout: 10000 });
    const status = await entryStatus(request, managerCookies, entryId);
    expect(status).toBe('pending_finance');

    await context.close();
  });

  test('manager rejects with a reason and the employee sees rejected status', async ({
    browser,
    page,
    request,
  }) => {
    const entryId = await createAndSubmitEntry(request, employeeCookies, 5, `reject-me-${PREFIX}`, '2026-08-02');

    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const mgrPage = await context.newPage();

    await mgrPage.goto('/approvals');
    const row = mgrPage.locator('li').filter({ hasText: '2026-08-02' }).first();
    await expect(row).toBeVisible({ timeout: 10000 });

    // Reject requires a reason (existing ApprovalButtons pattern, T-10-05-4).
    await row.getByRole('button', { name: 'Reject' }).click();
    const reasonArea = mgrPage.getByLabel('Reason for rejection (required)');
    await expect(reasonArea).toBeVisible();
    // Confirm stays disabled for a short reason.
    await reasonArea.fill('too short');
    await expect(row.getByRole('button', { name: 'Reject' })).toBeDisabled();
    await reasonArea.fill('Hours do not match the logged time');
    await row.getByRole('button', { name: 'Reject' }).click();
    await expect(mgrPage.getByText('Entry rejected')).toBeVisible({ timeout: 10000 });
    await expect(row).not.toBeVisible({ timeout: 10000 });

    // The reason persists in the immutable approval history (backend), and
    // the entry status flips to rejected for the owner.
    const status = await entryStatus(request, employeeCookies, entryId);
    expect(status).toBe('rejected');
    const reasonRow = psql(
      `SELECT comment FROM time_entry_approvals WHERE time_entry_id='${entryId}' AND action='reject' ORDER BY created_at DESC LIMIT 1`
    );
    expect(reasonRow).toContain('Hours do not match');

    // Employee sees the Rejected status badge on their time-entries view.
    const empContext = await browser.newContext();
    await useSession(empContext, employeeCookies);
    const empPage = await empContext.newPage();
    await empPage.goto('/time-entries');
    await expect(empPage.getByText('Rejected').first()).toBeVisible({ timeout: 10000 });

    await context.close();
    await empContext.close();
  });

  test('finance stage: manager approves → finance sees it → approves', async ({
    browser,
    page,
    request,
  }) => {
    const entryId = await createAndSubmitEntry(request, employeeCookies, 6, `finance-chain-${PREFIX}`, '2026-08-03');

    // Manager approves via API (UI round-trip covered above).
    const mgrApprove = await request.post(`http://localhost:8080/time-entries/${entryId}/approve`, {
      headers: { cookie: managerCookies.map((c) => `${c.name}=${c.value}`).join('; ') },
    });
    if (!mgrApprove.ok()) {
      throw new Error(`manager approve failed: ${mgrApprove.status()} ${await mgrApprove.text()}`);
    }

    // Finance (single stage → no tab bar) sees the pending_finance row.
    const context = await browser.newContext();
    await useSession(context, financeCookies);
    const finPage = await context.newPage();

    await finPage.goto('/approvals?stage=finance');
    const row = finPage.locator('li').filter({ hasText: '2026-08-03' }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.getByRole('button', { name: 'Approve' }).click();
    await expect(finPage.getByText('Entry approved')).toBeVisible({ timeout: 10000 });
    await expect(row).not.toBeVisible({ timeout: 10000 });

    // Chain terminal: approved.
    const status = await entryStatus(request, employeeCookies, entryId);
    expect(status).toBe('approved');

    await context.close();
  });
});
