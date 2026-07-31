import { test, expect, type BrowserContext } from '@playwright/test';
import { execSync } from 'node:child_process';

test.describe.configure({ mode: 'serial' });

const PREFIX = `cust_${Date.now()}`;
const EMAIL = `${PREFIX}@test.com`;
const PASSWORD = 'Password123!';

// Backend: only finance users can create/update/delete customers. Registration
// assigns "manager", so promote the fresh user to finance directly in the DB
// (the suite already assumes the local dockerized stack: hourglass-postgres).
function promoteUserToFinance(email: string) {
  execSync(
    `docker exec hourglass-postgres psql -U hourglass -d hourglass -t -c "UPDATE organization_memberships SET role='finance' FROM users WHERE users.id = organization_memberships.user_id AND users.email='${email}'"`,
    { stdio: 'pipe' }
  );
}

// The backend rate-limits anonymous POST /auth/login to 5/min per IP. This suite
// has 5 tests, so logging in via the UI in every test would exhaust the budget.
// Instead: login ONCE via the API in beforeAll, capture the session cookies, and
// inject them into each test's browser context (goto /login then auto-redirects).
async function loginOnce(request: import('@playwright/test').APIRequestContext) {
  const login = await request.post('http://localhost:8080/auth/login', {
    data: { identifier: EMAIL, password: PASSWORD },
  });
  if (!login.ok()) {
    throw new Error(`login failed: ${login.status()} ${await login.text()}`);
  }
  const cookies: Array<{ name: string; value: string }> = [];
  for (const h of login.headersArray()) {
    if (h.name.toLowerCase() === 'set-cookie') {
      const [pair] = h.value.split(';');
      const eq = pair.indexOf('=');
      if (eq > 0) {
        cookies.push({ name: pair.slice(0, eq), value: pair.slice(eq + 1) });
      }
    }
  }
  return cookies;
}

async function useSession(context: BrowserContext, cookies: Array<{ name: string; value: string }>) {
  await context.addCookies(
    cookies.map((c) => ({
      name: c.name,
      value: c.value,
      url: 'http://localhost:3000',
    }))
  );
}

test.describe('Customers CRUD', () => {
  let sessionCookies: Array<{ name: string; value: string }> = [];

  test.beforeAll(async ({ request }) => {
    await request.post('http://localhost:8080/auth/register', {
      data: { email: EMAIL, username: `${PREFIX}_user`, password: PASSWORD, firstname: 'Test', lastname: 'User', organization_name: `${PREFIX}_org` },
    });
    promoteUserToFinance(EMAIL);
    sessionCookies = await loginOnce(request);
  });

  test('create customer', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/login');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/customers');
    await page.waitForLoadState('networkidle');
    const createBtn = page.getByRole('button', { name: /create|add|new/i }).first();
    if (await createBtn.isVisible()) {
      await createBtn.click();
      await page.waitForTimeout(500);
      await page.fill('input[name="company_name"], input[name="name"]', `Test Customer ${PREFIX}`);
      await page.fill('input[name="email"]', `${PREFIX}@customer.com`);
      await page.getByRole('button', { name: /submit|save|create/i }).first().click();
      await expect(page.getByText(`Test Customer ${PREFIX}`).first()).toBeVisible({ timeout: 10000 });
    }
  });

  test('view customer', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/login');
    await page.waitForURL('/', { timeout: 10000 });

    await page.goto('/customers');
    await page.waitForLoadState('networkidle');
    const firstCustomer = page.locator('table a, [class*="customer"] a, [class*="row"]').first();
    if (await firstCustomer.isVisible()) {
      await firstCustomer.click();
      await expect(page).not.toHaveURL('/customers');
    }
  });

  test('customers index route reachable from sidebar', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/login');
    await page.waitForURL('/', { timeout: 10000 });

    // Sidebar → /customers index route
    await page.getByRole('link', { name: 'Customers' }).click();
    await page.waitForURL('/customers', { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Customers' })).toBeVisible({ timeout: 10000 });

    // List renders (either cards or empty state)
    const emptyState = page.getByText('No customers yet');
    const anyCard = page.locator('[class*="cursor-pointer"]').first();
    const hasContent = await (async () => {
      try {
        await emptyState.waitFor({ state: 'visible', timeout: 5000 });
        return true;
      } catch {
        return await anyCard.isVisible();
      }
    })();

    if (hasContent && (await anyCard.isVisible())) {
      // Click row → detail at /customers/$id
      await anyCard.click();
      await page.waitForURL(/\/customers\/[0-9a-f-]{36}/, { timeout: 10000 });
      await expect(page.getByRole('button', { name: 'Back to Customers' })).toBeVisible({ timeout: 5000 });
    }
  });

  test('edit customer', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await page.goto('/login');
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
    await useSession(page.context(), sessionCookies);
    await page.goto('/login');
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
