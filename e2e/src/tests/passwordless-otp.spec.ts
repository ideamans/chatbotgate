import { test, expect } from '@playwright/test';
import { waitForMessage, getMessage, extractOTP } from '../support/mailpit-helper';
import { routeStubAuthRequests } from '../support/stub-auth-route';

test.describe('Passwordless OTP flow', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page);
  });

  test('user can log in via OTP code from email', async ({ page }) => {
    const TEST_EMAIL = 'otp-success@example.com';
    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Fill in email and send login link
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // Wait for email and extract OTP
    console.log(`Waiting for email to ${TEST_EMAIL}...`);
    const message = await waitForMessage(TEST_EMAIL, {
      timeoutMs: 10_000,
      pollIntervalMs: 500,
    });

    const detail = await getMessage(message.ID);

    // Extract OTP from email text or HTML
    let otp = extractOTP(detail.Text);
    if (!otp && detail.HTML) {
      otp = extractOTP(detail.HTML);
    }

    expect(otp).toBeTruthy();
    expect(otp).toHaveLength(12);
    expect(otp).toMatch(/^[A-Z0-9]{12}$/);
    console.log(`Extracted OTP: ${otp}`);

    // Verify that the OTP input field exists on the email sent page
    const otpInput = page.locator('input[name="otp"]');
    await expect(otpInput).toBeVisible();

    // Verify button is initially disabled
    const verifyButton = page.getByRole('button', { name: 'Verify Code' });
    await expect(verifyButton).toBeDisabled();

    // Enter the OTP code
    await otpInput.fill(otp!);

    // Verify that the input turns green (validation passes)
    await expect(otpInput).toHaveCSS('border-color', /rgb\(16, 185, 129\)/); // Success color (emerald-500)

    // Verify button becomes enabled (stays btn-primary blue)
    await expect(verifyButton).toBeEnabled();

    // Submit the OTP form
    await Promise.all([
      page.waitForURL(/^((?!\/_auth).)*$/), // Wait for redirect away from auth pages
      verifyButton.click(),
    ]);

    // Verify we're logged in
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);
  });

  test('OTP with spaces is accepted', async ({ page }) => {
    const TEST_EMAIL = 'otp-spaces@example.com';
    await page.goto('/');

    // Send login email
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Wait for email and extract OTP
    const message = await waitForMessage(TEST_EMAIL);
    const detail = await getMessage(message.ID);
    const otp = extractOTP(detail.Text) || extractOTP(detail.HTML);

    expect(otp).toBeTruthy();
    console.log(`OTP: ${otp}`);

    // Enter OTP with spaces (like it appears in the email)
    const otpWithSpaces = `${otp!.slice(0, 4)} ${otp!.slice(4, 8)} ${otp!.slice(8, 12)}`;
    const otpInput = page.locator('input[name="otp"]');
    const verifyButton = page.getByRole('button', { name: 'Verify Code' });

    // Button should be disabled initially
    await expect(verifyButton).toBeDisabled();

    await otpInput.fill(otpWithSpaces);

    // Should still turn green (validation normalizes spaces)
    await expect(otpInput).toHaveCSS('border-color', /rgb\(16, 185, 129\)/);

    // Button should be enabled (stays btn-primary blue)
    await expect(verifyButton).toBeEnabled();

    // Submit and verify login
    await verifyButton.click();
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);
  });

  test('invalid OTP shows error and redirects back', async ({ page }) => {
    const TEST_EMAIL = 'otp-invalid@example.com';
    await page.goto('/');

    // Send login email
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Wait for the email sent page
    await expect(page).toHaveURL(/\/_auth\/email\/sent/);

    // Enter an invalid OTP (wrong format - too short)
    const otpInput = page.locator('input[name="otp"]');
    await otpInput.fill('INVALID');

    // Should NOT turn green (validation fails)
    await expect(otpInput).not.toHaveCSS('border-color', /rgb\(16, 185, 129\)/);

    // Try to submit anyway
    await page.getByRole('button', { name: 'Verify Code' }).click();

    // Should redirect back to email sent page with error
    await expect(page).toHaveURL(/\/_auth\/email\/sent/);
  });

  test('correct format but wrong OTP code fails authentication', async ({ page }) => {
    const TEST_EMAIL = 'otp-wrong-code@example.com';
    await page.goto('/');

    // Send login email to generate a real OTP
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    await expect(page).toHaveURL(/\/_auth\/email\/sent/);

    // Enter a valid-looking but incorrect OTP
    const wrongOTP = 'ABCDEFGH1234'; // Valid format but wrong code
    const otpInput = page.locator('input[name="otp"]');
    await otpInput.fill(wrongOTP);

    // Input should turn green (format is valid)
    await expect(otpInput).toHaveCSS('border-color', /rgb\(16, 185, 129\)/);

    // Submit the wrong OTP
    await page.getByRole('button', { name: 'Verify Code' }).click();

    // Should redirect back to email sent page (authentication failed)
    await expect(page).toHaveURL(/\/_auth\/email\/sent/);

    // Should NOT be logged in
    await page.goto('/');
    // If not logged in, should redirect to login page
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('OTP cannot be reused', async ({ page }) => {
    const TEST_EMAIL = 'otp-reuse@example.com';
    await page.goto('/');

    // Send login email
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Get OTP from email
    const message = await waitForMessage(TEST_EMAIL);
    const detail = await getMessage(message.ID);
    const otp = extractOTP(detail.Text) || extractOTP(detail.HTML);
    expect(otp).toBeTruthy();

    // Use OTP once
    await page.locator('input[name="otp"]').fill(otp!);
    await page.getByRole('button', { name: 'Verify Code' }).click();
    await expect(page.locator('[data-test="app-user-email"]')).toContainText(TEST_EMAIL);

    // Open a new page context (simulating a different session/browser)
    const newPage = await page.context().newPage();

    // Go to email sent page directly
    await newPage.goto('/_auth/email/sent');

    // Try to use the same OTP in the new context
    await newPage.locator('input[name="otp"]').fill(otp!);
    await newPage.getByRole('button', { name: 'Verify Code' }).click();

    // Should fail - redirect back to email sent page with error
    await expect(newPage).toHaveURL(/\/_auth\/email\/sent/);

    // Clean up
    await newPage.close();
  });

  test('email contains OTP code', async ({ page }) => {
    const TEST_EMAIL = 'otp-email-content@example.com';
    await page.goto('/');

    // Send login email
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Get email and verify OTP is present
    const message = await waitForMessage(TEST_EMAIL);
    const detail = await getMessage(message.ID);

    // Verify OTP in text version
    const otpFromText = extractOTP(detail.Text);
    expect(otpFromText).toBeTruthy();
    expect(otpFromText).toHaveLength(12);
    expect(otpFromText).toMatch(/^[A-Z0-9]{12}$/);

    // Verify OTP in HTML version
    const otpFromHTML = extractOTP(detail.HTML);
    expect(otpFromHTML).toBeTruthy();
    expect(otpFromHTML).toEqual(otpFromText); // Same OTP in both versions

    // Verify HTML contains the OTP label text
    expect(detail.HTML).toContain('enter this code');

    console.log('OTP in email:', otpFromText);
  });
});
