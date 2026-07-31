import { test, expect, type BrowserContext } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

// The backend rate-limits anonymous requests (outer 20/min across all routes;
// /auth/register + /auth/login additionally share a 5/min default — raise
// RATE_LIMIT for e2e runs). The app hydrates auth on protected page loads, and
// a missing session burns 2 anonymous calls (/auth/me 401 + failed refresh).
// To stay inside the budget we register + login ONCE via the API in
// beforeAll and inject the session cookies into each test's context, exactly
// like the customers suite.
async function sessionCookies(request: import('@playwright/test').APIRequestContext) {
  const cookies: Array<{ name: string; value: string }> = [];
  for (const h of (await request.post('http://localhost:8080/auth/login', {
    data: { identifier: EMAIL, password: PASSWORD },
  })).headersArray()) {
    if (h.name.toLowerCase() === 'set-cookie') {
      const [pair] = h.value.split(';');
      const eq = pair.indexOf('=');
      if (eq > 0) cookies.push({ name: pair.slice(0, eq), value: pair.slice(eq + 1) });
    }
  }
  return cookies;
}

async function useSession(context: BrowserContext, cookies: Array<{ name: string; value: string }>) {
  await context.addCookies(cookies.map((c) => ({ name: c.name, value: c.value, url: 'http://localhost:3000' })));
}

// Today landing contract (Plan 10-04): after login/register the app lands on
// `/` rendering the Today heading — never a redirect, never blank. The body
// must show the week section or a locked empty state.
async function expectTodayLanding(page: import('@playwright/test').Page) {
  await expect(page.getByRole('heading', { name: 'Today' })).toBeVisible({ timeout: 10000 });
  await expect(page.locator('body')).toContainText(
    /Your week|You're all caught up|Welcome to Hourglass/
  );
}

const PREFIX = `auth_${Date.now()}`;
const EMAIL = `${PREFIX}@example.com`;
const PASSWORD = 'password123';

test.describe('Auth Flow', () => {
  let sharedCookies: Array<{ name: string; value: string }> = [];

  test.beforeAll(async ({ request }) => {
    const reg = await request.post('http://localhost:8080/auth/register', {
      data: {
        email: EMAIL,
        username: `${PREFIX}_user`,
        password: PASSWORD,
        firstname: 'Test',
        lastname: 'User',
        organization_name: `${PREFIX}_org`,
      },
    });
    expect(reg.status()).toBe(200);
    sharedCookies = await sessionCookies(request);
    expect(sharedCookies.length).toBeGreaterThanOrEqual(2);
  });

  test('register with new organization', async ({ page }) => {
    await page.goto('/register');
    await page.fill('input[name="email"]', `test_${Date.now()}@example.com`);
    await page.fill('input[name="username"]', `user_${Date.now()}`);
    await page.fill('input[name="firstname"]', 'Test');
    await page.fill('input[name="lastname"]', 'User');
    await page.fill('input[name="password"]', 'password123');
    await page.fill('input[name="organization_name"]', `Org_${Date.now()}`);
    await page.click('button[type="submit"]');
    // After registration, the frontend caches auth data and lands on the
    // authenticated home — the Today landing page (Plan 10-04), not a
    // redirect to /time-entries.
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
    await expectTodayLanding(page);
  });

  test('register validation - show errors for empty form', async ({ page }) => {
    await page.goto('/register');
    await page.click('button[type="submit"]');
    await expect(page.getByText('Invalid email address')).toBeVisible();
    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible();
  });

  test('login with valid credentials', async ({ page }) => {
    // Log in via the browser form as the beforeAll-registered user.
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[name="identifier"]', EMAIL);
    await page.fill('input[name="password"]', PASSWORD);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
    await expectTodayLanding(page);
  });

  test('time-entries remains directly reachable (Track group surface intact)', async ({
    page,
    context,
  }) => {
    await useSession(context, sharedCookies);
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/time-entries/, { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Time' })).toBeVisible();
  });

  test('login with invalid credentials returns API error', async ({ request }) => {
    const res = await request.post('http://localhost:8080/auth/login', {
      data: { identifier: 'nonexistent@example.com', password: 'wrongpassword' },
    });
    expect(res.status()).toBe(401);
    const body = await res.json();
    expect(body.error).toMatch(/invalid credentials/i);
  });

  test('logout redirects to login', async ({ page, context }) => {
    await useSession(context, sharedCookies);
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

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

  test('expired access token recovers via silent refresh and rotates the refresh cookie', async ({
    page,
    request,
    context,
  }) => {
    // Fresh user so the refresh/replay below does not affect shared sessions.
    const email = `rotate_${Date.now()}@example.com`;
    const register = await request.post('http://localhost:8080/auth/register', {
      data: {
        email,
        username: `rotateuser_${Date.now()}`,
        password: PASSWORD,
        firstname: 'Test',
        lastname: 'User',
        organization_name: `RotateOrg_${Date.now()}`,
      },
    });
    expect(register.status()).toBe(200);

    const login = await request.post('http://localhost:8080/auth/login', {
      data: { identifier: email, password: PASSWORD },
    });
    expect(login.status()).toBe(200);
    const loginBody = await login.json();
    const refreshBefore = loginBody.data.refresh_token as string;
    expect(refreshBefore).toBeTruthy();

    // Seed the browser with the session, then corrupt the access token so the
    // next API call 401s and the client's refresh-on-401 path kicks in.
    await context.addCookies([
      { name: 'auth_token', value: loginBody.data.token, url: 'http://localhost:3000' },
      { name: 'refresh_token', value: refreshBefore, url: 'http://localhost:3000' },
      { name: 'auth_token', value: 'expired.garbage.token', url: 'http://localhost:3000' },
    ]);

    // /time-entries is protected: /auth/me 401s, the client silently refreshes
    // (rotating the refresh token) and the session continues — no re-login.
    await page.goto('/time-entries');
    await expect(page).toHaveURL(/\/time-entries/, { timeout: 15000 });

    // The guard's silent refresh lands a moment AFTER the navigation resolves
    // (goto returns at the load event, before the router's fetch completes).
    // Poll for the rotated cookie instead of reading it at an arbitrary
    // instant — the rotation must be observable in the browser.
    await expect
      .poll(
        async () =>
          (await context.cookies('http://localhost:3000')).find(
            (c) => c.name === 'refresh_token'
          )?.value,
        { timeout: 15000 }
      )
      .not.toBe(refreshBefore);
    const refreshAfter = (await context.cookies('http://localhost:3000')).find(
      (c) => c.name === 'refresh_token'
    )?.value;
    expect(refreshAfter).toBeTruthy();

    // Replay of the pre-rotation refresh cookie is detected as reuse: the
    // backend answers 401 (and clears the session cookies).
    const replay = await request.post('http://localhost:8080/auth/refresh', {
      headers: { cookie: `refresh_token=${refreshBefore}` },
    });
    expect(replay.status()).toBe(401);
  });
});
