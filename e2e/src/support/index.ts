/**
 * E2E Test Support Library
 *
 * Re-exports all helper functions and utilities for convenient importing.
 *
 * @example
 * // Instead of multiple imports:
 * import { authenticateViaOAuth2 } from '../support/auth-helpers';
 * import { expectOnLoginPage } from '../support/custom-assertions';
 *
 * // You can now use:
 * import { authenticateViaOAuth2, expectOnLoginPage } from '../support';
 */

// Authentication helpers
export {
  authenticateViaOAuth2,
  authenticateViaEmailLink,
  authenticateViaOTP,
  authenticateViaPassword,
  logout,
  navigateToProtectedPath,
  verifyAuthenticated,
  verifyAuthProvider,
  sendLoginEmail,
  type AuthOptions,
} from './auth-helpers';

// Custom assertions
export {
  expectOnLoginPage,
  expectOnLogoutPage,
  expectOnEmailSentPage,
  expectAuthenticatedAs,
  expectAuthenticatedViaProvider,
  expectOnPath,
  expectHeader,
  expectQueryParam,
  expectErrorMessage,
  expectSessionCookieExists,
  expectSessionCookieNotExists,
  expectSessionCookieHttpOnly,
  expectSessionCookieSecure,
  expectSessionCookieSameSite,
  expectAccessDenied,
  expectRateLimitError,
  expectDecrypts,
  expectDecompresses,
  expectNoOpenRedirect,
  expectCSRFToken,
} from './custom-assertions';

// Mailpit helpers
export {
  waitForLoginEmail,
  waitForMessage,
  getMessage,
  extractLoginUrl,
  extractOTP,
  getMessages,
  clearAllMessages,
  type MailpitMessage,
  type MailpitMessageDetail,
  type MailpitMessagesResponse,
} from './mailpit-helper';

// Stub auth routing
export { routeStubAuthRequests } from './stub-auth-route';
