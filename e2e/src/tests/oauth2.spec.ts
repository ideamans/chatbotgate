import { test, expect } from '@playwright/test';

const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';

test.describe('OAuth2 flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('http://localhost:3001/**', (route) => {
      const url = route.request().url().replace('http://localhost:3001', 'http://stub-auth:3001');
      route.continue({ url });
    });
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
      page.waitForURL(/http:\/\/proxy-app:4180\/?/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);

    await Promise.all([
      page.waitForURL(/\/_auth\/logout/),
      page.locator('[data-test="oauth-signout"]').click(),
    ]);

    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });
});
