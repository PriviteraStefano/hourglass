import { test, expect } from '@playwright/test';

test.describe('Auth Flow', () => {
  test('register with new organization', async ({ page }) => {
    await page.goto('/register');

    await page.fill('input[name="email"]', `test_${Date.now()}@example.com`);
    await page.fill('input[name="username"]', `user_${Date.now()}`);
    await page.fill('input[name="firstname"]', 'Test');
    await page.fill('input[name="lastname"]', 'User');
    await page.fill('input[name="password"]', 'password123');
    await page.fill('input[name="confirmPassword"]', 'password123');
    await page.fill('input[name="organization_name"]', `Org_${Date.now()}`);

    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('register validation - show errors for empty form', async ({ page }) => {
    await page.goto('/register');

    await page.click('button[type="submit"]');

    await expect(page.getByText('Email is required')).toBeVisible();
    await expect(page.getByText('Password is required')).toBeVisible();
  });

  test('login with valid credentials', async ({ page, request }) => {
    const email = `test_${Date.now()}@example.com`;
    const username = `user_${Date.now()}`;
    const password = 'password123';
    const orgName = `Org_${Date.now()}`;

    const registerResponse = await request.post('http://localhost:8080/auth/register', {
      data: {
        email,
        username,
        password,
        organization_name: orgName,
      },
    });
    expect(registerResponse.status()).toBe(201);

    await page.goto('/login');
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/(dashboard|time-entries)/, { timeout: 10000 });
  });

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[name="identifier"]', 'nonexistent@example.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    await expect(page.getByText(/invalid credentials/i)).toBeVisible();
  });

  test('logout redirects to login', async ({ page, request }) => {
    const email = `test_${Date.now()}@example.com`;
    const password = 'password123';

    await request.post('http://localhost:8080/auth/register', {
      data: {
        email,
        username: `user_${Date.now()}`,
        password,
        organization_name: `Org_${Date.now()}`,
      },
    });

    await page.goto('/login');
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');

    await page.waitForURL(/\/(dashboard|time-entries)/);

    await page.click('button:has-text("Logout")');

    await expect(page).toHaveURL('/login', { timeout: 10000 });
  });

  test('protected route redirects to login', async ({ page }) => {
    await page.goto('/dashboard');

    await expect(page).toHaveURL(/\/login/);
  });
});