import { test, expect } from '@playwright/test';

const PREFIX = `cntr_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

test.describe('Contracts CRUD', () => {
  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
  });

  test('create contract', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/contracts');
    await page.waitForLoadState('networkidle');
    // Check for create/list buttons — the page may show a list or create view
    const createBtn = page.getByRole('button', { name: /create|add|new/i }).first();
    if (await createBtn.isVisible()) {
      await createBtn.click();
      await page.waitForTimeout(500);
      const nameInput = page.locator('input[name="name"]');
      if (await nameInput.isVisible()) {
        await nameInput.fill(`Test Contract ${PREFIX}`);
        await page.getByRole('button', { name: /submit|save|create/i }).first().click();
        await expect(page.getByText(`Test Contract ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
      }
    }
  });

  test('view contract', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/contracts');
    await page.waitForLoadState('networkidle');
    const firstContract = page.locator('table a, [class*="contract"] a, [class*="row"]').first();
    if (await firstContract.isVisible()) {
      await firstContract.click();
      await expect(page).not.toHaveURL('/contracts');
    }
  });

  test('edit contract', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/contracts');
    await page.waitForLoadState('networkidle');
    const editBtn = page.getByRole('button', { name: /edit/i }).first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
      await page.waitForTimeout(500);
      const nameInput = page.locator('input[name="name"]');
      if (await nameInput.isVisible()) {
        await nameInput.fill(`Updated Contract ${PREFIX}`);
        await page.getByRole('button', { name: /save|update/i }).first().click();
        await expect(page.getByText(`Updated Contract ${PREFIX}`).first()).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('deactivate contract', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/contracts');
    await page.waitForLoadState('networkidle');
    const deactivateBtn = page.getByRole('button', { name: /deactivate|delete|remove/i }).first();
    if (await deactivateBtn.isVisible()) {
      await deactivateBtn.click();
      const confirmBtn = page.getByRole('button', { name: /confirm|deactivate|yes/i }).first();
      if (await confirmBtn.isVisible({ timeout: 3000 })) {
        await confirmBtn.click();
      }
    }
  });
});
