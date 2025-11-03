import { test, expect } from '@playwright/test';
import { waitForLoginEmail, clearAllMessages, extractLoginUrl } from '../support/mailpit-helper';
import { routeStubAuthRequests } from '../support/stub-auth-route';

test.describe('Passwordless email flow (Mailpit)', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Not clearing Mailpit messages to avoid conflicts with parallel tests
    // Each test uses a unique email address to ensure isolation
    await routeStubAuthRequests(page);
  });

  test('user can log in via email link from Mailpit', async ({ page }) => {
    // Use unique email for this test to avoid conflicts with parallel tests
    const TEST_EMAIL = 'mailpit-login@example.com';
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // The login page shows both OAuth2 and Email login options on the same page
    // Fill in the email address directly
    await page.getByLabel('Email Address').fill(TEST_EMAIL);

    // Click the send button and wait for the confirmation page
    await Promise.all([
      page.waitForURL(/\/_auth\/email\/send/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for the email to arrive in Mailpit and extract the login URL
    console.log(`Waiting for email to ${TEST_EMAIL}...`);
    const loginUrl = await waitForLoginEmail(TEST_EMAIL, {
      timeoutMs: 10_000, // 10 seconds timeout
      pollIntervalMs: 500, // Check every 500ms
    });

    console.log(`Got login URL: ${loginUrl}`);

    // Navigate to the login URL
    await page.goto(loginUrl);

    // Verify we're logged in by checking for the user email on the page
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);

    // Try to reuse the same token - should fail
    await page.goto(loginUrl);

    // Should see error message
    await expect(page.locator('body')).toContainText('Invalid or Expired Token');
  });

  test('can extract token from login URL', async ({ page }) => {
    // Use unique email for this test to avoid conflicts with parallel tests
    const TEST_EMAIL = 'mailpit-token@example.com';
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Send login email
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await Promise.all([
      page.waitForURL(/\/_auth\/email\/send/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email and extract login URL
    const loginUrl = await waitForLoginEmail(TEST_EMAIL);
    console.log(`Login URL: ${loginUrl}`);

    // Extract token from URL
    const url = new URL(loginUrl);
    const token = url.searchParams.get('token');

    expect(token).toBeTruthy();
    expect(token).toMatch(/^[A-Za-z0-9_\-+=]+$/); // URL-safe Base64 pattern

    console.log(`Extracted token: ${token}`);

    // Use the token to log in
    await page.goto(`/_auth/email/verify?token=${token}`);

    // Verify logged in
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);
  });

  test('email contains correct content', async ({ page }) => {
    // Use unique email for this test to avoid conflicts with parallel tests
    const TEST_EMAIL = 'mailpit-content@example.com';
    await page.goto('/');

    // Send login email
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Wait for email - this time we'll get the full message
    const { waitForMessage, getMessage } = await import('../support/mailpit-helper');
    const message = await waitForMessage(TEST_EMAIL);

    // Get full message details
    const detail = await getMessage(message.ID);

    // Verify email properties
    expect(message.Subject).toContain('Login Link');
    expect(message.From.Address).toBe('noreply@example.com');
    expect(message.To[0].Address).toBe(TEST_EMAIL);

    // Verify email body contains login URL
    expect(detail.Text).toContain('/_auth/email/verify?token=');
    expect(detail.HTML).toContain('/_auth/email/verify?token=');

    // Verify expiration time is mentioned (3 minutes in test config)
    expect(detail.Text).toContain('3 minutes');

    console.log('Email subject:', message.Subject);
    console.log('Email snippet:', message.Snippet);
  });
});
