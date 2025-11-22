import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';
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
    await emailInput.fill(TEST_EMAIL);

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
    await emailInput.fill(TEST_EMAIL);

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

  test('should handle email verification with original URL redirect', async ({ page, request }) => {
    const protectedPath = '/some-protected-resource';

    // Try to access protected path
    await page.goto(BASE_URL + protectedPath);
    await expect(page).toHaveURL(new RegExp(`${AUTH_PREFIX}/login$`));

    // Fill in email and submit directly on login page
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await Promise.all([
      page.waitForURL(new RegExp(`${AUTH_PREFIX}/email/sent`)),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for success message
    await expect(page.locator('body')).toContainText(/Check Your Email|メールを確認してください/i);

    // Fetch verification link from Mailpit
    const mailpitResponse = await request.get('http://localhost:8025/api/v1/messages');
    const mailpitData = await mailpitResponse.json();
    const latestMessage = mailpitData.messages?.[0];

    const messageResponse = await request.get(`http://localhost:8025/api/v1/message/${latestMessage.ID}`);
    const messageData = await messageResponse.json();

    // Extract and visit verification URL
    // Look for full URL starting with http:// or https://
    const htmlContent = messageData.HTML;
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
    await emailInput.fill(TEST_EMAIL);

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
