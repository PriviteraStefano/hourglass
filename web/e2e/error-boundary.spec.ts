import { test, expect } from '@playwright/test';
import {
  registerUser,
  fetchIds,
  loginOnce,
  useSession,
  seedBaseEntities,
  seedTimeEntries,
} from './helpers';

test.describe.configure({ mode: 'serial' });

// P0-4 verification: a failed list loader renders the recoverable RouteError
// panel (never a blank screen), "Try again" re-runs the loader and recovers to
// data, and the home link navigates to the landing route. The slim auth-page
// variant is asserted at the component level (AuthRouteError test in
// route-error.test.tsx) plus a browser-level no-blank-screen check on /login —
// the _auth layout swallows request failures by design, so a failed /auth/me
// must never blank-screen the login page.
test.describe('Error Boundary (P0-4)', () => {
  let sessionCookies: Array<{ name: string; value: string }> = [];
  const P = `errbd_${Date.now()}`;
  const SEEDED_DRAFT = `seeded-draft-${P}`;
  const OUTAGE = 'simulated outage';

  test.beforeAll(async ({ request }) => {
    const { email, password } = await registerUser(request, 'errbd');
    const { userId, orgId } = fetchIds(email);
    const base = seedBaseEntities(orgId, userId);
    seedTimeEntries(orgId, userId, base, P);
    sessionCookies = await loginOnce(request, email, password);
  });

  // Anchored to the real API path — a glob like "**/api/time-entries**" would
  // ALSO match the Vite module URL (src/api/time-entries.ts) and 500 it,
  // breaking app boot entirely.
  const TIME_ENTRIES_API = /\/api\/time-entries(\?|$)/;

  async function failTimeEntries(page: import('@playwright/test').Page) {
    await page.route(TIME_ENTRIES_API, (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: OUTAGE }),
      })
    );
  }

  test('failed list loader renders the error panel, not a blank screen', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await failTimeEntries(page);
    await page.goto('/time-entries');
    await page.waitForLoadState('networkidle');

    // No blank screen at any point: the body renders content before the panel
    // assertion (auto-retrying — the React root mounts after the load event)
    await expect(page.locator('body')).not.toBeEmpty({ timeout: 15000 });
    const bodyText = await page.locator('body').innerText();
    expect(bodyText.trim().length).toBeGreaterThan(0);

    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.getByText('Something went wrong')).toBeVisible();
    await expect(page.getByText(OUTAGE)).toBeVisible();
    await expect(page.getByRole('button', { name: /try again/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /go to today/i })).toBeVisible();
  });

  test('Try again re-runs the loader and recovers to data', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await failTimeEntries(page);
    await page.goto('/time-entries');
    await expect(page.getByRole('alert')).toBeVisible({ timeout: 15000 });

    // API recovers; the router invalidate re-runs the loader
    await page.unroute(TIME_ENTRIES_API);
    await page.getByRole('button', { name: /try again/i }).click();

    await expect(page.getByRole('alert')).not.toBeVisible({ timeout: 15000 });
    await expect(page.getByText(SEEDED_DRAFT)).toBeVisible({ timeout: 15000 });
  });

  test('home link navigates to the landing route', async ({ page }) => {
    await useSession(page.context(), sessionCookies);
    await failTimeEntries(page);
    await page.goto('/time-entries');
    await expect(page.getByRole('alert')).toBeVisible({ timeout: 15000 });

    // Let the API recover, then follow the home link to the landing route
    await page.unroute(TIME_ENTRIES_API);
    await page.getByRole('link', { name: /go to today/i }).click();

    await expect(page).toHaveURL(/\/time-entries/, { timeout: 15000 });
    await expect(page.getByRole('alert')).not.toBeVisible({ timeout: 15000 });
    await expect(page.getByText(SEEDED_DRAFT)).toBeVisible({ timeout: 15000 });
  });

  test('auth pages never blank-screen when the auth API fails (slim variant)', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    await page.route('**/api/auth/me**', (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'auth outage' }),
      })
    );
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('body')).not.toBeEmpty({ timeout: 15000 });
    const bodyText = await page.locator('body').innerText();
    expect(bodyText.trim().length).toBeGreaterThan(0);
    await expect(page.getByRole('button', { name: /log in/i })).toBeVisible();
    await context.close();
  });
});
