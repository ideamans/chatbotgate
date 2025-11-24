import { test, expect } from '@playwright/test';

const BASE_URL = 'http://localhost:4180';
const HEALTH_ENDPOINT = '/_auth/health';

/**
 * Health check response interface
 * Matches the Go HealthResponse struct in pkg/middleware/core/handlers.go
 */
interface HealthResponse {
  status: string;      // Current health status (starting/ready/draining/etc.)
  live: boolean;       // Process is alive
  ready: boolean;      // Ready to accept traffic
  since: string;       // ISO8601 timestamp of when middleware started
  detail: string;      // Human-readable detail message
  retry_after?: number; // Retry after N seconds (only present when 503)
}

test.describe('Health check endpoints', () => {
  test('liveness probe returns 200 OK', async ({ request }) => {
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT + '?probe=live');

    // Should always return 200 if process is alive
    expect(response.status()).toBe(200);

    // Should have JSON content type
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Parse response
    const body: HealthResponse = await response.json();

    // Verify liveness probe response
    expect(body.status).toBe('live');
    expect(body.live).toBe(true);
    expect(body.since).toBeTruthy();
    expect(body.detail).toBe('ok');

    // Verify since timestamp is valid ISO8601
    const sinceDate = new Date(body.since);
    expect(sinceDate.toString()).not.toBe('Invalid Date');

    // Since should be in the past
    expect(sinceDate.getTime()).toBeLessThanOrEqual(Date.now());
  });

  test('readiness probe returns 200 when ready', async ({ request }) => {
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT);

    // In E2E environment, middleware should be ready
    expect(response.status()).toBe(200);

    // Should have JSON content type
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');

    // Parse response
    const body: HealthResponse = await response.json();

    // Verify readiness probe response
    expect(body.status).toBe('ready');
    expect(body.live).toBe(true);
    expect(body.ready).toBe(true);
    expect(body.since).toBeTruthy();
    expect(body.detail).toBe('ok');

    // Should not have retry_after when ready (may be null or undefined)
    expect(body.retry_after == null).toBe(true);

    // Verify since timestamp is valid
    const sinceDate = new Date(body.since);
    expect(sinceDate.toString()).not.toBe('Invalid Date');
  });

  test('health check does not require authentication', async ({ request }) => {
    // Health check should work without authentication
    // (no session cookie, no OAuth2 auth)

    const response = await request.get(BASE_URL + HEALTH_ENDPOINT);

    // Should return 200 even without auth
    expect(response.status()).toBe(200);

    const body: HealthResponse = await response.json();
    expect(body.ready).toBe(true);
  });

  test('liveness probe does not require authentication', async ({ request }) => {
    // Liveness probe should work without authentication

    const response = await request.get(BASE_URL + HEALTH_ENDPOINT + '?probe=live');

    // Should return 200 even without auth
    expect(response.status()).toBe(200);

    const body: HealthResponse = await response.json();
    expect(body.live).toBe(true);
  });

  test('health check response includes all required fields', async ({ request }) => {
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT);
    const body: HealthResponse = await response.json();

    // Verify all required fields are present
    expect(body).toHaveProperty('status');
    expect(body).toHaveProperty('live');
    expect(body).toHaveProperty('ready');
    expect(body).toHaveProperty('since');
    expect(body).toHaveProperty('detail');

    // Verify types
    expect(typeof body.status).toBe('string');
    expect(typeof body.live).toBe('boolean');
    expect(typeof body.ready).toBe('boolean');
    expect(typeof body.since).toBe('string');
    expect(typeof body.detail).toBe('string');
  });

  test('liveness probe response includes all required fields', async ({ request }) => {
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT + '?probe=live');
    const body: HealthResponse = await response.json();

    // Verify all required fields are present
    expect(body).toHaveProperty('status');
    expect(body).toHaveProperty('live');
    expect(body).toHaveProperty('ready');
    expect(body).toHaveProperty('since');
    expect(body).toHaveProperty('detail');

    // Verify types
    expect(typeof body.status).toBe('string');
    expect(typeof body.live).toBe('boolean');
    expect(typeof body.ready).toBe('boolean');
    expect(typeof body.since).toBe('string');
    expect(typeof body.detail).toBe('string');
  });

  test('health check supports HEAD requests', async ({ request }) => {
    // Some load balancers use HEAD requests for health checks
    const response = await request.head(BASE_URL + HEALTH_ENDPOINT);

    // Should return 200
    expect(response.status()).toBe(200);

    // Should have JSON content type header
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');
  });

  test('liveness probe supports HEAD requests', async ({ request }) => {
    const response = await request.head(BASE_URL + HEALTH_ENDPOINT + '?probe=live');

    // Should return 200
    expect(response.status()).toBe(200);

    // Should have JSON content type header
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');
  });

  test('multiple health check requests return consistent results', async ({ request }) => {
    // Make multiple requests to ensure consistency
    const responses = await Promise.all([
      request.get(BASE_URL + HEALTH_ENDPOINT),
      request.get(BASE_URL + HEALTH_ENDPOINT),
      request.get(BASE_URL + HEALTH_ENDPOINT),
    ]);

    // All should return 200
    responses.forEach((response) => {
      expect(response.status()).toBe(200);
    });

    // Parse all responses
    const bodies = await Promise.all(responses.map((r) => r.json()));

    // All should have same status and ready state
    bodies.forEach((body: HealthResponse) => {
      expect(body.status).toBe('ready');
      expect(body.ready).toBe(true);
      expect(body.live).toBe(true);
    });

    // All should have the same 'since' timestamp (middleware started at the same time)
    const sinceTimes = bodies.map((b) => b.since);
    expect(sinceTimes[0]).toBe(sinceTimes[1]);
    expect(sinceTimes[1]).toBe(sinceTimes[2]);
  });

  test('concurrent health check requests are handled correctly', async ({ request }) => {
    // Make concurrent requests
    const concurrentRequests = 10;
    const promises = Array.from({ length: concurrentRequests }, () =>
      request.get(BASE_URL + HEALTH_ENDPOINT)
    );

    const responses = await Promise.all(promises);

    // All should succeed
    responses.forEach((response) => {
      expect(response.status()).toBe(200);
    });

    // All should return valid JSON
    const bodies = await Promise.all(responses.map((r) => r.json()));
    bodies.forEach((body: HealthResponse) => {
      expect(body.status).toBe('ready');
      expect(body.ready).toBe(true);
    });
  });

  test('health check with custom auth path prefix', async ({ request }) => {
    // Test with custom prefix configuration (port 4185, prefix: /_oauth2_proxy)
    const customPrefixUrl = 'http://localhost:4185';
    const customHealthEndpoint = '/_oauth2_proxy/health';

    const response = await request.get(customPrefixUrl + customHealthEndpoint);

    // Should return 200
    expect(response.status()).toBe(200);

    const body: HealthResponse = await response.json();
    expect(body.status).toBe('ready');
    expect(body.ready).toBe(true);
  });

  test('health check response time is reasonable', async ({ request }) => {
    // Health checks should be fast (< 100ms)
    const startTime = Date.now();
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT);
    const endTime = Date.now();

    expect(response.status()).toBe(200);

    const responseTime = endTime - startTime;
    // Health check should be fast (allow 500ms for network + processing)
    expect(responseTime).toBeLessThan(500);
  });

  test('liveness probe response time is reasonable', async ({ request }) => {
    // Liveness probes should be very fast
    const startTime = Date.now();
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT + '?probe=live');
    const endTime = Date.now();

    expect(response.status()).toBe(200);

    const responseTime = endTime - startTime;
    // Liveness should be even faster than readiness
    expect(responseTime).toBeLessThan(500);
  });

  test('health check only accepts GET and HEAD methods', async ({ request }) => {
    // GET should work (standard health check method)
    const getResponse = await request.get(BASE_URL + HEALTH_ENDPOINT);
    expect(getResponse.status()).toBe(200);

    // HEAD should work (load balancers often use HEAD for efficiency)
    const headResponse = await request.head(BASE_URL + HEALTH_ENDPOINT);
    expect(headResponse.status()).toBe(200);

    // CRITICAL: POST, PUT, DELETE, PATCH should be REJECTED with 405 Method Not Allowed
    // Health checks must be read-only operations (GET/HEAD only)
    // This follows HTTP spec and Kubernetes/Docker health check conventions
    const postResponse = await request.post(BASE_URL + HEALTH_ENDPOINT, {
      failOnStatusCode: false
    });
    expect(postResponse.status()).toBe(405);

    const putResponse = await request.put(BASE_URL + HEALTH_ENDPOINT, {
      failOnStatusCode: false
    });
    expect(putResponse.status()).toBe(405);

    const deleteResponse = await request.delete(BASE_URL + HEALTH_ENDPOINT, {
      failOnStatusCode: false
    });
    expect(deleteResponse.status()).toBe(405);

    const patchResponse = await request.patch(BASE_URL + HEALTH_ENDPOINT, {
      failOnStatusCode: false
    });
    expect(patchResponse.status()).toBe(405);
  });

  test('health check with invalid probe parameter returns readiness', async ({ request }) => {
    // Invalid probe parameter should default to readiness check
    const response = await request.get(BASE_URL + HEALTH_ENDPOINT + '?probe=invalid');

    expect(response.status()).toBe(200);

    const body: HealthResponse = await response.json();
    // Should behave like readiness probe (default)
    expect(body.status).toBe('ready');
    expect(body.ready).toBe(true);
  });

  test('health check works across different proxy instances', async ({ request }) => {
    // Test multiple proxy instances (different ports)
    const instances = [
      'http://localhost:4180',  // Default
      'http://localhost:4181',  // With whitelist
      'http://localhost:4182',  // With forwarding
    ];

    for (const baseUrl of instances) {
      const response = await request.get(baseUrl + HEALTH_ENDPOINT);

      // All instances should be healthy
      expect(response.status()).toBe(200);

      const body: HealthResponse = await response.json();
      expect(body.status).toBe('ready');
      expect(body.ready).toBe(true);
      expect(body.live).toBe(true);
    }
  });
});
