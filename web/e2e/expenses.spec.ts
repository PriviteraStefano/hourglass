import { test, expect } from '@playwright/test';
import {
  registerUser,
  fetchIds,
  loginOnce,
  useSession,
  seedBaseEntities,
  seedExpenses,
} from './helpers';

test.describe.configure({ mode: 'serial' });

// P0-2 verification for the expenses list view. The backend's POST /expenses is
// still broken (unit_id FK violation — deferred-items.md), so the dataset is
// seeded directly in Postgres: multiple categories, one receipt-backed row, one
// mileage row with km_distance.
test.describe('Expenses List View (P0-2)', () => {
  let sessionCookies: Array<{ name: string; value: string }> = [];
  const P = `explist_${Date.now()}`;
  const desc = (state: string) => `seeded-${state}-${P}`;

  test.beforeAll(async ({ request }) => {
    const { email, password } = await registerUser(request, 'explist');
    const { userId, orgId } = fetchIds(email);
    const base = seedBaseEntities(orgId, userId);
    seedExpenses(orgId, userId, base, P);
    sessionCookies = await loginOnce(request, email, password);
  });

  test('list tab shows seeded rows with categories', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/expenses');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('table', { name: 'Expenses list' })).toBeVisible();
    await expect(page.getByText(desc('meal-approved'))).toBeVisible();
    await expect(page.getByText(desc('meal-draft'))).toBeVisible();
    await expect(page.getByText(desc('mileage'))).toBeVisible();
    await expect(page.getByText(desc('receipt'))).toBeVisible();
    await expect(page.getByText(desc('other'))).toBeVisible();
    const table = page.getByRole('table', { name: 'Expenses list' });
    await expect(table.getByText('Meal', { exact: true }).first()).toBeVisible();
    await expect(table.getByText('Mileage', { exact: true }).first()).toBeVisible();
  });

  test('receipt indicator renders on receipt-backed rows', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/expenses');
    await page.waitForLoadState('networkidle');

    // PaperclipIcon carries title="Receipt attached" on the receipt column cell
    await expect(page.locator('span[title="Receipt attached"]')).toBeVisible();
  });

  test('mileage row surfaces distance', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/expenses');
    await page.waitForLoadState('networkidle');

    // km_distance 12.5 formats as "12.50 km" next to the description
    await expect(page.getByText(/12\.50 km/)).toBeVisible();
  });

  test('category + status filters compose and the URL round-trips through reload', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/expenses');
    await page.waitForLoadState('networkidle');

    // Status filter: Approved only
    await page.getByRole('button', { name: /^Status/ }).click();
    await page.getByRole('menuitemcheckbox', { name: 'Approved' }).click();
    await page.keyboard.press('Escape');
    // listStatuses is an array search param — TanStack serializes it as
    // a JSON-encoded array ["approved"]
    await expect(page).toHaveURL(/listStatuses=.*approved/);
    // Only the approved meal row survives the status filter alone
    await expect(page.getByText(desc('meal-approved'))).toBeVisible();
    await expect(page.getByText(desc('meal-draft'))).not.toBeVisible();
    await expect(page.getByText(desc('mileage'))).not.toBeVisible();

    // Compose with the category filter: Meal → still only the approved meal row
    await page.getByLabel('Category filter').selectOption({ label: 'Meal' });
    await expect(page).toHaveURL(/listCategory=meal/);
    await expect(page).toHaveURL(/listStatuses=.*approved/);
    await expect(page.getByText(desc('meal-approved'))).toBeVisible();
    await expect(page.getByText(desc('meal-draft'))).not.toBeVisible();

    // Reload: both filters restored from the URL
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/listCategory=meal/);
    await expect(page).toHaveURL(/listStatuses=.*approved/);
    await expect(page.getByText(desc('meal-approved'))).toBeVisible();
    await expect(page.getByText(desc('meal-draft'))).not.toBeVisible();
  });
});
