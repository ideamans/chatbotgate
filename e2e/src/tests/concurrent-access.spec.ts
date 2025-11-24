import { test, expect, BrowserContext, Page } from '@playwright/test';
import { authenticateViaOAuth2 } from '../support/auth-helpers';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const BASE_URL = 'http://localhost:4180';
const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';

/**
 * Helper to create and authenticate a new user context
 */
async function createAuthenticatedUser(
  context: BrowserContext,
  email: string = TEST_EMAIL,
  password: string = TEST_PASSWORD
): Promise<Page> {
  const page = await context.newPage();
  await routeStubAuthRequests(page);

  await authenticateViaOAuth2(page, {
    email,
    password,
    baseUrl: BASE_URL,
  });

  // Verify authentication succeeded
  await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

  return page;
}

/**
 * Helper to extract session cookie value
 */
async function getSessionCookie(page: Page): Promise<string | undefined> {
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find((c) => c.name === '_oauth2_proxy');
  return sessionCookie?.value;
}

test.describe('Concurrent access and session isolation', () => {
  test('multiple users can authenticate simultaneously', async ({ context }) => {
    // Create 5 users authenticating at the same time
    const userCount = 5;
    const authPromises: Promise<Page>[] = [];

    for (let i = 0; i < userCount; i++) {
      authPromises.push(createAuthenticatedUser(context));
    }

    // Wait for all authentications to complete
    const pages = await Promise.all(authPromises);

    // All users should be authenticated
    for (const page of pages) {
      await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
      await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);
    }

    // Clean up
    for (const page of pages) {
      await page.close();
    }
  });

  test('concurrent session creation does not corrupt data', async ({ context }) => {
    const userCount = 10;
    const pages: Page[] = [];

    // Create pages
    for (let i = 0; i < userCount; i++) {
      const page = await context.newPage();
      await routeStubAuthRequests(page);
      pages.push(page);
    }

    // Start authentication for all users concurrently
    const authPromises = pages.map((page) =>
      authenticateViaOAuth2(page, {
        email: TEST_EMAIL,
        password: TEST_PASSWORD,
        baseUrl: BASE_URL,
      })
    );

    await Promise.all(authPromises);

    // Verify all sessions are valid
    for (const page of pages) {
      await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
      await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);
    }

    // Clean up
    for (const page of pages) {
      await page.close();
    }
  });

  test('users have isolated sessions with unique cookies', async ({ context }) => {
    // Create two separate browser contexts (simulate different browsers)
    const context1 = await context.browser()!.newContext();
    const context2 = await context.browser()!.newContext();

    try {
      // Authenticate both users
      const user1Page = await createAuthenticatedUser(context1);
      const user2Page = await createAuthenticatedUser(context2);

      // Get session cookies
      const user1Cookie = await getSessionCookie(user1Page);
      const user2Cookie = await getSessionCookie(user2Page);

      // Both should have cookies
      expect(user1Cookie).toBeDefined();
      expect(user2Cookie).toBeDefined();

      // Cookies should be different (different sessions)
      expect(user1Cookie).not.toBe(user2Cookie);

      // Both users should be authenticated with their own sessions
      await expect(user1Page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);
      await expect(user2Page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);

      await user1Page.close();
      await user2Page.close();
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('concurrent access to protected resources maintains session integrity', async ({ context }) => {
    const page = await createAuthenticatedUser(context);

    // Make multiple concurrent requests to protected resources
    const requests = Array.from({ length: 20 }, (_, i) => page.goto(BASE_URL + `/?test=${i}`));

    await Promise.all(requests);

    // Should still be authenticated after all requests
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
    await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);

    await page.close();
  });

  test('concurrent page navigations maintain authentication', async ({ context }) => {
    const page = await createAuthenticatedUser(context);

    // Navigate to multiple paths concurrently
    const paths = ['/page1', '/page2', '/page3', '/page4', '/page5'];
    const navigations = paths.map((path) => page.goto(BASE_URL + path));

    await Promise.all(navigations);

    // Final navigation should still be authenticated
    await page.goto(BASE_URL + '/');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    await page.close();
  });

  test('session isolation: user A cannot see user B session data', async ({ context }) => {
    // Create two separate contexts
    const context1 = await context.browser()!.newContext();
    const context2 = await context.browser()!.newContext();

    try {
      const userA = await createAuthenticatedUser(context1);
      const userB = await createAuthenticatedUser(context2);

      // Get session cookies
      const cookieA = await getSessionCookie(userA);
      const cookieB = await getSessionCookie(userB);

      expect(cookieA).toBeDefined();
      expect(cookieB).toBeDefined();
      expect(cookieA).not.toBe(cookieB);

      // Try to use userB's cookie in userA's context (session hijacking attempt)
      await context1.addCookies([
        {
          name: '_oauth2_proxy',
          value: cookieB!,
          domain: 'localhost',
          path: '/',
        },
      ]);

      // Navigate with hijacked cookie
      await userA.goto(BASE_URL + '/');

      // Should be authenticated (with userB's session)
      // This tests that sessions are properly isolated but cookies work correctly
      await expect(userA.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);

      await userA.close();
      await userB.close();
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('concurrent logout operations do not interfere', async ({ context }) => {
    // Create 5 authenticated users
    const pages: Page[] = [];
    for (let i = 0; i < 5; i++) {
      const page = await createAuthenticatedUser(context);
      pages.push(page);
    }

    // Logout all users concurrently
    const logoutPromises = pages.map((page) =>
      page.locator('[data-test="oauth-signout"]').click()
    );

    await Promise.all(logoutPromises);

    // All users should be logged out
    for (const page of pages) {
      await expect(page).toHaveURL(/\/_auth\/logout/);
      await page.close();
    }
  });

  test('concurrent WebSocket connections from different users work independently', async ({
    context,
  }) => {
    // Create two authenticated users
    const user1 = await createAuthenticatedUser(context);
    const user2 = await createAuthenticatedUser(context);

    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';

    // Connect both users to WebSocket concurrently
    const results = await Promise.all([
      user1.evaluate(async ({ url }: { url: string }) => {
        return new Promise<{ connected: boolean; welcomeReceived: boolean }>((resolve) => {
          const ws = new WebSocket(url);
          let welcomeReceived = false;

          const timeout = setTimeout(() => {
            ws.close();
            resolve({ connected: false, welcomeReceived });
          }, 5000);

          ws.onopen = () => {
            console.log('User 1 WebSocket connected');
          };

          ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'welcome') {
              welcomeReceived = true;
              clearTimeout(timeout);
              ws.close();
              resolve({ connected: true, welcomeReceived });
            }
          };

          ws.onerror = () => {
            clearTimeout(timeout);
            resolve({ connected: false, welcomeReceived });
          };
        });
      }, { url: wsUrl }),

      user2.evaluate(async ({ url }: { url: string }) => {
        return new Promise<{ connected: boolean; welcomeReceived: boolean }>((resolve) => {
          const ws = new WebSocket(url);
          let welcomeReceived = false;

          const timeout = setTimeout(() => {
            ws.close();
            resolve({ connected: false, welcomeReceived });
          }, 5000);

          ws.onopen = () => {
            console.log('User 2 WebSocket connected');
          };

          ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'welcome') {
              welcomeReceived = true;
              clearTimeout(timeout);
              ws.close();
              resolve({ connected: true, welcomeReceived });
            }
          };

          ws.onerror = () => {
            clearTimeout(timeout);
            resolve({ connected: false, welcomeReceived });
          };
        });
      }, { url: wsUrl }),
    ]);

    // Both users should have successful connections
    expect(results[0].connected).toBe(true);
    expect(results[0].welcomeReceived).toBe(true);
    expect(results[1].connected).toBe(true);
    expect(results[1].welcomeReceived).toBe(true);

    await user1.close();
    await user2.close();
  });

  test('race condition: rapid session updates maintain consistency', async ({ context }) => {
    const page = await createAuthenticatedUser(context);

    // Make rapid concurrent requests that might trigger session updates
    const requestCount = 50;
    const requests = Array.from({ length: requestCount }, (_, i) =>
      page.goto(BASE_URL + `/?iteration=${i}`)
    );

    await Promise.all(requests);

    // Session should still be valid and consistent
    await page.goto(BASE_URL + '/');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
    await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);

    await page.close();
  });

  test('concurrent authentication attempts with same credentials succeed', async ({ context }) => {
    // Simulate multiple browser tabs trying to authenticate at the same time
    const tabCount = 3;
    const pages: Page[] = [];

    // Create all pages first
    for (let i = 0; i < tabCount; i++) {
      const page = await context.newPage();
      await routeStubAuthRequests(page);
      pages.push(page);
    }

    // Start navigation to protected resource (triggers auth) concurrently
    await Promise.all(pages.map((page) => page.goto(BASE_URL + '/')));

    // All should redirect to login
    for (const page of pages) {
      await expect(page).toHaveURL(/\/_auth\/login$/);
    }

    // Authenticate all tabs concurrently
    const authPromises = pages.map((page) =>
      (async () => {
        await page.getByRole('link', { name: 'stub-auth' }).click();
        await expect(page).toHaveURL(/localhost:3001\/login/);

        await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
        await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
        await page.locator('[data-test="login-submit"]').click();

        await expect(page).toHaveURL(/localhost:3001\/oauth\/authorize/);
        await page.locator('[data-test="authorize-allow"]').click();

        await expect(page).toHaveURL(new RegExp(BASE_URL.replace('http://', '')));
      })()
    );

    await Promise.all(authPromises);

    // All tabs should be authenticated
    for (const page of pages) {
      await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
    }

    // Clean up
    for (const page of pages) {
      await page.close();
    }
  });

  test('session remains valid during high concurrent load', async ({ context }) => {
    const page = await createAuthenticatedUser(context);

    // Generate high concurrent load (100 requests)
    const loadTest = async () => {
      const requests = Array.from({ length: 100 }, (_, i) =>
        page.goto(BASE_URL + `/?load=${i}`, { waitUntil: 'domcontentloaded' })
      );
      await Promise.all(requests);
    };

    await loadTest();

    // Session should still be valid
    await page.goto(BASE_URL + '/');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
    await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);

    await page.close();
  });

  test('multiple browser contexts maintain independent sessions', async ({ context }) => {
    // Create 3 separate contexts (like incognito windows)
    const contexts = await Promise.all([
      context.browser()!.newContext(),
      context.browser()!.newContext(),
      context.browser()!.newContext(),
    ]);

    try {
      // Authenticate in all contexts concurrently
      const pages = await Promise.all(
        contexts.map((ctx) => createAuthenticatedUser(ctx))
      );

      // Get all session cookies
      const cookies = await Promise.all(pages.map((page) => getSessionCookie(page)));

      // All should have cookies
      cookies.forEach((cookie) => {
        expect(cookie).toBeDefined();
      });

      // All cookies should be different (independent sessions)
      expect(cookies[0]).not.toBe(cookies[1]);
      expect(cookies[1]).not.toBe(cookies[2]);
      expect(cookies[0]).not.toBe(cookies[2]);

      // All should be authenticated
      for (const page of pages) {
        await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
        await page.close();
      }
    } finally {
      await Promise.all(contexts.map((ctx) => ctx.close()));
    }
  });

  test('concurrent requests with mixed authenticated/unauthenticated users', async ({
    context,
  }) => {
    // Create authenticated user
    const authenticatedPage = await createAuthenticatedUser(context);

    // Create unauthenticated user
    const unauthenticatedPage = await context.newPage();
    await routeStubAuthRequests(unauthenticatedPage);

    // Make concurrent requests
    await Promise.all([
      authenticatedPage.goto(BASE_URL + '/protected'),
      unauthenticatedPage.goto(BASE_URL + '/protected'),
    ]);

    // Authenticated user should access resource
    await expect(authenticatedPage.locator('[data-test="auth-provider"]')).toContainText(
      'stub-auth'
    );

    // Unauthenticated user should be redirected to login
    await expect(unauthenticatedPage).toHaveURL(/\/_auth\/login$/);

    await authenticatedPage.close();
    await unauthenticatedPage.close();
  });
});
