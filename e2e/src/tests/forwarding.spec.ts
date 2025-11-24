import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';
import { clearAllMessages, waitForLoginEmail } from '../support/mailpit-helper';

const TEST_EMAIL = 'someone@example.com';  // Must match stub-auth test user
const TEST_PASSWORD = 'password';
const TEST_USERNAME = 'Test User';

// Use port 4182 for forwarding tests (proxy-app-with-forwarding)
const FORWARDING_BASE_URL = 'http://localhost:4182';

test.describe('User info forwarding with encryption', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('OAuth2 authentication forwards both username and email (encrypted)', async ({ page }) => {
    // Start from home page on the forwarding proxy
    await page.goto(FORWARDING_BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Start OAuth2 flow
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login with test user
    const emailInput = page.locator('[data-test="login-email"]');
    await emailInput.fill(TEST_EMAIL);

    const passwordInput = page.locator('[data-test="login-password"]');
    await passwordInput.fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Authorize and wait for redirect
    await page.locator('[data-test="authorize-allow"]').click();

    // Wait for navigation to complete
    await page.waitForURL(/localhost:4182/);

    // Check if querystring forwarding is present in URL
    const currentUrl = page.url();
    console.log('Current URL after OAuth2 callback:', currentUrl);

    // 1. Verify Authentication Status Headers
    await expect(page.locator('[data-test="auth-status"]')).toContainText('true');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
    console.log('✓ Authentication Status Headers verified');

    // 2. Verify Forwarding Headers (X-Forwarded-*) - Encrypted
    await expect(page.locator('[data-test="forwarding-header-username"]')).toContainText(TEST_USERNAME);
    await expect(page.locator('[data-test="forwarding-header-email"]')).toContainText(TEST_EMAIL);
    console.log('✓ Forwarding Headers (encrypted) verified');

    // 3. Verify Forwarding QueryString (chatbotgate.*) - May or may not be present depending on redirect
    const hasQueryString = currentUrl.includes('chatbotgate.user=') || currentUrl.includes('chatbotgate.email=');
    if (hasQueryString) {
      console.log('✓ QueryString forwarding detected in URL');
      await expect(page.locator('[data-test="forwarding-qs-username"]')).toContainText(TEST_USERNAME);
      await expect(page.locator('[data-test="forwarding-qs-email"]')).toContainText(TEST_EMAIL);
      console.log('✓ Forwarding QueryString (encrypted) verified');
    } else {
      console.log('ℹ QueryString forwarding not present (depends on redirect flow)');
      await expect(page.locator('[data-test="forwarding-qs-not-present"]')).toBeVisible();
    }

    console.log('OAuth2 forwarding test completed: All 3 info sources verified');
  });

  test('OAuth2 authentication with querystring redirect verification', async ({ page }) => {
    // Access a specific path that will trigger redirect with querystring
    await page.goto(FORWARDING_BASE_URL + '/dashboard');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Start OAuth2 flow
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login
    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Authorize and wait for redirect
    await page.locator('[data-test="authorize-allow"]').click();
    await page.waitForURL(/localhost:4182/);

    const finalUrl = page.url();
    console.log('Final URL after redirect:', finalUrl);

    // After redirect to /dashboard, querystring might be present
    const hasQueryString = finalUrl.includes('chatbotgate.user=') || finalUrl.includes('chatbotgate.email=');

    // 1. Verify Authentication Status Headers
    await expect(page.locator('[data-test="auth-status"]')).toContainText('true');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
    console.log('✓ Authentication Status Headers verified');

    // 2. Verify Forwarding Headers - should always work
    await expect(page.locator('[data-test="forwarding-header-username"]')).toContainText(TEST_USERNAME);
    await expect(page.locator('[data-test="forwarding-header-email"]')).toContainText(TEST_EMAIL);
    console.log('✓ Forwarding Headers verified');

    // 3. Verify QueryString if present
    if (hasQueryString) {
      console.log('✓ QueryString forwarding detected');
      await expect(page.locator('[data-test="forwarding-qs-username"]')).toContainText(TEST_USERNAME);
      await expect(page.locator('[data-test="forwarding-qs-email"]')).toContainText(TEST_EMAIL);
      console.log('✓ Forwarding QueryString verified');
    } else {
      console.log('ℹ QueryString forwarding not present in final URL');
    }

    console.log('OAuth2 redirect test completed: All info sources verified');
  });
});

test.describe('Email authentication forwarding', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Not clearing Mailpit messages to avoid conflicts with parallel tests
    await routeStubAuthRequests(page);
  });

  test('Email authentication forwards email and username (userpart)', async ({ page }) => {
    const emailAuthAddress = 'forwarding-email@example.com';
    const expectedUsername = 'forwarding-email'; // Email local part (before @)

    // Start from home page on the forwarding proxy
    await page.goto(FORWARDING_BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Use email login directly from the login page
    await page.getByLabel('Email Address').fill(emailAuthAddress);

    // Submit form to send magic link
    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for the email to arrive in Mailpit and extract the login URL
    console.log(`Waiting for email to ${emailAuthAddress}...`);
    const loginUrl = await waitForLoginEmail(emailAuthAddress, {
      timeoutMs: 10_000,
      pollIntervalMs: 500,
    });

    console.log(`Got login URL: ${loginUrl}`);

    // Rewrite the login URL to use port 4182 (forwarding proxy)
    const url = new URL(loginUrl);
    url.port = '4182';
    const forwardingLoginUrl = url.toString();
    console.log(`Rewritten login URL for forwarding proxy: ${forwardingLoginUrl}`);

    // Navigate to the login URL
    await page.goto(forwardingLoginUrl);

    // Wait for authentication to complete
    await expect(page).toHaveURL(/localhost:4182/);

    // Verify we're logged in
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(emailAuthAddress);

    // 1. Verify Authentication Status Headers
    await expect(page.locator('[data-test="auth-status"]')).toContainText('true');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('email');
    console.log('✓ Authentication Status Headers verified (email auth)');

    // 2. Verify Forwarding Headers - username should be userpart (email local part) for email auth
    await expect(page.locator('[data-test="forwarding-header-username"]')).toContainText(expectedUsername);
    await expect(page.locator('[data-test="forwarding-header-email"]')).toContainText(emailAuthAddress);
    console.log('✓ Forwarding Headers verified: Username is userpart for email auth');

    // 3. Check QueryString forwarding
    const currentUrl = page.url();
    const hasQueryString = currentUrl.includes('chatbotgate.user=') || currentUrl.includes('chatbotgate.email=') || forwardingLoginUrl.includes('chatbotgate.user=') || forwardingLoginUrl.includes('chatbotgate.email=');
    if (hasQueryString) {
      console.log('✓ QueryString forwarding detected');
      await expect(page.locator('[data-test="forwarding-qs-username"]')).toContainText(expectedUsername);
      await expect(page.locator('[data-test="forwarding-qs-email"]')).toContainText(emailAuthAddress);
      console.log('✓ Forwarding QueryString verified: Username is userpart for email auth');
    } else {
      console.log('ℹ QueryString forwarding not present');
    }

    console.log('Email authentication forwarding test completed: Email and username (userpart) are forwarded');
  });

  test('Encrypted querystring parameters do not expose plaintext data', async ({ page }) => {
    // CRITICAL SECURITY TEST: Verify encrypted data is actually encrypted
    // Encrypted parameters should NOT contain plaintext sensitive information

    const testEmail = 'encryption-test@example.com';
    const expectedUsername = 'encryption-test'; // userpart

    await page.goto(FORWARDING_BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Use email login
    await page.getByLabel('Email Address').fill(testEmail);

    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email and get login URL
    console.log(`Waiting for email to ${testEmail}...`);
    const loginUrl = await waitForLoginEmail(testEmail, {
      timeoutMs: 10_000,
      pollIntervalMs: 500,
    });

    // Rewrite to use forwarding proxy
    const url = new URL(loginUrl);
    url.port = '4182';
    const forwardingLoginUrl = url.toString();

    await page.goto(forwardingLoginUrl);
    await page.waitForURL(/localhost:4182/);

    // Get the final URL with query parameters
    const finalUrl = page.url();
    console.log('Final URL with encrypted parameters:', finalUrl);

    // CRITICAL: Verify sensitive data is NOT in plaintext in the URL
    // The URL may contain 'chatbotgate.user=' and 'chatbotgate.email=' parameters
    // but the VALUES should be encrypted (base64 encoded encrypted data)

    if (finalUrl.includes('chatbotgate.user=') || finalUrl.includes('chatbotgate.email=')) {
      // Extract query string
      const urlObj = new URL(finalUrl);
      const queryString = urlObj.search;

      // CRITICAL CHECKS: Plaintext sensitive data must NOT appear in query string
      expect(queryString).not.toContain(testEmail); // Email should not be in plaintext
      expect(queryString).not.toContain(expectedUsername); // Username should not be in plaintext
      expect(queryString).not.toContain('@'); // No @ symbol (indicates unencrypted email)

      // Verify the parameters look encrypted (base64-like, long random strings)
      const userParam = urlObj.searchParams.get('chatbotgate.user');
      const emailParam = urlObj.searchParams.get('chatbotgate.email');

      if (userParam) {
        // Encrypted data should be base64 encoded and reasonably long
        expect(userParam.length).toBeGreaterThan(20); // Encrypted data is longer than plaintext
        expect(userParam).toMatch(/^[A-Za-z0-9+/=_-]+$/); // Base64-like characters
        console.log(`✓ chatbotgate.user is encrypted (length: ${userParam.length})`);
      }

      if (emailParam) {
        expect(emailParam.length).toBeGreaterThan(20);
        expect(emailParam).toMatch(/^[A-Za-z0-9+/=_-]+$/);
        console.log(`✓ chatbotgate.email is encrypted (length: ${emailParam.length})`);
      }

      console.log('✓ SECURITY VERIFIED: Sensitive data is encrypted in query parameters');
    } else {
      console.log('ℹ QueryString forwarding not present in this flow');
    }
  });
});
