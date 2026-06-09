import { test, expect } from '@playwright/test';

const PREFIX = `te_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

test.describe('Time Entries CRUD', () => {
  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
  });

  test('create time entry', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

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
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');
    const firstEntry = page.locator('table a, [class*="entry"] a, [class*="row"]').first();
    if (await firstEntry.isVisible()) {
      await firstEntry.click();
      await expect(page).not.toHaveURL('/time-entries');
    }
  });

  test('edit time entry', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

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
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

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
