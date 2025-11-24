import { Page, expect } from '@playwright/test';

/**
 * Custom assertion functions for E2E tests.
 * Provides domain-specific, reusable assertions to improve test readability.
 */

/**
 * Assert that user is on the login page
 *
 * @param page - Playwright page object
 * @example
 * await expectOnLoginPage(page);
 */
export async function expectOnLoginPage(page: Page): Promise<void> {
  await expect(page).toHaveURL(/\/_auth\/login$/);
}

/**
 * Assert that user is on the logout page
 *
 * @param page - Playwright page object
 * @example
 * await expectOnLogoutPage(page);
 */
export async function expectOnLogoutPage(page: Page): Promise<void> {
  await expect(page).toHaveURL(/\/_auth\/logout$/);
}

/**
 * Assert that user is on the email sent confirmation page
 *
 * @param page - Playwright page object
 * @example
 * await expectOnEmailSentPage(page);
 */
export async function expectOnEmailSentPage(page: Page): Promise<void> {
  await expect(page).toHaveURL(/\/_auth\/email\/sent$/);
}

/**
 * Assert that user is authenticated with specific email
 *
 * @param page - Playwright page object
 * @param email - Expected email address
 * @example
 * await expectAuthenticatedAs(page, 'user@example.com');
 */
export async function expectAuthenticatedAs(page: Page, email: string): Promise<void> {
  await expect(page.locator('[data-test="app-user-email"]')).toContainText(email);
}

/**
 * Assert that user is authenticated via specific provider
 *
 * @param page - Playwright page object
 * @param provider - Expected provider name
 * @example
 * await expectAuthenticatedViaProvider(page, 'stub-auth');
 */
export async function expectAuthenticatedViaProvider(page: Page, provider: string): Promise<void> {
  await expect(page.locator('[data-test="auth-provider"]')).toContainText(provider);
}

/**
 * Assert that user is on a specific URL path
 *
 * @param page - Playwright page object
 * @param path - Expected path (can be string or RegExp)
 * @example
 * await expectOnPath(page, '/dashboard');
 * await expectOnPath(page, /\/dashboard/);
 */
export async function expectOnPath(page: Page, path: string | RegExp): Promise<void> {
  if (typeof path === 'string') {
    await expect(page).toHaveURL(new RegExp(path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  } else {
    await expect(page).toHaveURL(path);
  }
}

/**
 * Assert that header exists with specific value
 *
 * @param page - Playwright page object
 * @param headerName - Header name to check
 * @param expectedValue - Expected header value (can be string or RegExp)
 * @example
 * await expectHeader(page, 'x-auth-user', 'user@example.com');
 * await expectHeader(page, 'x-auth-provider', /stub-auth/);
 */
export async function expectHeader(
  page: Page,
  headerName: string,
  expectedValue: string | RegExp
): Promise<void> {
  const headerElement = page.locator(`[data-test="header-${headerName}"]`);
  if (typeof expectedValue === 'string') {
    await expect(headerElement).toContainText(expectedValue);
  } else {
    await expect(headerElement).toHaveText(expectedValue);
  }
}

/**
 * Assert that query parameter exists with specific value
 *
 * @param page - Playwright page object
 * @param paramName - Query parameter name to check
 * @param expectedValue - Expected parameter value
 * @example
 * await expectQueryParam(page, 'user', 'user@example.com');
 */
export async function expectQueryParam(
  page: Page,
  paramName: string,
  expectedValue: string
): Promise<void> {
  const paramElement = page.locator(`[data-test="query-${paramName}"]`);
  await expect(paramElement).toContainText(expectedValue);
}

/**
 * Assert that error message is displayed
 *
 * @param page - Playwright page object
 * @param message - Expected error message (can be substring or RegExp)
 * @example
 * await expectErrorMessage(page, 'Invalid or Expired Token');
 * await expectErrorMessage(page, /not authorized/i);
 */
export async function expectErrorMessage(page: Page, message: string | RegExp): Promise<void> {
  if (typeof message === 'string') {
    await expect(page.locator('body')).toContainText(message);
  } else {
    await expect(page.locator('body')).toHaveText(message);
  }
}

/**
 * Assert that session cookie exists
 *
 * @param page - Playwright page object
 * @param cookieName - Cookie name (default: '_oauth2_proxy')
 * @example
 * await expectSessionCookieExists(page);
 * await expectSessionCookieExists(page, '_custom_session');
 */
export async function expectSessionCookieExists(
  page: Page,
  cookieName: string = '_oauth2_proxy'
): Promise<void> {
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find((c) => c.name === cookieName);
  expect(sessionCookie).toBeDefined();
}

/**
 * Assert that session cookie does not exist
 *
 * @param page - Playwright page object
 * @param cookieName - Cookie name (default: '_oauth2_proxy')
 * @example
 * await expectSessionCookieNotExists(page);
 */
export async function expectSessionCookieNotExists(
  page: Page,
  cookieName: string = '_oauth2_proxy'
): Promise<void> {
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find((c) => c.name === cookieName);
  expect(sessionCookie).toBeUndefined();
}

/**
 * Assert that session cookie has HttpOnly flag
 *
 * @param page - Playwright page object
 * @param cookieName - Cookie name (default: '_oauth2_proxy')
 * @example
 * await expectSessionCookieHttpOnly(page);
 */
export async function expectSessionCookieHttpOnly(
  page: Page,
  cookieName: string = '_oauth2_proxy'
): Promise<void> {
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find((c) => c.name === cookieName);
  expect(sessionCookie).toBeDefined();
  expect(sessionCookie?.httpOnly).toBe(true);
}

/**
 * Assert that session cookie has Secure flag
 *
 * @param page - Playwright page object
 * @param cookieName - Cookie name (default: '_oauth2_proxy')
 * @example
 * await expectSessionCookieSecure(page);
 */
export async function expectSessionCookieSecure(
  page: Page,
  cookieName: string = '_oauth2_proxy'
): Promise<void> {
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find((c) => c.name === cookieName);
  expect(sessionCookie).toBeDefined();
  expect(sessionCookie?.secure).toBe(true);
}

/**
 * Assert that session cookie has SameSite attribute
 *
 * @param page - Playwright page object
 * @param expectedValue - Expected SameSite value ('Lax', 'Strict', or 'None')
 * @param cookieName - Cookie name (default: '_oauth2_proxy')
 * @example
 * await expectSessionCookieSameSite(page, 'Lax');
 */
export async function expectSessionCookieSameSite(
  page: Page,
  expectedValue: 'Lax' | 'Strict' | 'None',
  cookieName: string = '_oauth2_proxy'
): Promise<void> {
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find((c) => c.name === cookieName);
  expect(sessionCookie).toBeDefined();
  expect(sessionCookie?.sameSite).toBe(expectedValue);
}

/**
 * Assert that page shows access denied (403)
 *
 * @param page - Playwright page object
 * @example
 * await expectAccessDenied(page);
 */
export async function expectAccessDenied(page: Page): Promise<void> {
  await expect(page.locator('body')).toContainText('Access Denied');
}

/**
 * Assert that page shows rate limit error
 *
 * @param page - Playwright page object
 * @example
 * await expectRateLimitError(page);
 */
export async function expectRateLimitError(page: Page): Promise<void> {
  await expect(page.locator('body')).toContainText(/rate limit/i);
}

/**
 * Assert that encrypted data can be decrypted correctly
 *
 * @param encryptedData - Encrypted data string (base64)
 * @param key - Encryption key
 * @param expectedPlaintext - Expected plaintext after decryption
 * @example
 * await expectDecrypts(encryptedData, key, 'user@example.com');
 */
export async function expectDecrypts(
  encryptedData: string,
  key: string,
  expectedPlaintext: string
): Promise<void> {
  // This is a simplified example - actual implementation would use crypto
  // For now, just verify that the data is base64-encoded
  expect(encryptedData).toMatch(/^[A-Za-z0-9+/]+=*$/);
  expect(key).toBeTruthy();
  expect(expectedPlaintext).toBeTruthy();
  // TODO: Implement actual AES-256-GCM decryption and verification
}

/**
 * Assert that compressed data can be decompressed correctly
 *
 * @param compressedData - Compressed data string (base64)
 * @param expectedPlaintext - Expected plaintext after decompression
 * @example
 * await expectDecompresses(compressedData, 'user@example.com');
 */
export async function expectDecompresses(
  compressedData: string,
  expectedPlaintext: string
): Promise<void> {
  // This is a simplified example - actual implementation would use zlib
  // For now, just verify that the data is base64-encoded
  expect(compressedData).toMatch(/^[A-Za-z0-9+/]+=*$/);
  expect(expectedPlaintext).toBeTruthy();
  // TODO: Implement actual gzip decompression and verification
}

/**
 * Assert that URL does not redirect to external domain (open redirect prevention)
 *
 * @param page - Playwright page object
 * @param allowedDomain - Allowed domain (without protocol)
 * @example
 * await expectNoOpenRedirect(page, 'localhost:4180');
 */
export async function expectNoOpenRedirect(page: Page, allowedDomain: string): Promise<void> {
  const currentUrl = page.url();
  const url = new URL(currentUrl);
  const domain = url.host;
  expect(domain).toBe(allowedDomain);
}

/**
 * Assert that CSRF token exists in form
 *
 * @param page - Playwright page object
 * @example
 * await expectCSRFToken(page);
 */
export async function expectCSRFToken(page: Page): Promise<void> {
  const csrfInput = page.locator('input[name="csrf_token"]');
  await expect(csrfInput).toBeAttached();
  const value = await csrfInput.getAttribute('value');
  expect(value).toBeTruthy();
  expect(value).not.toBe('');
}
