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

    // Registration doesn't auto-login — redirects to login page
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('register validation - show errors for empty form', async ({ page }) => {
    await page.goto('/register');

    await page.click('button[type="submit"]');

    await expect(page.getByText('Invalid email address')).toBeVisible();
    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible();
  });

  test('login with valid credentials', async ({ page, request }) => {
    const email = `test_${Date.now()}@example.com`;
    const username = `user_${Date.now()}`;
    const password = 'password123';
    const orgName = `Org_${Date.now()}`;

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

  test('logout redirects to login', async ({ page, request }) => {
    const email = `test_${Date.now()}@example.com`;
    const password = 'password123';

    const registerRes = await request.post('http://localhost:8080/auth/register', {
      data: { email, username: `user_${Date.now()}`, password, firstname: 'Test', lastname: 'User', organization_name: `Org_${Date.now()}` },
    });
    expect(registerRes.status()).toBe(201);

    // Login via browser to set cookies in the page context
    await page.goto('/login');
    await page.fill('input[name="identifier"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForURL('/', { timeout: 10000 });

    // Logout via API — this clears the auth cookies
    const logoutRes = await request.post('http://localhost:8080/auth/logout');
    expect(logoutRes.status()).toBe(200);

    // Navigate to a protected route — should redirect to login
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('protected route redirects to login', async ({ page }) => {
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});
