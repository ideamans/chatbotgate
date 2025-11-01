import { test, expect } from '@playwright/test';
import { clearOtpFile, waitForOtp } from '../support/otp-reader';
import { routeStubAuthRequests } from '../support/stub-auth-route';
import path from 'path';

// Test users for whitelist testing
const ALLOWED_EMAIL_USER = 'allowed@example.com';
const ALLOWED_DOMAIN_USER = 'user@allowed.example.com';
const DENIED_USER = 'denied@example.com';
const TEST_PASSWORD = 'password';

// OTP file for whitelist proxy
const WHITELIST_OTP_FILE = path.resolve(__dirname, '../../tmp/passwordless-otp-wl.jsonl');

// Base URL for whitelist proxy
const WHITELIST_PROXY_URL = 'http://localhost:4181';

test.describe('Whitelist Authorization - OAuth2', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('should allow access for allowed email address via OAuth2', async ({ page }) => {
    // Try to access protected resource on whitelist proxy
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Click OAuth2 provider
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login with allowed email user
    const emailInput = page.locator('[data-test="login-email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill(ALLOWED_EMAIL_USER);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Authorize the app
    await Promise.all([
      page.waitForURL(/localhost:4181\/?$/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should successfully access the protected resource
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(ALLOWED_EMAIL_USER);
  });

  test('should allow access for allowed domain via OAuth2', async ({ page }) => {
    // Try to access protected resource on whitelist proxy
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Click OAuth2 provider
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login with allowed domain user
    const emailInput = page.locator('[data-test="login-email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill(ALLOWED_DOMAIN_USER);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Authorize the app
    await Promise.all([
      page.waitForURL(/localhost:4181\/?$/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should successfully access the protected resource
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(ALLOWED_DOMAIN_USER);
  });

  test('should deny access for non-whitelisted email via OAuth2', async ({ page }) => {
    // Try to access protected resource on whitelist proxy
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Click OAuth2 provider
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login with denied user
    const emailInput = page.locator('[data-test="login-email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill(DENIED_USER);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Authorize the app
    await Promise.all([
      page.waitForURL(/localhost:4181/),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should show forbidden error page
    await expect(page.locator('body')).toContainText(/Access Denied|Forbidden|アクセスが拒否されました/i);
    await expect(page.locator('body')).toContainText(/not authorized|pre-authorized|許可されていません/i);

    // Should show link back to login
    const backLink = page.getByRole('link', { name: /Back to login|ログイン/ });
    await expect(backLink).toBeVisible();
  });
});

test.describe('Whitelist Authorization - Email Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await clearOtpFile(WHITELIST_OTP_FILE);
    await routeStubAuthRequests(page);
  });

  test('should allow access for allowed email address via email link', async ({ page }) => {
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Fill in the email address
    await page.getByLabel('Email Address').fill(ALLOWED_EMAIL_USER);

    await Promise.all([
      page.waitForURL(/localhost:4181\/_auth\/email\/send/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for OTP from the whitelist OTP file
    const otp = await waitForOtp(ALLOWED_EMAIL_USER, { otpFile: WHITELIST_OTP_FILE });

    // Access the login URL
    await page.goto(otp.login_url);

    // Should successfully access the protected resource
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(ALLOWED_EMAIL_USER);
  });

  test('should allow access for allowed domain via email link', async ({ page }) => {
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Fill in the email address
    await page.getByLabel('Email Address').fill(ALLOWED_DOMAIN_USER);

    await Promise.all([
      page.waitForURL(/localhost:4181\/_auth\/email\/send/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for OTP from the whitelist OTP file
    const otp = await waitForOtp(ALLOWED_DOMAIN_USER, { otpFile: WHITELIST_OTP_FILE });

    // Access the login URL
    await page.goto(otp.login_url);

    // Should successfully access the protected resource
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(ALLOWED_DOMAIN_USER);
  });

  test('should deny access for non-whitelisted email via email link', async ({ page }) => {
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Fill in the email address
    await page.getByLabel('Email Address').fill(DENIED_USER);

    // Click send - should be rejected immediately
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Wait for page to load (might redirect or show error inline)
    await page.waitForLoadState('networkidle');

    // Should show forbidden error because email is not in whitelist
    await expect(page.locator('body')).toContainText(/Access Denied|Forbidden|アクセスが拒否されました/i);
    await expect(page.locator('body')).toContainText(/not authorized|pre-authorized|許可されていません/i);

    // Should show link back to login
    const backLink = page.getByRole('link', { name: /Back to login|ログイン/ });
    await expect(backLink).toBeVisible();
  });
});
