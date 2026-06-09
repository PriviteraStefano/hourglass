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
    // After registration, frontend caches auth data and navigates to /
    await expect(page).toHaveURL('/', { timeout: 10000 });
  });

  test('register validation - show errors for empty form', async ({ page }) => {
    await page.goto('/register');
    await page.click('button[type="submit"]');
    await expect(page.getByText('Invalid email address')).toBeVisible();
    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible();
  });

  test('login with valid credentials', async ({ page, request }) => {
    // Register via API (clean — no browser cookies set)
    const email = `logintest_${Date.now()}@example.com`;
    const password = 'password123';
    const res = await request.post('http://localhost:8080/auth/register', {
      data: {
        email,
        username: `loginuser_${Date.now()}`,
        password,
        firstname: 'Test',
        lastname: 'User',
        organization_name: `LoginOrg_${Date.now()}`,
      },
    });
    expect(res.status()).toBe(201);

    // Now log in via browser form
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
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

  test('logout redirects to login', async ({ page, request }) => {
    // Register via API
    const email = `logouttest_${Date.now()}@example.com`;
    await request.post('http://localhost:8080/auth/register', {
      data: {
        email,
        username: `logoutuser_${Date.now()}`,
        password: 'password123',
        firstname: 'Test',
        lastname: 'User',
        organization_name: `LogoutOrg_${Date.now()}`,
      },
    });

    // Login via browser to get cookies
    await page.goto('/login');
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    // Logout via API fetch from page context (this clears cookies in the browser)
    await page.evaluate(() =>
      fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {})
    );

    // Navigate to protected route with waitUntil: 'commit' to avoid navigation conflicts
    await page.goto('/time-entries', { waitUntil: 'commit' });
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('protected route redirects to login', async ({ page }) => {
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});
