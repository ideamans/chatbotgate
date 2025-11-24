import { test, expect } from '@playwright/test';
import { clearAllMessages } from '../support/mailpit-helper';
import { routeStubAuthRequests } from '../support/stub-auth-route';

test.describe('Email authentication rate limiting', () => {
  test.beforeEach(async ({ page }) => {
    await clearAllMessages();
    await routeStubAuthRequests(page);
  });

  test('should enforce rate limit after multiple login attempts', async ({ page }) => {
    // Use a unique email for this test with random component to avoid parallel test conflicts
    const TEST_EMAIL = `ratelimit-test-${Date.now()}-${Math.random().toString(36).substring(7)}@example.com`;

    await page.goto('/');
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Default rate limit is 5 per minute
    // Send 5 login requests (should all succeed)
    for (let i = 0; i < 5; i++) {
      await page.getByLabel('Email Address').fill(TEST_EMAIL);
      await Promise.all([
        page.waitForURL(/\/_auth\/email\/sent/),
        page.getByRole('button', { name: 'Send Login Link' }).click(),
      ]);

      // Go back to login page for next attempt
      if (i < 4) {
        await page.goto('/_auth/login');
      }
    }

    // 6th request should be rate limited
    await page.goto('/_auth/login');
    await page.getByLabel('Email Address').fill(TEST_EMAIL);
    await page.getByRole('button', { name: 'Send Login Link' }).click();

    // Should show rate limit error
    await expect(page.locator('body')).toContainText(/rate limit|too many requests/i, { timeout: 5000 });
  });

  test('rate limit is per email address', async ({ page }) => {
    // Use unique emails with random components to avoid parallel test conflicts
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(7);
    const EMAIL1 = `ratelimit-user1-${timestamp}-${random}@example.com`;
    const EMAIL2 = `ratelimit-user2-${timestamp}-${random}@example.com`;

    await page.goto('/');

    // Send 5 requests for EMAIL1 (should all succeed)
    for (let i = 0; i < 5; i++) {
      await page.getByLabel('Email Address').fill(EMAIL1);
      await Promise.all([
        page.waitForURL(/\/_auth\/email\/sent/),
        page.getByRole('button', { name: 'Send Login Link' }).click(),
      ]);
      await page.goto('/_auth/login');
    }

    // EMAIL1 should be rate limited now
    await page.getByLabel('Email Address').fill(EMAIL1);
    await page.getByRole('button', { name: 'Send Login Link' }).click();
    await expect(page.locator('body')).toContainText(/rate limit|too many requests/i, { timeout: 5000 });

    // But EMAIL2 should still work (rate limit is per email address)
    await page.goto('/_auth/login');
    await page.getByLabel('Email Address').fill(EMAIL2);
    await Promise.all([
      page.waitForURL(/\/_auth\/email\/sent/),
      page.getByRole('button', { name: 'Send Login Link' }).click(),
    ]);

    // EMAIL2 login should succeed
    await expect(page).toHaveURL(/\/_auth\/email\/sent/);
  });
});
