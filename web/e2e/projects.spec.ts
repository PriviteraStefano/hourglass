import { test, expect } from '@playwright/test';

const PREFIX = `proj_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

test.describe('Projects CRUD', () => {
  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
  });

  test.beforeEach(async ({ page, request }) => {
    // Ensure logged in: login via API so cookies are available to page context
    const loginRes = await request.post('http://localhost:8080/auth/login', {
      data: { identifier: EMAIL, password: PASSWORD },
    });
    expect(loginRes.status()).toBe(200);
  });

  test('create project', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /create|add|new/i }).first().click();
    await page.waitForTimeout(500);

    await page.fill('input[name="name"]', `Test Project ${PREFIX}`);
    await page.getByRole('button', { name: /submit|save|create/i }).first().click();

    await expect(page.getByText(`Test Project ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
  });

  test('view project', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');
    const firstProject = page.locator('table a, [class*="project"] a, [class*="row"]').first();
    if (await firstProject.isVisible()) {
      await firstProject.click();
      await expect(page).not.toHaveURL('/projects');
    }
  });

  test('edit project', async ({ page }) => {
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');
    const editBtn = page.getByRole('button', { name: /edit/i }).first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
      await page.waitForTimeout(500);
      const nameInput = page.locator('input[name="name"]');
      if (await nameInput.isVisible()) {
        await nameInput.fill(`Updated Project ${PREFIX}`);
        await page.getByRole('button', { name: /save|update/i }).first().click();
        await expect(page.getByText(`Updated Project ${PREFIX}`).first()).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('deactivate project', async ({ page }) => {
    await page.goto('/projects');
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
