import { Page } from '@playwright/test';

const PUBLIC_STUB_AUTH_ORIGIN = 'http://localhost:3001';
const INTERNAL_STUB_AUTH_ORIGIN = process.env.STUB_AUTH_BASE_URL ?? 'http://stub-auth:3001';

export async function routeStubAuthRequests(page: Page): Promise<void> {
  // When running Playwright on the host (not in Docker), stub-auth is accessible at localhost:3001
  // No route rewriting is needed. Only rewrite if STUB_AUTH_BASE_URL is explicitly set.
  if (process.env.STUB_AUTH_BASE_URL && INTERNAL_STUB_AUTH_ORIGIN !== PUBLIC_STUB_AUTH_ORIGIN) {
    await page.route(`${PUBLIC_STUB_AUTH_ORIGIN}/**`, (route) => {
      const targetUrl = route
        .request()
        .url()
        .replace(PUBLIC_STUB_AUTH_ORIGIN, INTERNAL_STUB_AUTH_ORIGIN);

      route.continue({ url: targetUrl });
    });
  }
}
