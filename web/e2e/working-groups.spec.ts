import { test, expect } from '@playwright/test';
import {
  registerUser,
  fetchIds,
  psql,
  loginOnce,
  useSession,
} from './helpers';

test.describe.configure({ mode: 'serial' });

// Working Groups surface (Plan 10-06): list + create/edit + members against
// the live WG API. The org creator (manager role) is the WG manager; a second
// user (custom names so the org-member pickers stay unambiguous — registerUser
// hardcodes "Test User") is moved into the same org to serve as delegate +
// member. The activity is seeded WITHOUT a WG so the empty state is testable
// first, then the create dialog fills it.

const PREFIX = `wg_${Date.now()}`;
const PASSWORD = 'Password123!';

let managerEmail: string;
let memberEmail: string;
let orgId: string;
let managerId: string;
let activityId: string;
let unitId: string;
let managerCookies: Array<{ name: string; value: string }> = [];

async function registerNamedUser(
  request: import('@playwright/test').APIRequestContext,
  prefix: string,
  firstname: string,
  lastname: string
): Promise<{ email: string; password: string }> {
  const stamp = `${prefix}_${Date.now()}`;
  const email = `${stamp}@test.com`;
  const res = await request.post('http://localhost:8080/auth/register', {
    data: {
      email,
      username: stamp,
      password: PASSWORD,
      firstname,
      lastname,
      organization_name: `${stamp}_org`,
    },
  });
  if (!res.ok()) {
    throw new Error(`register failed: ${res.status()} ${await res.text()}`);
  }
  return { email, password: PASSWORD };
}

test.beforeAll(async ({ request }) => {
  // Org creator (role manager) — becomes the WG manager.
  const manager = await registerUser(request, `wgm`);
  managerEmail = manager.email;
  const ids = fetchIds(managerEmail);
  orgId = ids.orgId;
  managerId = ids.userId;

  // Member user moved into the same org (fresh org from registration is
  // abandoned). Distinct names so pickers resolve unambiguously.
  const member = await registerNamedUser(request, `wge`, 'Eve', 'E2E');
  memberEmail = member.email;
  const memberUserId = fetchIds(memberEmail).userId;
  psql(
    `UPDATE organization_memberships SET organization_id='${orgId}', role='employee' WHERE user_id='${memberUserId}'`
  );

  // Seed unit + activity WITHOUT a working group (empty state must be real).
  unitId = psql(
    `INSERT INTO units (org_id, name, description, code) VALUES ('${orgId}', 'E2E Unit', '', '') RETURNING id`
  );
  psql(
    `INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ('${orgId}', 'engagement', true) ON CONFLICT (org_id, name) DO NOTHING`
  );
  activityId = psql(
    `INSERT INTO activities (org_id, name, description, kind, governance_model, created_by_org_id) VALUES ('${orgId}', 'E2E WG Activity', '', 'engagement', 'creator_controlled', '${orgId}') RETURNING id`
  );

  // Session cookies AFTER the membership move so the JWT carries the org.
  managerCookies = await loginOnce(request, managerEmail, PASSWORD);
});

test.describe('Working Groups CRUD e2e (10-06)', () => {
  test('empty state → create WG via dialog → card appears with name/manager/members', async ({
    browser,
    page,
  }) => {
    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const wgPage = await context.newPage();

    await wgPage.goto('/working-groups');
    await expect(
      wgPage.getByRole('heading', { name: 'Working Groups', exact: true })
    ).toBeVisible({ timeout: 10000 });

    // Locked empty state copy + CTA.
    await expect(
      wgPage.getByRole('heading', { name: 'No working groups yet' })
    ).toBeVisible({ timeout: 10000 });
    await expect(
      wgPage.getByText('Working groups assign people to activities. Create one to start staffing work.')
    ).toBeVisible();

    // Create via dialog (header CTA; the empty state CTA is a second match).
    await wgPage.getByRole('button', { name: 'New working group' }).first().click();
    await expect(
      wgPage.getByRole('heading', { name: 'New working group' })
    ).toBeVisible();

    await wgPage.getByPlaceholder('Working group name').fill(`E2E WG ${PREFIX}`);

    await wgPage.getByPlaceholder('Select activity...').click();
    await wgPage.getByRole('option', { name: 'E2E WG Activity' }).click();

    await wgPage.getByPlaceholder('Select manager...').click();
    await wgPage.getByRole('option', { name: 'Test User' }).click();

    await wgPage.getByRole('button', { name: 'Create', exact: true }).click();

    // Card appears with name + resolved manager + member count. The closed
    // dialog's combobox portal stays mounted (hidden), so scope assertions
    // to the card surface.
    const card = wgPage.getByText(`E2E WG ${PREFIX}`).first();
    await expect(card).toBeVisible({ timeout: 10000 });
    const cardSurface = card.locator('xpath=ancestor::div[contains(@class,"bg-card")]');
    await expect(cardSurface.getByText('Test User')).toBeVisible();
    await expect(cardSurface.getByText('0 members')).toBeVisible();
    await expect(cardSurface.getByText('E2E WG Activity')).toBeVisible();

    await context.close();
  });

  test('search filters the grid client-side', async ({ browser, page }) => {
    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const wgPage = await context.newPage();

    await wgPage.goto('/working-groups');
    await expect(wgPage.getByText(`E2E WG ${PREFIX}`).first()).toBeVisible({
      timeout: 10000,
    });

    await wgPage
      .getByPlaceholder('Search working groups...')
      .fill(`E2E WG ${PREFIX}`);
    await expect(wgPage.getByText(`E2E WG ${PREFIX}`).first()).toBeVisible();
    await expect(wgPage.getByText('No working groups match your search')).not.toBeVisible();

    await wgPage
      .getByPlaceholder('Search working groups...')
      .fill('zzz-no-match');
    await expect(
      wgPage.getByText('No working groups match your search')
    ).toBeVisible({ timeout: 10000 });

    await context.close();
  });

  test('edit: change name + delegate → persists after reload', async ({
    browser,
    page,
  }) => {
    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const wgPage = await context.newPage();

    await wgPage.goto('/working-groups');
    const card = wgPage.getByText(`E2E WG ${PREFIX}`).first();
    await expect(card).toBeVisible({ timeout: 10000 });

    await card.locator('xpath=ancestor::div[contains(@class,"bg-card")]').getByRole('button', { name: 'Edit' }).click();
    await expect(
      wgPage.getByRole('heading', { name: 'Edit working group' })
    ).toBeVisible();

    await wgPage.getByPlaceholder('Working group name').fill(`Renamed WG ${PREFIX}`);

    // Add a delegate via the chips multi-select, then close the popup (the
    // multi-select stays open for further picks and overlays the footer).
    await wgPage.getByPlaceholder('Select delegates...').click();
    await wgPage.getByRole('option', { name: 'Eve E2E' }).click();
    await expect(wgPage.getByText('Eve E2E').first()).toBeVisible();
    await wgPage.keyboard.press('Escape');

    await wgPage.getByRole('button', { name: 'Save', exact: true }).click();

    // Renamed card persists across a reload.
    await expect(wgPage.getByText(`Renamed WG ${PREFIX}`).first()).toBeVisible({
      timeout: 10000,
    });
    await wgPage.reload();
    await expect(wgPage.getByText(`Renamed WG ${PREFIX}`).first()).toBeVisible({
      timeout: 10000,
    });

    // Delegate persisted server-side (backend is authoritative, T-10-06-1).
    const memberUserId = fetchIds(memberEmail).userId;
    const wgId = psql(
      `SELECT id FROM working_groups WHERE name='Renamed WG ${PREFIX}' LIMIT 1`
    );
    expect(wgId).not.toBe('');
    const delegateIds = psql(
      `SELECT delegate_ids FROM working_groups WHERE id='${wgId}'`
    );
    expect(delegateIds).toContain(memberUserId);

    await context.close();
  });

  test('members: add → listed with role badge → remove → gone', async ({
    browser,
    page,
  }) => {
    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const wgPage = await context.newPage();

    await wgPage.goto('/working-groups');
    const card = wgPage.getByText(`Renamed WG ${PREFIX}`).first();
    await expect(card).toBeVisible({ timeout: 10000 });

    await card.locator('xpath=ancestor::div[contains(@class,"bg-card")]').getByRole('button', { name: 'Members' }).click();
    await expect(
      wgPage.getByRole('heading', { name: /Members — / })
    ).toBeVisible();

    // Add member (user + unit + default role 'member').
    await wgPage.getByPlaceholder('Select user...').click();
    await wgPage.getByRole('option', { name: 'Eve E2E' }).click();
    await wgPage.getByPlaceholder('Unit...').click();
    await wgPage.getByRole('option', { name: 'E2E Unit' }).click();
    await wgPage.getByRole('button', { name: 'Add member' }).click();

    // Listed with role badge (scoped to the member row — combobox portals
    // from the add form stay mounted and also contain the name).
    const memberRow = wgPage
      .locator('div.rounded-lg.border')
      .filter({ hasText: 'Eve E2E' })
      .first();
    await expect(memberRow).toBeVisible({ timeout: 10000 });
    await expect(memberRow.getByText('member', { exact: true })).toBeVisible();

    // Card member count reflects the addition after invalidation.
    await wgPage.getByRole('button', { name: 'Close' }).first().click();
    await expect(wgPage.getByText('1 member').first()).toBeVisible({
      timeout: 10000,
    });

    // Remove member via destructive confirm → gone.
    await card.locator('xpath=ancestor::div[contains(@class,"bg-card")]').getByRole('button', { name: 'Members' }).click();
    const memberRowAgain = wgPage
      .locator('div.rounded-lg.border')
      .filter({ hasText: 'Eve E2E' })
      .first();
    await expect(memberRowAgain).toBeVisible({ timeout: 10000 });
    await memberRowAgain.getByRole('button', { name: 'Remove' }).click();
    const confirmDialog = wgPage.getByRole('alertdialog');
    await expect(
      confirmDialog.getByRole('heading', { name: 'Remove member' })
    ).toBeVisible();
    await confirmDialog.getByRole('button', { name: 'Remove' }).click();

    await expect(wgPage.getByText('No members yet.')).toBeVisible({
      timeout: 10000,
    });

    await context.close();
  });

  test('delete: destructive confirm → card gone + empty state returns', async ({
    browser,
    page,
  }) => {
    const context = await browser.newContext();
    await useSession(context, managerCookies);
    const wgPage = await context.newPage();

    await wgPage.goto('/working-groups');
    const card = wgPage.getByText(`Renamed WG ${PREFIX}`).first();
    await expect(card).toBeVisible({ timeout: 10000 });

    await card.locator('xpath=ancestor::div[contains(@class,"bg-card")]').getByRole('button', { name: 'Delete' }).click();
    await expect(
      wgPage.getByRole('heading', { name: 'Delete working group' })
    ).toBeVisible();
    await wgPage.getByRole('button', { name: 'Delete', exact: true }).click();

    // Card gone + the empty state returns (only WG was deleted).
    await expect(card).not.toBeVisible({ timeout: 10000 });
    await expect(
      wgPage.getByRole('heading', { name: 'No working groups yet' })
    ).toBeVisible({ timeout: 10000 });

    await context.close();
  });
});
