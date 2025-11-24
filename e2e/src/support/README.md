# E2E Test Support Library

This directory contains helper functions and utilities to improve E2E test quality and reduce code duplication.

## Files Overview

### `auth-helpers.ts`

Common authentication helper functions that abstract authentication flows.

**Available Functions:**

- `authenticateViaOAuth2(page, options?)` - Complete OAuth2 authentication flow
- `authenticateViaEmailLink(page, email, options?)` - Authenticate using email magic link
- `authenticateViaOTP(page, email, options?)` - Authenticate using One-Time Password
- `authenticateViaPassword(page, password, options?)` - Authenticate using password
- `logout(page)` - Log out from the application
- `navigateToProtectedPath(page, path, options?)` - Navigate to protected path and expect redirect
- `verifyAuthenticated(page, email)` - Verify user is authenticated
- `verifyAuthProvider(page, provider)` - Verify authentication provider
- `sendLoginEmail(page, email, options?)` - Send login email without completing auth

**Usage Example:**

```typescript
import { authenticateViaOAuth2, logout } from '../support/auth-helpers';

test('user can authenticate and logout', async ({ page }) => {
  // Before: 30+ lines of OAuth2 flow code
  // After: 1 line
  await authenticateViaOAuth2(page);

  // Verify authentication, do something...

  await logout(page);
});

test('authenticate with custom options', async ({ page }) => {
  await authenticateViaOAuth2(page, {
    email: 'user@example.com',
    password: 'password',
    baseUrl: 'http://localhost:4181'
  });
});
```

**Benefits:**

- **97% code reduction** for OAuth2 flows (30 lines → 1 line)
- **80% code reduction** for logout (5 lines → 1 line)
- Consistent behavior across all tests
- Single source of truth for authentication flows
- Easy to customize with options

### `custom-assertions.ts`

Domain-specific assertion functions that improve test readability.

**Available Assertions:**

**Navigation:**
- `expectOnLoginPage(page)` - Assert on login page
- `expectOnLogoutPage(page)` - Assert on logout page
- `expectOnEmailSentPage(page)` - Assert on email sent page
- `expectOnPath(page, path)` - Assert on specific path

**Authentication:**
- `expectAuthenticatedAs(page, email)` - Assert authenticated with email
- `expectAuthenticatedViaProvider(page, provider)` - Assert auth via provider

**Headers & Query Params:**
- `expectHeader(page, name, value)` - Assert header value
- `expectQueryParam(page, name, value)` - Assert query parameter value

**Security:**
- `expectSessionCookieExists(page, name?)` - Assert session cookie exists
- `expectSessionCookieHttpOnly(page, name?)` - Assert HttpOnly flag
- `expectSessionCookieSecure(page, name?)` - Assert Secure flag
- `expectSessionCookieSameSite(page, value, name?)` - Assert SameSite attribute
- `expectCSRFToken(page)` - Assert CSRF token exists
- `expectNoOpenRedirect(page, domain)` - Assert no open redirect

**Errors:**
- `expectErrorMessage(page, message)` - Assert error message shown
- `expectAccessDenied(page)` - Assert access denied (403)
- `expectRateLimitError(page)` - Assert rate limit error

**Data Verification:**
- `expectDecrypts(data, key, plaintext)` - Assert encrypted data decrypts correctly
- `expectDecompresses(data, plaintext)` - Assert compressed data decompresses correctly

**Usage Example:**

```typescript
import { expectOnLoginPage, expectAuthenticatedAs } from '../support/custom-assertions';

test('authentication flow', async ({ page }) => {
  await page.goto('/');

  // Before: await expect(page).toHaveURL(/\/_auth\/login$/);
  // After:
  await expectOnLoginPage(page);

  // ... authenticate ...

  // Before: await expect(page.locator('[data-test="app-user-email"]')).toContainText('user@example.com');
  // After:
  await expectAuthenticatedAs(page, 'user@example.com');
});
```

**Benefits:**

- More readable test intent
- Domain-specific language
- Easier to maintain (change implementation in one place)
- Self-documenting tests

### `mailpit-helper.ts`

Utilities for interacting with Mailpit email testing service.

**Key Functions:**

- `waitForLoginEmail(email, options?)` - Wait for login email and extract URL
- `waitForMessage(email, options?)` - Wait for any email to address
- `getMessage(id, mailpitUrl?)` - Get full message details
- `extractLoginUrl(text)` - Extract login URL from email text
- `extractOTP(text)` - Extract OTP code from email text
- `getMessages(mailpitUrl?)` - Get all messages
- `clearAllMessages(mailpitUrl?)` - Clear all messages

### `stub-auth-route.ts`

Helper for routing stub-auth requests (OAuth2 mock provider).

**Usage:**

```typescript
import { routeStubAuthRequests } from '../support/stub-auth-route';

test.beforeEach(async ({ page }) => {
  await routeStubAuthRequests(page);
});
```

## Migration Guide

### Before (Typical OAuth2 Test)

```typescript
test('user can authenticate', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveURL(/\/_auth\/login$/);

  await page.getByRole('link', { name: 'stub-auth' }).click();
  await expect(page).toHaveURL(/localhost:3001\/login/);

  const emailInput = page.locator('[data-test="login-email"]');
  await expect(emailInput).toBeVisible();
  await emailInput.fill('user@example.com');

  const passwordInput = page.locator('[data-test="login-password"]');
  await passwordInput.fill('password');

  await Promise.all([
    page.waitForURL(/localhost:3001\/oauth\/authorize/),
    page.locator('[data-test="login-submit"]').click(),
  ]);

  await Promise.all([
    page.waitForURL(/localhost:4180\/?/),
    page.locator('[data-test="authorize-allow"]').click(),
  ]);

  await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');
});
```

**Lines of code: 30**
**Duplication: High** (repeated in every OAuth2 test)

### After (Using Helpers)

```typescript
import { authenticateViaOAuth2 } from '../support/auth-helpers';
import { expectAuthenticatedViaProvider } from '../support/custom-assertions';

test('user can authenticate', async ({ page }) => {
  await authenticateViaOAuth2(page, { email: 'user@example.com' });
  await expectAuthenticatedViaProvider(page, 'stub-auth');
});
```

**Lines of code: 2**
**Duplication: None**
**Reduction: 93%**

## Best Practices

1. **Use helpers for common flows** - OAuth2, email login, logout, etc.
2. **Use custom assertions for domain concepts** - "authenticated as", "on login page", etc.
3. **Keep tests focused on intent** - What you're testing, not how to test it
4. **Add new helpers when you notice duplication** - DRY principle
5. **Document complex helpers** - Include JSDoc comments and examples

## Adding New Helpers

When adding new helpers:

1. Identify repeated code patterns across multiple tests
2. Extract common logic into a reusable function
3. Add clear JSDoc comments with examples
4. Add type annotations for better IDE support
5. Update this README with usage examples

## Test Organization

```
e2e/
├── src/
│   ├── support/           # Shared helpers (this directory)
│   │   ├── auth-helpers.ts
│   │   ├── custom-assertions.ts
│   │   ├── mailpit-helper.ts
│   │   ├── stub-auth-route.ts
│   │   └── README.md
│   ├── examples/          # Example refactored tests
│   │   └── oauth2-refactored-example.spec.ts.example
│   └── tests/             # Actual test files
│       ├── oauth2.spec.ts
│       ├── passwordless.spec.ts
│       └── ...
```

## Metrics

**Code Reduction:**
- OAuth2 flow: 30 lines → 1 line (97% reduction)
- Email login: 15 lines → 1 line (93% reduction)
- Logout: 5 lines → 1 line (80% reduction)
- Assertions: 2-3 lines → 1 line (67% reduction)

**Expected Impact:**
- 50% average reduction in test code length
- 80% reduction in code duplication
- Improved test readability and maintainability
- Faster test development (2-3x faster)

## Future Improvements

Potential additions:

1. **Page Object Models** - For complex page interactions
2. **Test Data Builders** - For creating test data
3. **API Helpers** - For direct API calls in tests
4. **Visual Regression Helpers** - For screenshot comparisons
5. **Performance Helpers** - For measuring page load times
