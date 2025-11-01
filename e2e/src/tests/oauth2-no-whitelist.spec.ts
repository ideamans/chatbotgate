import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const NO_EMAIL_USER = 'noemail@example.com'; // Special user that doesn't provide email
const TEST_PASSWORD = 'password';

/**
 * This test suite verifies OAuth2 flow when NO whitelist is configured.
 * When no whitelist is set, email address is NOT required for authentication.
 * Users can authenticate successfully even if the OAuth2 provider doesn't provide email.
 *
 * NOTE: This test suite requires running the proxy with proxy.e2e.no-whitelist.yaml
 */
test.describe('OAuth2 flow without whitelist', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('should allow authentication when OAuth2 provider does not provide email (no whitelist)', async ({
    page,
  }) => {
    // Try to access protected resource
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Click OAuth2 provider
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login with user that doesn't provide email (noemail@example.com)
    const emailInput = page.locator('[data-test="login-email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill(NO_EMAIL_USER);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Authorize the app
    await Promise.all([
      // After authorization, should be redirected to home page
      page.waitForURL(/localhost:4180\/?$/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should successfully authenticate even without email
    // Because no whitelist is configured, email is not required
    // The page should NOT show an error
    await expect(page.locator('body')).not.toContainText(/Email Address Required|メールアドレスが必要です/i);
    await expect(page.locator('body')).not.toContainText(/Access Denied|アクセス拒否/i);

    // Should be able to access the app
    // Note: Email might not be displayed since provider didn't provide it
    const appContent = page.locator('[data-test="app-content"]');
    await expect(appContent).toBeVisible();

    // Can sign out
    await Promise.all([
      page.waitForURL(/\/_auth\/logout/),
      page.locator('[data-test="oauth-signout"]').click(),
    ]);

    // After logout, should need to login again
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('should work with regular users who provide email (no whitelist)', async ({ page }) => {
    const regularEmail = 'someone@example.com';

    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill(regularEmail);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await Promise.all([
      page.waitForURL(/localhost:4180\/?/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should authenticate successfully
    // With no whitelist, ANY authenticated user is allowed
    // Even emails not in a whitelist should work (because there's no whitelist)
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(regularEmail);
  });
});
