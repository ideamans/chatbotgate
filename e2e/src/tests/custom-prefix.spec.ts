import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';
import { waitForMessage, getMessage } from '../support/mailpit-helper';

// OAuth2 tests use someone@example.com (registered in stub-auth)
const OAUTH2_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';
// Email auth tests use unique email (chatbotgate's email auth doesn't need stub-auth)
const EMAIL_AUTH_EMAIL = 'custom-prefix-email@example.com';
const BASE_URL = 'http://localhost:4185';
const AUTH_PREFIX = '/_oauth2_proxy';

test.describe('Custom auth prefix (/_oauth2_proxy)', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('user can authenticate via OAuth2 with custom prefix', async ({ page }) => {
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));

    await page.getByRole('link', { name: 'stub-auth' }).click();

    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill(OAUTH2_EMAIL);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await Promise.all([
      page.waitForURL(/localhost:4185\/?/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Verify authentication succeeded by checking X-Auth-Provider
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    // Test logout with custom prefix
    await Promise.all([
      page.waitForURL(new RegExp(`${AUTH_PREFIX}/logout`)),
      page.locator('[data-test="oauth-signout"]').click(),
    ]);

    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));
  });

  test('should redirect to original URL after authentication with custom prefix', async ({ page }) => {
    // Try to access a protected path
    await page.goto(BASE_URL + '/protected-path');

    // Should be redirected to login (with custom prefix)
    await expect(page).toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));

    // Complete OAuth2 flow
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill(OAUTH2_EMAIL);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await Promise.all([
      page.waitForURL(/localhost:4185\/protected-path/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should be back to the original protected path
    await expect(page).toHaveURL(/\/protected-path$/);
  });

  test('should handle email verification with original URL redirect', async ({ page }) => {
    const protectedPath = '/some-protected-resource';

    // Try to access protected path
    await page.goto(BASE_URL + protectedPath);
    await expect(page).toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));

    // Fill in email and submit directly on login page
    await page.getByLabel('Email Address').fill(EMAIL_AUTH_EMAIL);
    await Promise.all([
      page.waitForURL(new RegExp(`${AUTH_PREFIX}/email/sent`)),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for success message
    await expect(page.locator('body')).toContainText(/Check Your Email|メールを確認してください/i);

    // Fetch verification link from Mailpit using helper with retry logic
    const message = await waitForMessage(EMAIL_AUTH_EMAIL, { timeoutMs: 30_000, pollIntervalMs: 500 });
    const messageDetail = await getMessage(message.ID);

    // Extract and visit verification URL
    // Look for full URL starting with http:// or https://
    const htmlContent = messageDetail.HTML;
    const urlMatch = htmlContent.match(/href="(https?:\/\/[^"]*\/_oauth2_proxy\/email\/verify[^"]*)"/);
    const verifyUrl = urlMatch![1];

    // Visit verification URL (browser will redirect to protected path automatically)
    await page.goto(verifyUrl);

    // Verify we're on the protected path and authenticated
    await expect(page).toHaveURL(new RegExp(`${protectedPath}$`));
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('email');
  });

  test('custom prefix paths should not conflict with upstream paths', async ({ page }) => {
    // First authenticate
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));

    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill(OAUTH2_EMAIL);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await Promise.all([
      page.waitForURL(/localhost:4185\/?/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Now try to access a path that would conflict with default /_auth prefix
    // This path should be proxied to upstream, not trapped by auth middleware
    await page.goto(BASE_URL + '/_auth/some-resource');

    // Should not be redirected to login (already authenticated)
    // Should be proxied to upstream
    // Target app returns 404 for unknown paths, so we expect either:
    // - Upstream response (not a login page)
    // - Or specific error from upstream
    await expect(page).not.toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));

    // Verify we're still authenticated
    await page.goto(BASE_URL + '/');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
  });
});
