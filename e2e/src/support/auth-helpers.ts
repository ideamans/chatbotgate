import { Page, expect } from '@playwright/test';
import { waitForLoginEmail, waitForMessage, getMessage, extractOTP } from './mailpit-helper';
import { routeStubAuthRequests } from './stub-auth-route';

/**
 * Common authentication helper functions for E2E tests.
 * Reduces code duplication and improves test maintainability.
 */

export interface AuthOptions {
  baseUrl?: string;
  email?: string;
  password?: string;
}

/**
 * Authenticate via OAuth2 (stub-auth provider)
 *
 * @param page - Playwright page object
 * @param options - Authentication options
 * @returns Promise that resolves when authentication is complete
 *
 * @example
 * await authenticateViaOAuth2(page);
 * await authenticateViaOAuth2(page, { email: 'user@example.com', baseUrl: 'http://localhost:4181' });
 */
export async function authenticateViaOAuth2(
  page: Page,
  options: AuthOptions = {}
): Promise<void> {
  const {
    email = 'someone@example.com',
    password = 'password',
    baseUrl = 'http://localhost:4180'
  } = options;

  // Set up stub-auth request routing
  await routeStubAuthRequests(page);

  // Navigate to the app
  await page.goto(baseUrl + '/');
  await expect(page).toHaveURL(/\/_auth\/login$/);

  // Click OAuth2 provider link
  await page.getByRole('link', { name: 'stub-auth' }).click();
  await expect(page).toHaveURL(/localhost:3001\/login/);

  // Fill in credentials
  await page.locator('[data-test="login-email"]').fill(email);
  await page.locator('[data-test="login-password"]').fill(password);

  // Submit login form
  await Promise.all([
    page.waitForURL(/localhost:3001\/oauth\/authorize/),
    page.locator('[data-test="login-submit"]').click(),
  ]);

  // Authorize the app
  await Promise.all([
    page.waitForURL(new RegExp(baseUrl.replace('http://', ''))),
    page.locator('[data-test="authorize-allow"]').click(),
  ]);
}

/**
 * Authenticate via email link (magic link from Mailpit)
 *
 * @param page - Playwright page object
 * @param email - Email address to send login link to
 * @param options - Additional options
 * @returns Promise that resolves when authentication is complete
 *
 * @example
 * await authenticateViaEmailLink(page, 'user@example.com');
 * await authenticateViaEmailLink(page, 'user@example.com', { baseUrl: 'http://localhost:4181' });
 */
export async function authenticateViaEmailLink(
  page: Page,
  email: string,
  options: { baseUrl?: string } = {}
): Promise<void> {
  const { baseUrl = 'http://localhost:4180' } = options;

  await routeStubAuthRequests(page);

  // Navigate to login page
  await page.goto(baseUrl + '/');
  await expect(page).toHaveURL(/\/_auth\/login$/);

  // Fill in email address
  await page.getByLabel('Email Address').fill(email);

  // Send login link
  await Promise.all([
    page.waitForURL(/\/_auth\/email\/sent/),
    page.getByRole('button', { name: 'Send Login Link' }).click(),
  ]);

  // Wait for email and extract login URL
  const loginUrl = await waitForLoginEmail(email, {
    timeoutMs: 10_000,
    pollIntervalMs: 500,
  });

  // Navigate to the login URL
  await page.goto(loginUrl);

  // Verify authentication succeeded
  await expect(page.locator('[data-test="app-user-email"]')).toContainText(email);
}

/**
 * Authenticate via OTP (One-Time Password from email)
 *
 * @param page - Playwright page object
 * @param email - Email address to send OTP to
 * @param options - Additional options
 * @returns Promise that resolves when authentication is complete
 *
 * @example
 * await authenticateViaOTP(page, 'user@example.com');
 * await authenticateViaOTP(page, 'user@example.com', { baseUrl: 'http://localhost:4181' });
 */
export async function authenticateViaOTP(
  page: Page,
  email: string,
  options: { baseUrl?: string } = {}
): Promise<void> {
  const { baseUrl = 'http://localhost:4180' } = options;

  await routeStubAuthRequests(page);

  // Navigate to login page
  await page.goto(baseUrl + '/');
  await expect(page).toHaveURL(/\/_auth\/login$/);

  // Fill in email address
  await page.getByLabel('Email Address').fill(email);

  // Send login link (which also contains OTP)
  await Promise.all([
    page.waitForURL(/\/_auth\/email\/sent/),
    page.getByRole('button', { name: 'Send Login Link' }).click(),
  ]);

  // Wait for email and extract OTP
  const message = await waitForMessage(email, {
    timeoutMs: 10_000,
    pollIntervalMs: 500,
  });
  const detail = await getMessage(message.ID);
  const otp = extractOTP(detail.Text || detail.HTML);

  if (!otp) {
    throw new Error(`OTP not found in email to ${email}`);
  }

  // Enter OTP
  await page.getByLabel('One-Time Password').fill(otp);

  // Submit OTP
  await Promise.all([
    page.waitForURL(new RegExp(baseUrl.replace('http://', ''))),
    page.getByRole('button', { name: 'Verify Code' }).click(),
  ]);

  // Verify authentication succeeded
  await expect(page.locator('[data-test="app-user-email"]')).toContainText(email);
}

/**
 * Authenticate via password
 *
 * @param page - Playwright page object
 * @param password - Password to authenticate with
 * @param options - Additional options
 * @returns Promise that resolves when authentication is complete
 *
 * @example
 * await authenticateViaPassword(page, 'P@ssW0rd');
 * await authenticateViaPassword(page, 'P@ssW0rd', { baseUrl: 'http://localhost:4184' });
 */
export async function authenticateViaPassword(
  page: Page,
  password: string,
  options: { baseUrl?: string } = {}
): Promise<void> {
  const { baseUrl = 'http://localhost:4184' } = options;

  // Navigate to the app
  await page.goto(baseUrl);
  await expect(page).toHaveURL(/\/_auth\/login$/);

  // Verify password form is visible
  await expect(page.locator('#password-input')).toBeVisible();

  // Enter password
  await page.locator('#password-input').fill(password);

  // Submit password
  await page.getByRole('button', { name: 'Sign In' }).click();

  // Wait for redirect to home page
  await expect(page).toHaveURL(baseUrl + '/');
}

/**
 * Log out from the application
 *
 * @param page - Playwright page object
 * @returns Promise that resolves when logout is complete
 *
 * @example
 * await logout(page);
 */
export async function logout(page: Page): Promise<void> {
  await Promise.all([
    page.waitForURL(/\/_auth\/logout/),
    page.locator('[data-test="oauth-signout"]').click(),
  ]);
}

/**
 * Navigate to a protected path and expect redirect to login
 *
 * @param page - Playwright page object
 * @param path - Protected path to navigate to
 * @param options - Additional options
 * @returns Promise that resolves when navigation is complete
 *
 * @example
 * await navigateToProtectedPath(page, '/dashboard');
 * await navigateToProtectedPath(page, '/api/data', { baseUrl: 'http://localhost:4181' });
 */
export async function navigateToProtectedPath(
  page: Page,
  path: string,
  options: { baseUrl?: string } = {}
): Promise<void> {
  const { baseUrl = 'http://localhost:4180' } = options;

  await page.goto(baseUrl + path);
  await expect(page).toHaveURL(/\/_auth\/login$/);
}

/**
 * Verify user is authenticated by checking for user info on page
 *
 * @param page - Playwright page object
 * @param email - Expected email address
 * @returns Promise that resolves when verification is complete
 *
 * @example
 * await verifyAuthenticated(page, 'user@example.com');
 */
export async function verifyAuthenticated(page: Page, email: string): Promise<void> {
  await expect(page.locator('[data-test="app-user-email"]')).toContainText(email);
}

/**
 * Verify user is authenticated by checking for auth provider header
 *
 * @param page - Playwright page object
 * @param provider - Expected provider name
 * @returns Promise that resolves when verification is complete
 *
 * @example
 * await verifyAuthProvider(page, 'stub-auth');
 */
export async function verifyAuthProvider(page: Page, provider: string): Promise<void> {
  await expect(page.locator('[data-test="auth-provider"]')).toContainText(provider);
}

/**
 * Send login email without completing authentication
 * Useful for testing rate limiting, email content, etc.
 *
 * @param page - Playwright page object
 * @param email - Email address to send login link to
 * @param options - Additional options
 * @returns Promise that resolves when email is sent
 *
 * @example
 * await sendLoginEmail(page, 'user@example.com');
 * await sendLoginEmail(page, 'user@example.com', { baseUrl: 'http://localhost:4181' });
 */
export async function sendLoginEmail(
  page: Page,
  email: string,
  options: { baseUrl?: string } = {}
): Promise<void> {
  const { baseUrl = 'http://localhost:4180' } = options;

  await routeStubAuthRequests(page);

  // Navigate to login page
  await page.goto(baseUrl + '/');
  await expect(page).toHaveURL(/\/_auth\/login$/);

  // Fill in email address
  await page.getByLabel('Email Address').fill(email);

  // Send login link
  await Promise.all([
    page.waitForURL(/\/_auth\/email\/sent/),
    page.getByRole('button', { name: 'Send Login Link' }).click(),
  ]);
}
