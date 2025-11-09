import { test, expect } from '@playwright/test';

const PASSWORD_BASE_URL = 'http://localhost:4184';

test.describe('Password authentication flow', () => {
  test('user can log in with correct password', async ({ page }) => {
    // Navigate to the app
    await page.goto(PASSWORD_BASE_URL);

    // Should be redirected to login page
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Check that password form is present
    await expect(page.locator('#password-input')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible();

    // Enter the correct password
    await page.locator('#password-input').fill('P@ssW0rd');

    // Click the sign in button
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Should be redirected to the home page after successful authentication
    await expect(page).toHaveURL(PASSWORD_BASE_URL + '/');

    // Check that user is authenticated (should see email in the app)
    await expect(page.locator('[data-test="app-user-email"]')).toContainText('password@localhost');
  });

  test('user cannot log in with incorrect password', async ({ page }) => {
    await page.goto(PASSWORD_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Check that password form is present
    await expect(page.locator('#password-input')).toBeVisible();
    const button = page.getByRole('button', { name: 'Sign In' });
    await expect(button).toBeVisible();

    // Enter an incorrect password
    await page.locator('#password-input').fill('WrongPassword');

    // Listen for the alert dialog
    page.once('dialog', async dialog => {
      expect(dialog.message()).toContain('Invalid password');
      await dialog.accept();
    });

    // Click the sign in button
    await button.click();

    // Should still be on the login page
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('server validates password field', async ({ page, request }) => {
    await page.goto(PASSWORD_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Get cookies for authenticated request
    const cookies = await page.context().cookies();

    // Try to POST without password field (should fail)
    const response1 = await request.post(`${PASSWORD_BASE_URL}/_auth/password/login`, {
      headers: {
        'Content-Type': 'application/json',
        'Cookie': cookies.map(c => `${c.name}=${c.value}`).join('; '),
      },
      data: JSON.stringify({}),
    });
    expect(response1.status()).toBe(400);

    // Try to POST with wrong password (should fail)
    const response2 = await request.post(`${PASSWORD_BASE_URL}/_auth/password/login`, {
      headers: {
        'Content-Type': 'application/json',
        'Cookie': cookies.map(c => `${c.name}=${c.value}`).join('; '),
      },
      data: JSON.stringify({ password: 'WrongPassword' }),
    });
    expect(response2.status()).toBe(401);

    // POST with correct password (should succeed)
    const response3 = await request.post(`${PASSWORD_BASE_URL}/_auth/password/login`, {
      headers: {
        'Content-Type': 'application/json',
        'Cookie': cookies.map(c => `${c.name}=${c.value}`).join('; '),
      },
      data: JSON.stringify({ password: 'P@ssW0rd' }),
    });
    expect(response3.status()).toBe(200);
  });

  test('empty password field is not allowed', async ({ page }) => {
    await page.goto(PASSWORD_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Check that password input has required attribute
    await expect(page.locator('#password-input')).toHaveAttribute('required', '');

    // Try to submit with empty password (HTML5 validation should prevent submission)
    const button = page.getByRole('button', { name: 'Sign In' });
    await button.click();

    // Should still be on login page (browser validation prevents submission)
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });

  test('user stays authenticated across page reloads', async ({ page }) => {
    await page.goto(PASSWORD_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Login with password
    await page.locator('#password-input').fill('P@ssW0rd');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await expect(page).toHaveURL(PASSWORD_BASE_URL + '/');

    // Reload the page
    await page.reload();

    // Should still be on the home page (not redirected to login)
    await expect(page).toHaveURL(PASSWORD_BASE_URL + '/');
    await expect(page.locator('[data-test="app-user-email"]')).toContainText('password@localhost');
  });

  test('user can logout', async ({ page }) => {
    await page.goto(PASSWORD_BASE_URL);
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Login with password
    await page.locator('#password-input').fill('P@ssW0rd');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await expect(page).toHaveURL(PASSWORD_BASE_URL + '/');

    // Logout
    await page.goto(`${PASSWORD_BASE_URL}/_auth/logout`);

    // Should show logout success page
    await expect(page).toHaveURL(/\/_auth\/logout$/);

    // Verify logout message is displayed
    await expect(page.locator('.alert-success')).toBeVisible();

    // Try to access home page again (should redirect to login since session is cleared)
    await page.goto(PASSWORD_BASE_URL);

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);
  });
});
