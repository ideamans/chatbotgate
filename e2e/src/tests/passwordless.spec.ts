import { test, expect } from '@playwright/test';
import { waitForLoginEmail, clearAllMessages } from '../support/mailpit-helper';
import { routeStubAuthRequests } from '../support/stub-auth-route';

// Use unique email address to avoid conflicts with parallel tests
const TEST_EMAIL = 'passwordless-basic@example.com';

test.describe('Passwordless email flow', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Not clearing Mailpit messages to avoid conflicts with parallel tests
    // Each test uses a unique email address to ensure isolation
    await routeStubAuthRequests(page);
  });

  test('user can log in with OTP and cannot reuse it', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // The login page shows both OAuth2 and Email login options on the same page
    // Fill in the email address directly
    await page.getByLabel('Email Address').fill(TEST_EMAIL);

    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email to arrive in Mailpit and extract login URL
    console.log(`Waiting for email to ${TEST_EMAIL}...`);
    const loginUrl = await waitForLoginEmail(TEST_EMAIL, {
      timeoutMs: 30_000,
      pollIntervalMs: 500,
    });

    console.log(`Got login URL: ${loginUrl}`);

    // Navigate to the login URL
    await page.goto(loginUrl);

    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);

    // Try to reuse the same URL - should fail
    await page.goto(loginUrl);

    await expect(page.locator('body')).toContainText('Invalid or Expired Token');
  });
});
