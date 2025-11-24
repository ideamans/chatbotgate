import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';

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
});
