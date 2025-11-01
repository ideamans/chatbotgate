import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';

test.describe('Authentication security', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('should reject CSRF attack with mismatched state', async ({ page }) => {
    // Set oauth_state cookie with one value
    await page.context().addCookies([{
      name: 'oauth_state',
      value: 'legitimate-state-12345',
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      sameSite: 'Lax'
    }, {
      name: 'oauth_provider',
      value: 'stub-auth',
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      sameSite: 'Lax'
    }]);

    // Try to access callback with different state (CSRF attack simulation)
    await page.goto('/_auth/oauth2/callback?state=attacker-state-99999&code=valid-code');

    // Should reject with "Invalid state" error
    await expect(page.locator('body')).toContainText(/invalid state/i);
  });

  test('should reject invalid email token', async ({ page }) => {
    // Try to access verify endpoint with invalid token
    await page.goto('/_auth/email/verify?token=invalid-token-12345');

    // Should show error message
    await expect(page.locator('body')).toContainText('Invalid or Expired Token');
  });

  test('should reject empty email token', async ({ page }) => {
    // Try to access verify endpoint without token
    await page.goto('/_auth/email/verify');

    // Should show error (400 Bad Request or similar)
    await expect(page.locator('body')).toContainText(/invalid|error|bad request/i);
  });

  test('should handle OAuth2 callback without state', async ({ page }) => {
    // Try to access callback without state parameter
    await page.goto('/_auth/oauth2/callback?code=test-code');

    // Should show error
    await expect(page.locator('body')).toContainText(/invalid|error/i);
  });

  test('should handle OAuth2 callback without code', async ({ page }) => {
    // Try to access callback without code parameter
    await page.goto('/_auth/oauth2/callback?state=test-state');

    // Should show error
    await expect(page.locator('body')).toContainText(/invalid|error|not found/i);
  });

  test('should prevent open redirect attacks', async ({ page }) => {
    // Test 1: Attempt to redirect to external URL with absolute URL
    await page.goto('/_auth/login');

    // Manually set a malicious redirect cookie
    await page.context().addCookies([{
      name: '_oauth2_redirect',
      value: 'https://evil.com/steal-session',
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      sameSite: 'Lax'
    }]);

    // Complete authentication
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill('someone@example.com');
    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill('password');

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();

    // Should redirect to home (/) instead of malicious URL
    await page.waitForURL(/localhost:4180\/?$/);
    await expect(page).toHaveURL(/localhost:4180\/?$/);
    await expect(page).not.toHaveURL(/evil\.com/);
  });

  test('should prevent protocol-relative URL redirects', async ({ page }) => {
    await page.goto('/_auth/login');

    // Set a protocol-relative redirect URL
    await page.context().addCookies([{
      name: '_oauth2_redirect',
      value: '//evil.com/phishing',
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      sameSite: 'Lax'
    }]);

    // Complete authentication
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill('someone@example.com');
    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill('password');

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();

    // Should redirect to home (/) instead of protocol-relative URL
    await page.waitForURL(/localhost:4180\/?$/);
    await expect(page).toHaveURL(/localhost:4180\/?$/);
  });

  test('should set HttpOnly flag on session cookie', async ({ page }) => {
    // Complete authentication to get a session cookie
    await page.goto('/');
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill(TEST_EMAIL);
    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();
    await page.waitForURL(/localhost:4180\/?$/);

    // Try to access session cookie from JavaScript (should fail due to HttpOnly)
    const canAccessCookie = await page.evaluate(() => {
      // Try to find session cookie in document.cookie
      const cookies = document.cookie;
      return cookies.includes('mop-e2e');
    });

    // Session cookie should NOT be accessible from JavaScript
    expect(canAccessCookie).toBe(false);
  });

  test('should invalidate session after logout', async ({ page }) => {
    // Authenticate
    await page.goto('/');
    await page.getByRole('link', { name: 'stub-auth' }).click();
    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill(TEST_EMAIL);
    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();
    await page.waitForURL(/localhost:4180\/?$/);

    // Verify authenticated
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);

    // Logout
    await Promise.all([
      page.waitForURL(/\/_auth\/logout/),
      page.locator('[data-test="oauth-signout"]').click(),
    ]);

    // Try to access protected resource - should redirect to login
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });
});
