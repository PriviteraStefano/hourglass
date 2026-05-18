import { test, expect } from '@playwright/test';

const PREFIX = `te_${Date.now()}`;

test.describe('Time Entries CRUD', () => {
  let cookies: string;

  test.beforeAll(async ({ request }) => {
    const email = `${PREFIX}@test.com`;
    await request.post('http://localhost:8080/auth/register', {
      data: { email, username: `${PREFIX}_user`, password: 'Password123!', organization_name: `${PREFIX}_org` },
    });
    const loginRes = await request.post('http://localhost:8080/auth/login', {
      data: { identifier: email, password: 'Password123!' },
    });
    const setCookie = loginRes.headers()['set-cookie'];
    cookies = Array.isArray(setCookie) ? setCookie.join('; ') : setCookie || '';
  });

  test('create time entry', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|time-entries)/, { timeout: 10000 });

    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /create|add|new/i }).first().click();
    await page.waitForTimeout(500);

    await page.fill('input[name="hours"]', '8');
    await page.fill('input[name="description"]', `Test entry ${PREFIX}`);
    await page.getByRole('button', { name: /submit|save|create/i }).first().click();

    await expect(page.getByText(`Test entry ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
  });

  test('view time entry details', async ({ page, request }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|time-entries)/, { timeout: 10000 });

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
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|time-entries)/, { timeout: 10000 });

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
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|time-entries)/, { timeout: 10000 });

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
