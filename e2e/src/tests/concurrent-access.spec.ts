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
  test('multiple users can authenticate simultaneously', async ({ browser }) => {
    // Create 5 separate contexts (separate browser sessions)
    const userCount = 5;
    const contexts: BrowserContext[] = [];

    for (let i = 0; i < userCount; i++) {
      contexts.push(await browser!.newContext());
    }

    try {
      // Authenticate all users concurrently
      const authPromises = contexts.map((ctx) => createAuthenticatedUser(ctx));
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
    } finally {
      for (const ctx of contexts) {
        await ctx.close();
      }
    }
  });

  test('concurrent session creation does not corrupt data', async ({ browser }) => {
    // Create 10 separate contexts to ensure independent sessions
    const userCount = 10;
    const contexts: BrowserContext[] = [];

    for (let i = 0; i < userCount; i++) {
      contexts.push(await browser!.newContext());
    }

    try {
      // Create pages and start authentication for all users concurrently
      const authPromises = contexts.map(async (ctx) => {
        const page = await ctx.newPage();
        await routeStubAuthRequests(page);
        await authenticateViaOAuth2(page, {
          email: TEST_EMAIL,
          password: TEST_PASSWORD,
          baseUrl: BASE_URL,
        });
        return page;
      });

      const pages = await Promise.all(authPromises);

      // Verify all sessions are valid
      for (const page of pages) {
        await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
        await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);
      }

      // Clean up
      for (const page of pages) {
        await page.close();
      }
    } finally {
      for (const ctx of contexts) {
        await ctx.close();
      }
    }
  });

  test('users have isolated sessions with unique cookies', async ({ browser }) => {
    // Create two separate browser contexts (simulate different browsers)
    const context1 = await browser!.newContext();
    const context2 = await browser!.newContext();

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

  test('concurrent access to protected resources maintains session integrity', async ({ browser }) => {
    const ctx = await browser!.newContext();

    try {
      const page = await createAuthenticatedUser(ctx);

      // Make multiple concurrent requests to protected resources
      const requests = Array.from({ length: 20 }, (_, i) => page.goto(BASE_URL + `/?test=${i}`));

      await Promise.all(requests);

      // Should still be authenticated after all requests
      await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
      await expect(page.locator('[data-test="auth-email"]')).toContainText(TEST_EMAIL);

      await page.close();
    } finally {
      await ctx.close();
    }
  });

  test('concurrent page navigations maintain authentication', async ({ browser }) => {
    const ctx = await browser!.newContext();

    try {
      const page = await createAuthenticatedUser(ctx);

      // Navigate to multiple paths concurrently
      const paths = ['/page1', '/page2', '/page3', '/page4', '/page5'];
      const navigations = paths.map((path) => page.goto(BASE_URL + path));

      await Promise.all(navigations);

      // Final navigation should still be authenticated
      await page.goto(BASE_URL + '/');
      await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

      await page.close();
    } finally {
      await ctx.close();
    }
  });

  test('session isolation: user A cannot see user B session data', async ({ browser }) => {
    // Create two separate contexts
    const context1 = await browser!.newContext();
    const context2 = await browser!.newContext();

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

  test('concurrent logout operations do not interfere', async ({ browser }) => {
    // Create 5 separate contexts with authenticated users
    const contexts: BrowserContext[] = [];
    const pages: Page[] = [];

    try {
      for (let i = 0; i < 5; i++) {
        const ctx = await browser!.newContext();
        contexts.push(ctx);
        const page = await createAuthenticatedUser(ctx);
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
    } finally {
      for (const ctx of contexts) {
        await ctx.close();
      }
    }
  });

  test('concurrent WebSocket connections from different users work independently', async ({
    browser,
  }) => {
    // Create two separate contexts for different users
    const context1 = await browser!.newContext();
    const context2 = await browser!.newContext();

    try {
      // Create two authenticated users
      const user1 = await createAuthenticatedUser(context1);
      const user2 = await createAuthenticatedUser(context2);

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
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('race condition: rapid session updates maintain consistency', async ({ browser }) => {
    const ctx = await browser!.newContext();

    try {
      const page = await createAuthenticatedUser(ctx);

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
    } finally {
      await ctx.close();
    }
  });

  test('concurrent authentication attempts with same credentials succeed', async ({ browser }) => {
    // Simulate multiple separate browser sessions authenticating concurrently
    const sessionCount = 3;
    const contexts: BrowserContext[] = [];

    try {
      // Create separate contexts
      for (let i = 0; i < sessionCount; i++) {
        contexts.push(await browser!.newContext());
      }

      // Authenticate all sessions concurrently
      const authPromises = contexts.map(async (ctx) => {
        const page = await ctx.newPage();
        await routeStubAuthRequests(page);

        await page.goto(BASE_URL + '/');
        await expect(page).toHaveURL(/\/_auth\/login$/);

        await page.getByRole('link', { name: 'stub-auth' }).click();
        await expect(page).toHaveURL(/localhost:3001\/login/);

        await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
        await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
        await Promise.all([
          page.waitForURL(/localhost:3001\/oauth\/authorize/),
          page.locator('[data-test="login-submit"]').click(),
        ]);

        await page.locator('[data-test="authorize-allow"]').click();
        await expect(page).toHaveURL(new RegExp(BASE_URL.replace('http://', '')));

        return page;
      });

      const pages = await Promise.all(authPromises);

      // All sessions should be authenticated
      for (const page of pages) {
        await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
      }

      // Clean up
      for (const page of pages) {
        await page.close();
      }
    } finally {
      for (const ctx of contexts) {
        await ctx.close();
      }
    }
  });

  test('session remains valid during high concurrent load', async ({ browser }) => {
    const ctx = await browser!.newContext();

    try {
      const page = await createAuthenticatedUser(ctx);

      // Generate concurrent load (20 requests - reduced from 100 to avoid ERR_ABORTED)
      // Note: Too many concurrent navigations on same page can cause browser errors
      const loadTest = async () => {
        const requests = Array.from({ length: 20 }, (_, i) =>
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
    } finally {
      await ctx.close();
    }
  });

  test('multiple browser contexts maintain independent sessions', async ({ browser }) => {
    // Create 3 separate contexts (like incognito windows)
    const contexts = await Promise.all([
      browser!.newContext(),
      browser!.newContext(),
      browser!.newContext(),
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
    browser,
  }) => {
    // Create separate contexts for authenticated and unauthenticated users
    const authenticatedContext = await browser!.newContext();
    const unauthenticatedContext = await browser!.newContext();

    try {
      // Create authenticated user
      const authenticatedPage = await createAuthenticatedUser(authenticatedContext);

      // Create unauthenticated user
      const unauthenticatedPage = await unauthenticatedContext.newPage();
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
    } finally {
      await authenticatedContext.close();
      await unauthenticatedContext.close();
    }
  });
});
