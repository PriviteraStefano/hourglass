import { test, expect } from '@playwright/test';
import { fetchIds, psql, promoteToFinance } from './helpers';

test.describe.configure({ mode: 'serial' });

const PREFIX = `act_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

test.describe('Activities CRUD', () => {
  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
    // Fresh orgs have an empty activity_kinds catalog (kinds are seeded only
    // for the MVP org) — insert the canonical kinds so POST /activities
    // passes the D-2 catalog validation.
    const { orgId } = fetchIds(EMAIL);
    psql(
      `INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ('${orgId}', 'engagement', true), ('${orgId}', 'phase', true) ON CONFLICT (org_id, name) DO NOTHING`
    );
    // Update/Delete are finance-gated on the backend — promote for edit/delete coverage.
    promoteToFinance(EMAIL);
  });

  async function login(page: import('@playwright/test').Page) {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });
  }

  test('create activity', async ({ page }) => {
    await login(page);

    await page.goto('/activities');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /new activity/i }).click();
    await page.waitForTimeout(500);
    await page.getByPlaceholder('Activity name').fill(`Test Activity ${PREFIX}`);
    await page.getByPlaceholder('Select kind...').click();
    await page.getByRole('option', { name: /engagement/i }).click();
    await page.getByRole('button', { name: /^Create$/ }).click();
    await expect(page.getByText(`Test Activity ${PREFIX}`).first()).toBeVisible({
      timeout: 10000,
    });
  });

  test('view activity', async ({ page }) => {
    await login(page);

    await page.goto('/activities');
    await page.waitForLoadState('networkidle');
    const firstActivity = page
      .locator('[class*="cursor-pointer"]')
      .filter({ hasText: `Test Activity ${PREFIX}` })
      .first();
    if (await firstActivity.isVisible()) {
      await firstActivity.click();
      await expect(page).not.toHaveURL('/activities');
    }
  });

  test('edit activity', async ({ page }) => {
    await login(page);

    await page.goto('/activities');
    await page.waitForLoadState('networkidle');
    const row = page
      .locator('[class*="cursor-pointer"]')
      .filter({ hasText: `Test Activity ${PREFIX}` })
      .first();
    if (await row.isVisible()) {
      await row.click();
      await page.getByRole('button', { name: /edit/i }).click();
      await page.waitForTimeout(500);
      const nameInput = page.getByPlaceholder('Activity name');
      if (await nameInput.isVisible()) {
        await nameInput.fill(`Updated Activity ${PREFIX}`);
        await page.getByRole('button', { name: /save changes/i }).click();
        await expect(
          page.getByText(`Updated Activity ${PREFIX}`).first()
        ).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('delete activity', async ({ page }) => {
    await login(page);

    await page.goto('/activities');
    await page.waitForLoadState('networkidle');
    const row = page
      .locator('[class*="cursor-pointer"]')
      .filter({ hasText: `Updated Activity ${PREFIX}` })
      .first();
    if (await row.isVisible()) {
      await row.click();
      await page.getByRole('button', { name: /delete/i }).click();
      const confirmBtn = page.getByRole('button', {
        name: /^Delete$/,
      });
      if (await confirmBtn.isVisible({ timeout: 3000 })) {
        await confirmBtn.click();
      }
      await expect(page).toHaveURL(/\/activities/, { timeout: 10000 });
    }
  });
});
