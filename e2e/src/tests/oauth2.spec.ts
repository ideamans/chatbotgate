import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';
const NO_EMAIL_USER = 'noemail@example.com'; // Special user that doesn't provide email

test.describe('OAuth2 flow', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('user can authenticate via stub provider and sign out', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

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
      page.waitForURL(/localhost:4180\/?/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Verify authentication succeeded by checking X-Auth-Provider
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    await Promise.all([
      page.waitForURL(/\/_auth\/logout/),
      page.locator('[data-test="oauth-signout"]').click(),
    ]);

    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('should redirect to original URL after authentication', async ({ page }) => {
    // Try to access a protected path
    await page.goto('/protected-path');

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

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
      page.waitForURL(/localhost:4180\/protected-path/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should be back to the original protected path
    await expect(page).toHaveURL(/\/protected-path$/);
  });

  test('should allow access when OAuth2 provider does not provide email and no whitelist is configured', async ({
    page,
  }) => {
    // Try to access protected resource on :4180 (no whitelist)
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
      page.waitForURL(/localhost:4180\/?$/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should successfully access because:
    // 1. No whitelist is configured on :4180
    // 2. Authentication is sufficient without email
    await expect(page).toHaveURL(/localhost:4180\/?$/);

    // Should be on the protected page (not an error page)
    await expect(page.locator('body')).not.toContainText(/Email Address Required|メールアドレスが必要です/i);
    await expect(page.locator('body')).not.toContainText(/Forbidden|アクセスが拒否されました/i);

    // Verify authentication succeeded by checking X-Auth-Provider
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
  });

  test('should show error when OAuth2 provider does not provide email and whitelist is configured', async ({
    page,
  }) => {
    // Try to access protected resource on :4181 (with whitelist)
    await page.goto('http://localhost:4181/');
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

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
      // After authorization, should be redirected back to proxy
      page.waitForURL(/localhost:4181/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should show error page because:
    // 1. Whitelist is configured on :4181 (allowed_emails: ["allowed@example.com"], allowed_domains: ["@allowed.example.com"])
    // 2. Provider didn't provide email address
    // 3. Email is required for authorization check when whitelist is configured
    await expect(page.locator('body')).toContainText(/Email Address Required|メールアドレスが必要です/i);

    // Should show link back to login
    const backLink = page.getByRole('link', { name: /Back to login|ログイン/ });
    await expect(backLink).toBeVisible();
  });
});
