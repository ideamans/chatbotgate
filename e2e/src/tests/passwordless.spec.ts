import { test, expect } from '@playwright/test';
import { clearOtpFile, waitForOtp } from '../support/otp-reader';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const TEST_EMAIL = 'someone@example.com';

test.describe('Passwordless email flow', () => {
  test.beforeEach(async ({ page }) => {
    await clearOtpFile();
    await routeStubAuthRequests(page);
  });

  test('user can log in with OTP and cannot reuse it', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // The login page shows both OAuth2 and Email login options on the same page
    // Fill in the email address directly
    await page.getByLabel('Email Address').fill(TEST_EMAIL);

    await Promise.all([
      page.waitForURL(/\/_auth\/email\/send/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    const otp = await waitForOtp(TEST_EMAIL);

    await page.goto(otp.login_url);

    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);

    await page.goto(otp.login_url);

    await expect(page.locator('body')).toContainText('Invalid or Expired Token');
  });
});
