import { test, expect } from '@playwright/test';
import { waitForLoginEmail, clearAllMessages } from '../support/mailpit-helper';
import { routeStubAuthRequests } from '../support/stub-auth-route';

// Test users for whitelist testing
// These must match the users registered in stub-auth
const ALLOWED_EMAIL_USER = 'allowed@example.com'; // Whitelisted email
const ALLOWED_DOMAIN_USER = 'user@allowed.example.com'; // Whitelisted domain (registered in stub-auth)
const DENIED_USER = 'denied@example.com'; // Not whitelisted (registered in stub-auth)
const TEST_PASSWORD = 'password';

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

    // Should show specific forbidden/access denied error (not just any mention of these words)
    const mainContent = page.locator('main, [role="main"], body');
    await expect(mainContent).toContainText(/Access Denied|Forbidden|アクセスが拒否されました/i);
    await expect(mainContent).toContainText(/(not |pre-)authorized|(not.*)?whitelist|許可されていません/i);

    // Should show link back to login
    const backLink = page.getByRole('link', { name: /Back to login|ログイン/ });
    await expect(backLink).toBeVisible();

    // CRITICAL: Verify user is NOT granted access (still on error page or redirected to login)
    const currentUrl = page.url();
    expect(currentUrl).not.toMatch(/localhost:4181\/?$/); // Not on home page
  });
});

test.describe('Whitelist Authorization - Email Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Not clearing Mailpit messages to avoid conflicts with parallel tests
    await routeStubAuthRequests(page);
  });

  test('should allow access for allowed email address via email link', async ({ page }) => {
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Fill in the email address
    await page.getByLabel('Email Address').fill(ALLOWED_EMAIL_USER);

    await Promise.all([
      page.waitForURL(/localhost:4181\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email from Mailpit and extract login URL
    const loginUrl = await waitForLoginEmail(ALLOWED_EMAIL_USER, {
      timeoutMs: 10_000,
      pollIntervalMs: 500,
    });

    // Access the login URL
    await page.goto(loginUrl);

    // Should successfully access the protected resource
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(ALLOWED_EMAIL_USER);
  });

  test('should allow access for allowed domain via email link', async ({ page }) => {
    await page.goto(`${WHITELIST_PROXY_URL}/`);
    await expect(page).toHaveURL(/localhost:4181\/_auth\/login$/);

    // Fill in the email address
    await page.getByLabel('Email Address').fill(ALLOWED_DOMAIN_USER);

    await Promise.all([
      page.waitForURL(/localhost:4181\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email from Mailpit and extract login URL
    const loginUrl = await waitForLoginEmail(ALLOWED_DOMAIN_USER, {
      timeoutMs: 10_000,
      pollIntervalMs: 500,
    });

    // Access the login URL
    await page.goto(loginUrl);

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

    // Should show specific forbidden error because email is not in whitelist
    const mainContent = page.locator('main, [role="main"], body');
    await expect(mainContent).toContainText(/Access Denied|Forbidden|not.*whitelist|アクセスが拒否されました/i);
    await expect(mainContent).toContainText(/(not |pre-)authorized|(not.*)?whitelist|許可されていません/i);

    // Should show link back to login
    const backLink = page.getByRole('link', { name: /Back to login|ログイン/ });
    await expect(backLink).toBeVisible();
  });
});
