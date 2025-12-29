import { test, expect } from '@playwright/test';

const DUMMY_UPSTREAM_BASE_URL = 'http://localhost:4186';

test.describe('Dummy upstream server (no config file)', () => {
  test('user can log in with default password and see dummy upstream response', async ({ page }) => {
    // Navigate to the app
    await page.goto(DUMMY_UPSTREAM_BASE_URL);

    // Should be redirected to login page
    await expect(page).toHaveURL(/\/_auth\/login/);

    // Check that password form is present (default config enables password auth)
    await expect(page.locator('#password-input')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible();

    // Enter the default password (P@ssW0rd)
    await page.locator('#password-input').fill('P@ssW0rd');

    // Click the sign in button
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Should be redirected to the home page after successful authentication
    await expect(page).toHaveURL(DUMMY_UPSTREAM_BASE_URL + '/');

    // Verify dummy upstream server response is displayed
    await expect(page.locator('h1')).toContainText('Dummy Upstream Server');
    await expect(page.locator('body')).toContainText('placeholder response');
  });

  test('dummy upstream shows request path', async ({ page }) => {
    // First, authenticate
    await page.goto(DUMMY_UPSTREAM_BASE_URL);
    await page.locator('#password-input').fill('P@ssW0rd');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await expect(page).toHaveURL(DUMMY_UPSTREAM_BASE_URL + '/');

    // Navigate to a specific path
    await page.goto(DUMMY_UPSTREAM_BASE_URL + '/some/test/path');

    // Verify the path is shown in the response
    await expect(page.locator('body')).toContainText('/some/test/path');
  });

  test('user can logout and re-authenticate', async ({ page }) => {
    // First, authenticate
    await page.goto(DUMMY_UPSTREAM_BASE_URL);
    await page.locator('#password-input').fill('P@ssW0rd');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await expect(page).toHaveURL(DUMMY_UPSTREAM_BASE_URL + '/');

    // Logout
    await page.goto(`${DUMMY_UPSTREAM_BASE_URL}/_auth/logout`);
    await expect(page).toHaveURL(/\/_auth\/logout$/);

    // Try to access home page again (should redirect to login)
    await page.goto(DUMMY_UPSTREAM_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login/);

    // Re-authenticate
    await page.locator('#password-input').fill('P@ssW0rd');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await expect(page).toHaveURL(DUMMY_UPSTREAM_BASE_URL + '/');

    // Verify dummy upstream response
    await expect(page.locator('h1')).toContainText('Dummy Upstream Server');
  });

  test('wrong password is rejected', async ({ page }) => {
    await page.goto(DUMMY_UPSTREAM_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login/);

    // Enter wrong password
    await page.locator('#password-input').fill('WrongPassword');

    // Listen for the alert dialog
    page.once('dialog', async dialog => {
      expect(dialog.message()).toContain('Invalid password');
      await dialog.accept();
    });

    // Click the sign in button
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Should still be on login page (may have query params)
    await expect(page).toHaveURL(/\/_auth\/login/);
  });
});
