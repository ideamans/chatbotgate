import { test, expect } from '@playwright/test';
import { routeStubAuthRequests } from '../support/stub-auth-route';

const TEST_EMAIL = 'test@example.com';

test.describe('Email address save functionality', () => {
  test.beforeEach(async ({ page, context }) => {
    // Clear localStorage before each test
    await context.clearCookies();

    // Setup routing before navigation
    await routeStubAuthRequests(page);

    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Clear localStorage
    await page.evaluate(() => {
      localStorage.clear();
    });
  });

  test('without save checkbox: email address is cleared after reload', async ({ page }) => {
    // Fill in email address without checking the save checkbox
    const emailInput = page.getByLabel('Email Address');
    await emailInput.fill(TEST_EMAIL);

    // Verify email is filled
    await expect(emailInput).toHaveValue(TEST_EMAIL);

    // Reload the page
    await page.reload();
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Email should be cleared after reload
    await expect(emailInput).toHaveValue('');

    // Verify localStorage is empty
    const savedEmail = await page.evaluate(() => localStorage.getItem('saved_email'));
    const saveEnabled = await page.evaluate(() => localStorage.getItem('save_email_enabled'));
    expect(savedEmail).toBeNull();
    expect(saveEnabled).toBeNull();
  });

  test('with save checkbox: email address is preserved after reload', async ({ page }) => {
    // Check the save checkbox
    const saveCheckbox = page.locator('#save-email-checkbox');
    await saveCheckbox.check();

    // Verify checkbox is checked
    await expect(saveCheckbox).toBeChecked();

    // Fill in email address
    const emailInput = page.getByLabel('Email Address');
    await emailInput.fill(TEST_EMAIL);

    // Wait for localStorage to be updated
    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('saved_email'));
    }).toBe(TEST_EMAIL);

    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('save_email_enabled'));
    }).toBe('true');

    // Reload the page
    await page.reload();
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Email and checkbox state should be preserved after reload
    await expect(emailInput).toHaveValue(TEST_EMAIL);
    await expect(saveCheckbox).toBeChecked();
  });

  test('uncheck save checkbox: email address is cleared after reload', async ({ page }) => {
    // First, check the save checkbox and fill email
    const saveCheckbox = page.locator('#save-email-checkbox');
    await saveCheckbox.check();

    const emailInput = page.getByLabel('Email Address');
    await emailInput.fill(TEST_EMAIL);

    // Wait for localStorage to be updated
    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('saved_email'));
    }).toBe(TEST_EMAIL);

    // Now uncheck the save checkbox
    await saveCheckbox.uncheck();

    // Wait for localStorage to be cleared
    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('saved_email'));
    }).toBeNull();

    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('save_email_enabled'));
    }).toBeNull();

    // Email input should still have the value (not cleared immediately)
    await expect(emailInput).toHaveValue(TEST_EMAIL);

    // Reload the page
    await page.reload();
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Email and checkbox should be cleared after reload
    await expect(emailInput).toHaveValue('');
    await expect(saveCheckbox).not.toBeChecked();
  });

  test('save checkbox persists email on input change (real-time save)', async ({ page }) => {
    // Check the save checkbox
    const saveCheckbox = page.locator('#save-email-checkbox');
    await saveCheckbox.check();

    const emailInput = page.getByLabel('Email Address');

    // Type email character by character to test real-time saving
    await emailInput.type('t', { delay: 50 });
    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('saved_email'));
    }).toBe('t');

    await emailInput.type('est@', { delay: 50 });
    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('saved_email'));
    }).toBe('test@');

    await emailInput.type('example.com', { delay: 50 });
    await expect.poll(async () => {
      return await page.evaluate(() => localStorage.getItem('saved_email'));
    }).toBe(TEST_EMAIL);

    // Reload and verify
    await page.reload();
    await expect(page).toHaveURL(/\/_auth\/login$/);
    await expect(emailInput).toHaveValue(TEST_EMAIL);
    await expect(saveCheckbox).toBeChecked();
  });
});
