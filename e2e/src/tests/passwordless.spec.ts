import { test, expect } from '@playwright/test';
import { clearOtpFile, waitForOtp } from '../support/otp-reader';

const TEST_EMAIL = 'someone@example.com';

test.describe('Passwordless email flow', () => {
  test.beforeEach(async () => {
    await clearOtpFile();
  });

  test('user can log in with OTP and cannot reuse it', async ({ page }) => {
    await page.route('http://localhost:3001/**', (route) => {
      const url = route.request().url().replace('http://localhost:3001', 'http://stub-auth:3001');
      route.continue({ url });
    });

    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    await page.getByRole('link', { name: 'Or login with Email' }).click();

    await expect(page).toHaveURL(/\/_auth\/email$/);

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
