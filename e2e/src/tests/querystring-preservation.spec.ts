import { test, expect } from '@playwright/test';
import { waitForLoginEmail, clearAllMessages } from '../support/mailpit-helper';
import { routeStubAuthRequests } from '../support/stub-auth-route';

// Test user credentials
const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';
const TEST_USERNAME = 'Test User';

// Base URLs
const PROXY_BASE_URL = 'http://localhost:4180';
const FORWARDING_BASE_URL = 'http://localhost:4182';

test.describe('QueryString preservation without forwarding (:4180)', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Not clearing Mailpit messages to avoid conflicts with parallel tests
    await routeStubAuthRequests(page);
  });

  test('should preserve original query parameters after OAuth2 authentication', async ({ page }) => {
    // Access URL with query parameters
    const originalParams = 'foo=bar&baz=qux&test=123';
    await page.goto(`${PROXY_BASE_URL}/dashboard?${originalParams}`);

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Complete OAuth2 flow
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();
    await page.waitForURL(/localhost:4180\/dashboard/);

    // Verify the final URL contains original parameters
    const finalUrl = page.url();
    console.log('Final URL after OAuth2:', finalUrl);

    expect(finalUrl).toContain('foo=bar');
    expect(finalUrl).toContain('baz=qux');
    expect(finalUrl).toContain('test=123');

    // Should NOT contain chatbotgate.* parameters (forwarding disabled)
    expect(finalUrl).not.toContain('chatbotgate.user=');
    expect(finalUrl).not.toContain('chatbotgate.email=');

    console.log('✓ Original query parameters preserved after OAuth2 authentication');
  });

  test('should preserve original query parameters after email authentication', async ({ page }) => {
    // Access URL with query parameters
    const originalParams = 'foo=bar&baz=qux&test=456';
    await page.goto(`${PROXY_BASE_URL}/dashboard?${originalParams}`);

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Fill email address for passwordless login
    await page.getByLabel('Email Address').fill(TEST_EMAIL);

    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email and get login URL
    console.log(`Waiting for email to ${TEST_EMAIL}...`);
    const loginUrl = await waitForLoginEmail(TEST_EMAIL, {
      timeoutMs: 30_000,
      pollIntervalMs: 500,
    });
    console.log(`Got login URL: ${loginUrl}`);

    // Navigate to login URL
    await page.goto(loginUrl);
    await page.waitForURL(/localhost:4180\/dashboard/);

    // Verify the final URL contains original parameters
    const finalUrl = page.url();
    console.log('Final URL after email auth:', finalUrl);

    expect(finalUrl).toContain('foo=bar');
    expect(finalUrl).toContain('baz=qux');
    expect(finalUrl).toContain('test=456');

    // Should NOT contain chatbotgate.* parameters (forwarding disabled)
    expect(finalUrl).not.toContain('chatbotgate.user=');
    expect(finalUrl).not.toContain('chatbotgate.email=');

    console.log('✓ Original query parameters preserved after email authentication');
  });
});

test.describe('QueryString merging with forwarding (:4182)', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Not clearing Mailpit messages to avoid conflicts with parallel tests
    await routeStubAuthRequests(page);
  });

  test('should merge original parameters with chatbotgate.* after OAuth2 authentication', async ({ page }) => {
    // Access URL with query parameters
    const originalParams = 'foo=bar&baz=qux&test=789';
    await page.goto(`${FORWARDING_BASE_URL}/dashboard?${originalParams}`);

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Complete OAuth2 flow
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();
    await page.waitForURL(/localhost:4182\/dashboard/);

    // Verify the final URL contains both original and chatbotgate parameters
    const finalUrl = page.url();
    console.log('Final URL after OAuth2 with forwarding:', finalUrl);

    // Original parameters should be preserved
    expect(finalUrl).toContain('foo=bar');
    expect(finalUrl).toContain('baz=qux');
    expect(finalUrl).toContain('test=789');

    // chatbotgate.* parameters should be added (forwarding enabled with querystring)
    expect(finalUrl).toContain('chatbotgate.user=');
    expect(finalUrl).toContain('chatbotgate.email=');

    // Verify the page can decode both original and forwarding parameters
    await expect(page.locator('[data-test="auth-status"]')).toContainText('true');
    await expect(page.locator('[data-test="forwarding-qs-username"]')).toContainText(TEST_USERNAME);
    await expect(page.locator('[data-test="forwarding-qs-email"]')).toContainText(TEST_EMAIL);

    console.log('✓ Original parameters preserved and chatbotgate.* parameters added');
  });

  test('should merge original parameters with chatbotgate.* after email authentication', async ({ page }) => {
    const expectedUsername = 'someone'; // Email local part (before @) from TEST_EMAIL

    // Access URL with query parameters
    const originalParams = 'foo=bar&baz=qux&test=abc';
    await page.goto(`${FORWARDING_BASE_URL}/dashboard?${originalParams}`);

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Fill email address for passwordless login
    await page.getByLabel('Email Address').fill(TEST_EMAIL);

    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email and get login URL
    console.log(`Waiting for email to ${TEST_EMAIL}...`);
    const loginUrl = await waitForLoginEmail(TEST_EMAIL, {
      timeoutMs: 30_000,
      pollIntervalMs: 500,
    });
    console.log(`Got login URL: ${loginUrl}`);

    // Navigate to login URL
    await page.goto(loginUrl);

    // Wait for authentication to complete (might redirect to / instead of /dashboard for email auth)
    await page.waitForLoadState('networkidle');

    // Get the actual URL after authentication
    const finalUrl = page.url();
    console.log('Final URL after email auth with forwarding:', finalUrl);

    // Original parameters should be preserved
    expect(finalUrl).toContain('foo=bar');
    expect(finalUrl).toContain('baz=qux');
    expect(finalUrl).toContain('test=abc');

    // chatbotgate.* parameters should be added (forwarding enabled with querystring)
    // Note: For email auth, both chatbotgate.email and chatbotgate.user (userpart) should be present
    expect(finalUrl).toContain('chatbotgate.email=');
    expect(finalUrl).toContain('chatbotgate.user=');

    // Verify the page can decode both original and forwarding parameters
    await expect(page.locator('[data-test="auth-status"]')).toContainText('true');
    await expect(page.locator('[data-test="forwarding-qs-email"]')).toContainText(TEST_EMAIL);

    // Username should be userpart (email local part) for email auth
    await expect(page.locator('[data-test="forwarding-qs-username"]')).toContainText(expectedUsername);

    console.log('✓ Original parameters preserved and chatbotgate.* parameters added (username is userpart for email auth)');
  });

  test('should preserve special characters in query parameters', async ({ page }) => {
    // IMPORTANT: Test URL encoding/decoding of special characters
    // Special chars like &, =, ?, #, +, spaces must be properly encoded/decoded

    const specialChars = {
      space: 'hello world',
      ampersand: 'foo&bar',
      equals: 'key=value',
      plus: 'one+two',
      hash: 'tag#123',
      percent: '50%off',
      japanese: '日本語',
    };

    // Build query string with special characters (properly encoded)
    const params = new URLSearchParams(specialChars);
    const queryString = params.toString();

    await page.goto(`${FORWARDING_BASE_URL}/test?${queryString}`);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Complete OAuth2 flow
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    await page.locator('[data-test="authorize-allow"]').click();
    await page.waitForURL(/localhost:4182\/test/);

    // Parse final URL parameters
    const finalUrl = new URL(page.url());
    const finalParams = finalUrl.searchParams;

    // CRITICAL: Verify all special characters are preserved correctly
    expect(finalParams.get('space')).toBe(specialChars.space);
    expect(finalParams.get('ampersand')).toBe(specialChars.ampersand);
    expect(finalParams.get('equals')).toBe(specialChars.equals);
    expect(finalParams.get('plus')).toBe(specialChars.plus);
    expect(finalParams.get('hash')).toBe(specialChars.hash);
    expect(finalParams.get('percent')).toBe(specialChars.percent);
    expect(finalParams.get('japanese')).toBe(specialChars.japanese);

    console.log('✓ All special characters preserved correctly after authentication');
  });
});
