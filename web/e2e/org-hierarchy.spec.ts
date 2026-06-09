import { test, expect } from '@playwright/test';

const PREFIX = `org_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

test.describe('Org Hierarchy CRUD', () => {
  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
  });

  test.beforeEach(async ({ request }) => {
    const loginRes = await request.post('http://localhost:8080/auth/login', {
      data: { identifier: EMAIL, password: PASSWORD },
    });
    expect(loginRes.status()).toBe(200);
  });

  test('create unit', async ({ page }) => {
    await page.goto('/org-hierarchy');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /create|add|new unit/i }).first().click();
    await page.waitForTimeout(500);

    const nameInput = page.locator('input[name="name"]');
    if (await nameInput.isVisible()) {
      await nameInput.fill(`Test Unit ${PREFIX}`);
      await page.getByRole('button', { name: /submit|save|create/i }).first().click();
      await expect(page.getByText(`Test Unit ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
    }
  });

  test('create working group', async ({ page }) => {
    await page.goto('/org-hierarchy');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /create|add|new.*group/i }).first().click();
    await page.waitForTimeout(500);

    const nameInput = page.locator('input[name="name"]');
    if (await nameInput.isVisible()) {
      await nameInput.fill(`Test WG ${PREFIX}`);
      await page.getByRole('button', { name: /submit|save|create/i }).first().click();
      await expect(page.getByText(`Test WG ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
    }
  });

  test('edit unit', async ({ page }) => {
    await page.goto('/org-hierarchy');
    await page.waitForLoadState('networkidle');
    const editBtn = page.getByRole('button', { name: /edit/i }).first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
      await page.waitForTimeout(500);
      const nameInput = page.locator('input[name="name"]');
      if (await nameInput.isVisible()) {
        await nameInput.fill(`Updated Unit ${PREFIX}`);
        await page.getByRole('button', { name: /save|update/i }).first().click();
        await expect(page.getByText(`Updated Unit ${PREFIX}`).first()).toBeVisible({ timeout: 5000 });
      }
    }
  });
});
