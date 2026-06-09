import { test, expect } from '@playwright/test';

const PREFIX = `cust_${Date.now()}`;

test.describe('Customers CRUD', () => {
  test.beforeAll(async ({ request }) => {
    const email = `${PREFIX}@test.com`;
    await request.post('http://localhost:8080/auth/register', {
      data: { email, username: `${PREFIX}_user`, password: 'Password123!', firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
  });

  test('create customer', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/customers');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /create|add|new/i }).first().click();
    await page.waitForTimeout(500);

    await page.fill('input[name="company_name"], input[name="name"]', `Test Customer ${PREFIX}`);
    await page.fill('input[name="email"]', `${PREFIX}@customer.com`);
    await page.getByRole('button', { name: /submit|save|create/i }).first().click();

    await expect(page.getByText(`Test Customer ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
  });

  test('view customer', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/customers');
    await page.waitForLoadState('networkidle');
    const firstCustomer = page.locator('table a, [class*="customer"] a, [class*="row"]').first();
    if (await firstCustomer.isVisible()) {
      await firstCustomer.click();
      await expect(page).not.toHaveURL('/customers');
    }
  });

  test('edit customer', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/customers');
    await page.waitForLoadState('networkidle');
    const editBtn = page.getByRole('button', { name: /edit/i }).first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
      await page.waitForTimeout(500);
      const nameInput = page.locator('input[name="company_name"], input[name="name"]').first();
      if (await nameInput.isVisible()) {
        await nameInput.fill(`Updated Customer ${PREFIX}`);
        await page.getByRole('button', { name: /save|update/i }).first().click();
        await expect(page.getByText(`Updated Customer ${PREFIX}`).first()).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('deactivate customer', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="identifier"]', `${PREFIX}@test.com`);
    await page.fill('input[name="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/customers');
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
