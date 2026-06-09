import { test, expect } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

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
    // Registration doesn't auto-login on the backend (no cookies),
    // but the frontend caches the user in query cache and navigates to /
    // From there the auth guard passes (cached data) — showing the home page
    await expect(page).toHaveURL('/', { timeout: 10000 });
  });

  test('register validation - show errors for empty form', async ({ page }) => {
    await page.goto('/register');
    await page.click('button[type="submit"]');
    await expect(page.getByText('Invalid email address')).toBeVisible();
    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible();
  });

  test('login with valid credentials', async ({ page }) => {
    const email = `logintest_${Date.now()}@example.com`;
    const username = `loginuser_${Date.now()}`;
    const password = 'password123';
    const orgName = `LoginOrg_${Date.now()}`;

    await page.goto('/register');
    await page.fill('input[name="email"]', email);
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="firstname"]', 'Test');
    await page.fill('input[name="lastname"]', 'User');
    await page.fill('input[name="password"]', password);
    await page.fill('input[name="organization_name"]', orgName);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    // Logout first to clear the cached auth state
    await page.evaluate(() => fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {}));
    await page.goto('/time-entries');
    await page.waitForURL(/\/login/, { timeout: 10000 });

    // Now test login with the registered user
    await page.goto('/login');
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
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
    const email = `logouttest_${Date.now()}@example.com`;
    const password = 'password123';

    // Register and login via browser
    await page.goto('/register');
    await page.fill('input[name="email"]', email);
    await page.fill('input[name="username"]', `logoutuser_${Date.now()}`);
    await page.fill('input[name="firstname"]', 'Test');
    await page.fill('input[name="lastname"]', 'User');
    await page.fill('input[name="password"]', password);
    await page.fill('input[name="organization_name"]', `LogoutOrg_${Date.now()}`);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    // Logout via API
    await page.evaluate(() => fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {}));

    // Navigate to protected route -> should redirect to login
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('protected route redirects to login', async ({ page }) => {
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});
