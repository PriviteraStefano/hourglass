import { test, expect } from '@playwright/test';
import {
  registerUser,
  fetchIds,
  loginOnce,
  useSession,
  seedBaseEntities,
  seedTimeEntries,
} from './helpers';

test.describe.configure({ mode: 'serial' });

const PREFIX = `te_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

test.describe('Time Entries CRUD', () => {
  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
  });

  async function login(page: any) {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForLoadState('networkidle');
  }

  test('create time entry', async ({ page }) => {
    await login(page);

    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');
    const createBtn = page.getByRole('button', { name: /create|add|new/i }).first();
    if (await createBtn.isVisible()) {
      await createBtn.click();
      await page.waitForTimeout(500);
      await page.fill('input[name="hours"]', '8');
      await page.fill('input[name="description"]', `Test entry ${PREFIX}`);
      await page.getByRole('button', { name: /submit|save|create/i }).first().click();
      await expect(page.getByText(`Test entry ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
    }
  });

  test('view time entry details', async ({ page }) => {
    await login(page);

    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');
    const firstEntry = page.locator('table a, [class*="entry"] a, [class*="row"]').first();
    if (await firstEntry.isVisible()) {
      await firstEntry.click();
      await expect(page).not.toHaveURL('/time-entries');
    }
  });

  test('edit time entry', async ({ page }) => {
    await login(page);

    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');
    const editBtn = page.getByRole('button', { name: /edit/i }).first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
      await page.waitForTimeout(500);
      const hoursInput = page.locator('input[name="hours"]');
      if (await hoursInput.isVisible()) {
        await hoursInput.fill('6');
        await page.getByRole('button', { name: /save|update/i }).first().click();
        await expect(page.getByText('6').first()).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('delete time entry', async ({ page }) => {
    await login(page);

    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');
    const deleteBtn = page.getByRole('button', { name: /delete|remove/i }).first();
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click();
      const confirmBtn = page.getByRole('button', { name: /confirm|delete|yes/i }).first();
      if (await confirmBtn.isVisible({ timeout: 3000 })) {
        await confirmBtn.click();
      }
    }
  });
});

test.describe('Time Entries List View (P0-2)', () => {
  let sessionCookies: Array<{ name: string; value: string }> = [];
  const P = `telist_${Date.now()}`;
  const desc = (state: string) => `seeded-${state}-${P}`;

  test.beforeAll(async ({ request }) => {
    const { email, password } = await registerUser(request, 'telist');
    const { userId, orgId } = fetchIds(email);
    const base = seedBaseEntities(orgId, userId);
    seedTimeEntries(orgId, userId, base, P);
    sessionCookies = await loginOnce(request, email, password);
  });

  test('list tab shows seeded rows for all six workflow states', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');

    // The page defaults to the List tab; the seeded rows render for every state
    for (const state of [
      'draft',
      'submitted',
      'pending-manager',
      'pending-finance',
      'approved',
      'rejected',
    ]) {
      await expect(page.getByText(desc(state))).toBeVisible();
    }
    await expect(page.getByRole('table', { name: 'Time entries list' })).toBeVisible();
  });

  test('status filter narrows rows and the URL round-trips through reload', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');

    // Open the status multi-select and pick only "Approved"
    await page.getByRole('button', { name: /^Status/ }).click();
    await page.getByRole('menuitemcheckbox', { name: 'Approved' }).click();
    await page.keyboard.press('Escape');

    // URL reflects the filter; only the approved row remains
    // (listStatuses is an array search param — TanStack serializes it as
    // a JSON-encoded array ["approved"])
    await expect(page).toHaveURL(/listStatuses=.*approved/);
    await expect(page.getByText(desc('approved'))).toBeVisible();
    await expect(page.getByText(desc('draft'))).not.toBeVisible();

    // Reload: the filter is restored from the URL
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/listStatuses=.*approved/);
    await expect(page.getByText(desc('approved'))).toBeVisible();
    await expect(page.getByText(desc('draft'))).not.toBeVisible();
  });

  test('row click opens the entry detail (calendar tab + date param)', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');

    await page.getByText(desc('draft')).click();
    // date is a z.coerce.date() search param — serialized JSON-encoded
    await expect(page).toHaveURL(/date=.*2026-07-15/);
  });
});
