import { test, expect } from '@playwright/test';

test.describe('Auth Flow', () => {
  test('register with new organization', async ({ page }) => {
    await page.goto('/register');

    await page.fill('input[name="email"]', `test_${Date.now()}@example.com`);
    await page.fill('input[name="username"]', `user_${Date.now()}`);
    await page.fill('input[name="firstname"]', 'Test');
    await page.fill('input[name="lastname"]', 'User');
    await page.fill('input[name="password"]', 'password123');
    await page.fill('input[name="organization_name"]', `Org_${Date.now()}`);

    await page.click('button[type="submit"]');

    // Registration doesn't auto-login — user is redirected to login page
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('register validation - show errors for empty form', async ({ page }) => {
    await page.goto('/register');

    await page.click('button[type="submit"]');

    // Actual Zod validation messages from the form schema
    await expect(page.getByText('Invalid email address')).toBeVisible();
    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible();
  });

  test('login with valid credentials', async ({ page, request }) => {
    const email = `logintest_${Date.now()}@example.com`;
    const username = `loginuser_${Date.now()}`;
    const password = 'password123';
    const orgName = `LoginOrg_${Date.now()}`;

    const registerResponse = await request.post('http://localhost:8080/auth/register', {
      data: { email, username, password, firstname: 'Test', lastname: 'User', organization_name: orgName },
    });
    expect(registerResponse.status()).toBe(201);

    await page.goto('/login');
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');

    // Login redirects to home page (/)
    await expect(page).toHaveURL('/', { timeout: 10000 });
  });

  test('login with invalid credentials returns API error', async ({ request }) => {
    const res = await request.post('http://localhost:8080/auth/login', {
      data: { identifier: 'nonexistent@example.com', password: 'wrongpassword' },
    });
    expect(res.status()).toBe(401);
    const body = await res.json();
    expect(body.error).toMatch(/invalid credentials/i);
  });

  test('logout redirects to login', async ({ page }) => {
    // Register and login via browser to set auth cookies
    const email = `logouttest_${Date.now()}@example.com`;
    const password = 'password123';

    await page.goto('/register');
    await page.fill('input[name="email"]', email);
    await page.fill('input[name="username"]', `logoutuser_${Date.now()}`);
    await page.fill('input[name="firstname"]', 'Test');
    await page.fill('input[name="lastname"]', 'User');
    await page.fill('input[name="password"]', password);
    await page.fill('input[name="organization_name"]', `LogoutOrg_${Date.now()}`);
    await page.click('button[type="submit"]');

    // After register, user is redirected to /login — now log in
    await page.waitForURL(/\/login/, { timeout: 10000 });
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    // Now logout via the browser API call — same context
    await page.evaluate(async () => {
      await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' });
    });

    // Navigate to protected route — should redirect to login
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('protected route redirects to login', async ({ page }) => {
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});
