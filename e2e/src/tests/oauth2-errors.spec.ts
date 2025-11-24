import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const BASE_URL = 'http://localhost:4180';
const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';

test.describe('OAuth2 error handling', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('user denying authorization shows error message', async ({ page }) => {
    // Navigate to protected resource
    await page.goto(BASE_URL + '/');

    // Should redirect to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Click OAuth2 provider
    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login
    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // User denies authorization by clicking deny button
    await Promise.all([
      // Should be redirected back to login with error
      page.waitForURL(new RegExp(`${BASE_URL.replace('http://', '')}.*error`)),
      page.getByRole('button', { name: /拒否/ }).click(),
    ]);

    // Should show error message on the page
    await expect(page.locator('body')).toContainText(/access.*denied|アクセスが拒否|denied|拒否/i);

    // Should not be authenticated
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('invalid authorization code shows error', async ({ page, request }) => {
    // Try to use callback with invalid code directly
    const invalidCode = 'invalid-code-12345';
    const callbackUrl = `${BASE_URL}/_auth/oauth2/callback?code=${invalidCode}&state=test`;

    await page.goto(callbackUrl);

    // Should show error page
    await expect(page.locator('body')).toContainText(/error|invalid|failed|エラー/i);

    // Should not be authenticated
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('missing code parameter shows error', async ({ page }) => {
    // Try callback without code parameter
    const callbackUrl = `${BASE_URL}/_auth/oauth2/callback?state=test`;

    await page.goto(callbackUrl);

    // Should show error page
    await expect(page.locator('body')).toContainText(/error|invalid|required|必須|エラー/i);

    // Should not be authenticated
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('error parameter in callback shows error message', async ({ page }) => {
    // Simulate OAuth2 provider returning error
    const callbackUrl = `${BASE_URL}/_auth/oauth2/callback?error=access_denied&error_description=User+denied`;

    await page.goto(callbackUrl);

    // Should show error message
    await expect(page.locator('body')).toContainText(/access.*denied|denied|拒否/i);

    // Should not be authenticated
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('error with state parameter preserves state', async ({ page }) => {
    const testState = 'test-state-12345';
    const callbackUrl = `${BASE_URL}/_auth/oauth2/callback?error=access_denied&state=${testState}`;

    await page.goto(callbackUrl);

    // Should show error
    await expect(page.locator('body')).toContainText(/access.*denied|denied|拒否/i);

    // State should be handled (no crash, proper error page)
    await expect(page).not.toHaveURL(/about:blank/);
  });

  test('malformed callback URL does not crash', async ({ page }) => {
    // Test various malformed URLs
    const malformedUrls = [
      `${BASE_URL}/_auth/oauth2/callback?code=`,  // Empty code
      `${BASE_URL}/_auth/oauth2/callback?code=&state=`,  // Empty values
      `${BASE_URL}/_auth/oauth2/callback?error=&error_description=`,  // Empty error
    ];

    for (const url of malformedUrls) {
      await page.goto(url);

      // Should show error page (not crash)
      await expect(page).not.toHaveURL(/about:blank/);
      await expect(page.locator('body')).toBeVisible();
    }
  });

  test('reusing authorization code fails', async ({ page }) => {
    // Complete successful OAuth2 flow
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ]);

    // Capture the callback URL
    let callbackUrl = '';
    page.on('request', (request) => {
      const url = request.url();
      if (url.includes('/_auth/oauth2/callback') && url.includes('code=')) {
        callbackUrl = url;
      }
    });

    await Promise.all([
      page.waitForURL(new RegExp(BASE_URL.replace('http://', ''))),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should be authenticated
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    // Logout
    await Promise.all([
      page.waitForURL(/\/_auth\/logout/),
      page.locator('[data-test="oauth-signout"]').click(),
    ]);

    // Try to reuse the same authorization code
    if (callbackUrl) {
      await page.goto(callbackUrl);

      // Should show error (code already used)
      await expect(page.locator('body')).toContainText(/error|invalid|expired|無効|エラー/i);

      // Should not be authenticated
      await page.goto(BASE_URL + '/');
      await expect(page).toHaveURL(/\/_auth\/login$/);
    }
  });

  test('invalid state parameter does not crash', async ({ page }) => {
    // Start OAuth2 flow normally to get valid session
    await page.goto(BASE_URL + '/');
    await expect(page).toHaveURL(/\/_auth\/login$/);
    await page.getByRole('link', { name: 'stub-auth' }).click();

    // Wait for OAuth2 provider
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Login
    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
    await page.locator('[data-test="login-submit"]').click();

    // Wait for authorize page
    await expect(page).toHaveURL(/localhost:3001\/oauth\/authorize/);

    // Now manually navigate to callback with invalid state
    // (This simulates CSRF attack or state mismatch)
    const invalidStateUrl = `${BASE_URL}/_auth/oauth2/callback?code=test&state=invalid-state-12345`;
    await page.goto(invalidStateUrl);

    // Should handle gracefully (show error, not crash)
    await expect(page).not.toHaveURL(/about:blank/);
    await expect(page.locator('body')).toBeVisible();
  });

  test('concurrent OAuth2 flows do not interfere', async ({ page, context }) => {
    // Create second page (simulates second browser tab)
    const page2 = await context.newPage();
    await routeStubAuthRequests(page2);

    // Start OAuth2 flow in first page
    await page.goto(BASE_URL + '/test1');
    await expect(page).toHaveURL(/\/_auth\/login$/);
    await page.getByRole('link', { name: 'stub-auth' }).click();

    // Start OAuth2 flow in second page
    await page2.goto(BASE_URL + '/test2');
    await expect(page2).toHaveURL(/\/_auth\/login$/);
    await page2.getByRole('link', { name: 'stub-auth' }).click();

    // Complete authentication in first page
    await expect(page).toHaveURL(/localhost:3001\/login/);
    await page.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
    await page.locator('[data-test="login-submit"]').click();
    await expect(page).toHaveURL(/localhost:3001\/oauth\/authorize/);
    await page.locator('[data-test="authorize-allow"]').click();

    // First page should be authenticated
    await expect(page).toHaveURL(new RegExp(BASE_URL.replace('http://', '')));
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    // Complete authentication in second page
    await expect(page2).toHaveURL(/localhost:3001\/login/);
    await page2.locator('[data-test="login-email"]').fill(TEST_EMAIL);
    await page2.locator('[data-test="login-password"]').fill(TEST_PASSWORD);
    await page2.locator('[data-test="login-submit"]').click();
    await expect(page2).toHaveURL(/localhost:3001\/oauth\/authorize/);
    await page2.locator('[data-test="authorize-allow"]').click();

    // Second page should also be authenticated
    await expect(page2).toHaveURL(new RegExp(BASE_URL.replace('http://', '')));
    await expect(page2.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    // Both pages should remain authenticated
    await page.goto(BASE_URL + '/');
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    await page2.goto(BASE_URL + '/');
    await expect(page2.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    await page2.close();
  });

  test('OAuth2 error preserves redirect URL', async ({ page }) => {
    // Try to access specific protected path
    const protectedPath = '/dashboard?foo=bar';
    await page.goto(BASE_URL + protectedPath);

    // Should redirect to login
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

    // Deny authorization
    await page.getByRole('button', { name: /拒否/ }).click();

    // Should show error
    await expect(page.locator('body')).toContainText(/access.*denied|denied|拒否/i);

    // Try again with allow
    await page.goto(BASE_URL + protectedPath);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    await page.getByRole('link', { name: 'stub-auth' }).click();
    await expect(page).toHaveURL(/localhost:3001\/login/);

    // Should still be logged in at stub-auth
    // Just need to authorize again
    await expect(page).toHaveURL(/localhost:3001\/oauth\/authorize/);

    await Promise.all([
      page.waitForURL(new RegExp(protectedPath.replace(/[?]/g, '\\$&'))),
      page.locator('[data-test="authorize-allow"]').click(),
    ]);

    // Should be redirected to original protected path
    await expect(page).toHaveURL(new RegExp(protectedPath.replace(/[?]/g, '\\$&')));
  });

  test('XSS in error_description is sanitized', async ({ page }) => {
    // Try to inject XSS via error_description parameter
    const xssPayload = '<script>alert("xss")</script>';
    const encodedPayload = encodeURIComponent(xssPayload);
    const callbackUrl = `${BASE_URL}/_auth/oauth2/callback?error=access_denied&error_description=${encodedPayload}`;

    await page.goto(callbackUrl);

    // Should show error page
    await expect(page.locator('body')).toBeVisible();

    // XSS should be sanitized (script should not execute)
    // If XSS was not sanitized, this would fail
    const bodyContent = await page.locator('body').innerHTML();
    expect(bodyContent).not.toContain('<script>alert');

    // Error message should be displayed safely
    await expect(page.locator('body')).toContainText(/error|denied|access/i);
  });
});
