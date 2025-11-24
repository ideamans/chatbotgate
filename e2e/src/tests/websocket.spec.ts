import { test, expect } from '@playwright/test';
import { authenticateViaOAuth2 } from '../support/auth-helpers';

const BASE_URL = 'http://localhost:4180';
const TEST_EMAIL = 'someone@example.com';
const TEST_PASSWORD = 'password';

/**
 * WebSocket test helper - executes WebSocket operations in browser context
 */
interface WebSocketTestResult {
  success: boolean;
  error?: string;
  messages?: any[];
  welcomeMessage?: any;
}

/**
 * Connect to WebSocket and test bidirectional communication
 * This runs in the browser context where authentication cookies are available
 */
async function testWebSocketConnection(
  page: any,
  wsUrl: string,
  testMessage: any
): Promise<WebSocketTestResult> {
  return await page.evaluate(
    async ({ url, msg }: { url: string; msg: any }) => {
      return new Promise<WebSocketTestResult>((resolve) => {
        const messages: any[] = [];
        let welcomeMessage: any = null;
        const ws = new WebSocket(url);
        let connected = false;

        const timeout = setTimeout(() => {
          ws.close();
          resolve({
            success: false,
            error: 'WebSocket connection timeout',
            messages,
          });
        }, 10000); // 10 second timeout

        ws.onopen = () => {
          console.log('WebSocket connected');
          connected = true;
        };

        ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            messages.push(data);

            // First message should be welcome message
            if (messages.length === 1) {
              welcomeMessage = data;
              // Send test message after receiving welcome
              ws.send(JSON.stringify(msg));
            } else if (messages.length === 2) {
              // Second message should be echo response
              clearTimeout(timeout);
              ws.close();
              resolve({
                success: true,
                messages,
                welcomeMessage,
              });
            }
          } catch (error: any) {
            clearTimeout(timeout);
            ws.close();
            resolve({
              success: false,
              error: `Failed to parse message: ${error.message}`,
              messages,
            });
          }
        };

        ws.onerror = (error) => {
          clearTimeout(timeout);
          resolve({
            success: false,
            error: 'WebSocket connection error',
            messages,
          });
        };

        ws.onclose = () => {
          console.log('WebSocket closed');
          if (!connected) {
            clearTimeout(timeout);
            resolve({
              success: false,
              error: 'WebSocket connection failed to establish',
              messages,
            });
          }
        };
      });
    },
    { url: wsUrl, msg: testMessage }
  );
}

test.describe('WebSocket proxying', () => {
  test('authenticated user can connect to WebSocket', async ({ page }) => {
    // First authenticate via OAuth2
    await authenticateViaOAuth2(page, {
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
      baseUrl: BASE_URL,
    });

    // Verify authentication succeeded
    await expect(page.locator('[data-test="auth-provider"]')).toContainText('stub-auth');

    // Connect to WebSocket through proxy
    // Browser will send authentication cookies automatically
    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';
    const testMessage = { type: 'test', content: 'Hello WebSocket!' };

    const result = await testWebSocketConnection(page, wsUrl, testMessage);

    // Verify connection succeeded
    expect(result.success).toBe(true);
    expect(result.error).toBeUndefined();

    // Verify welcome message contains authentication info
    expect(result.welcomeMessage).toBeDefined();
    expect(result.welcomeMessage.type).toBe('welcome');
    expect(result.welcomeMessage.authenticated).toBe(true);
    expect(result.welcomeMessage.provider).toBe('stub-auth');

    // Verify echo response
    expect(result.messages).toHaveLength(2);
    const echoMsg = result.messages![1];
    expect(echoMsg.type).toBe('echo');
    expect(echoMsg.original).toEqual(testMessage);
    expect(echoMsg.authenticated).toBe(true);
    expect(echoMsg.provider).toBe('stub-auth');
  });

  test('unauthenticated user cannot connect to WebSocket', async ({ page }) => {
    // Try to connect without authentication
    // This should fail or redirect to login

    // First, visit a page to establish browser context
    await page.goto(BASE_URL + '/');

    // Should be redirected to login
    await expect(page).toHaveURL(/\/_auth\/login$/);

    // Try to connect to WebSocket (will fail because no auth cookies)
    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';
    const testMessage = { type: 'test', content: 'Hello' };

    const result = await testWebSocketConnection(page, wsUrl, testMessage);

    // Connection should fail (proxy should reject unauthenticated WebSocket connections)
    // Note: Depending on proxy implementation, this might timeout or close immediately
    expect(result.success).toBe(false);
  });

  test('WebSocket supports bidirectional communication', async ({ page }) => {
    // Authenticate first
    await authenticateViaOAuth2(page, {
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
      baseUrl: BASE_URL,
    });

    // Connect to WebSocket
    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';

    // Test multiple message exchanges
    const result = await page.evaluate(async ({ url }: { url: string }) => {
      return new Promise<{ success: boolean; messageCount: number; error?: string }>((resolve) => {
        const ws = new WebSocket(url);
        let messageCount = 0;
        const testMessages = [
          { type: 'test', content: 'Message 1' },
          { type: 'test', content: 'Message 2' },
          { type: 'test', content: 'Message 3' },
        ];
        let currentMessageIndex = 0;

        const timeout = setTimeout(() => {
          ws.close();
          resolve({ success: false, messageCount, error: 'Timeout' });
        }, 10000);

        ws.onopen = () => {
          console.log('WebSocket connected for bidirectional test');
        };

        ws.onmessage = (event) => {
          messageCount++;
          const data = JSON.parse(event.data);

          if (data.type === 'welcome') {
            // After welcome, send first test message
            ws.send(JSON.stringify(testMessages[0]));
          } else if (data.type === 'echo') {
            // Received echo, send next message or finish
            if (currentMessageIndex < testMessages.length - 1) {
              currentMessageIndex++;
              ws.send(JSON.stringify(testMessages[currentMessageIndex]));
            } else {
              // All messages sent and received
              clearTimeout(timeout);
              ws.close();
              resolve({ success: true, messageCount });
            }
          }
        };

        ws.onerror = () => {
          clearTimeout(timeout);
          resolve({ success: false, messageCount, error: 'WebSocket error' });
        };
      });
    }, { url: wsUrl });

    expect(result.success).toBe(true);
    // Should receive: 1 welcome + 3 echo responses = 4 messages
    expect(result.messageCount).toBe(4);
  });

  test('WebSocket forwards authentication headers to upstream', async ({ page }) => {
    // Authenticate with specific user
    await authenticateViaOAuth2(page, {
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
      baseUrl: BASE_URL,
    });

    // Connect to WebSocket
    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';
    const testMessage = { type: 'header-test' };

    const result = await testWebSocketConnection(page, wsUrl, testMessage);

    expect(result.success).toBe(true);

    // Verify that welcome message contains user email from forwarded headers
    // Note: This depends on the upstream WebSocket server echoing back the headers
    expect(result.welcomeMessage).toBeDefined();
    expect(result.welcomeMessage.authenticated).toBe(true);

    // The echo message should also contain authentication info
    const echoMsg = result.messages![1];
    expect(echoMsg.authenticated).toBe(true);
    expect(echoMsg.provider).toBe('stub-auth');
  });

  test('WebSocket connection closes gracefully', async ({ page }) => {
    // Authenticate
    await authenticateViaOAuth2(page, {
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
      baseUrl: BASE_URL,
    });

    // Connect and then close WebSocket
    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';

    const result = await page.evaluate(async ({ url }: { url: string }) => {
      return new Promise<{ connectionEstablished: boolean; closedCleanly: boolean }>((resolve) => {
        const ws = new WebSocket(url);
        let connectionEstablished = false;

        ws.onopen = () => {
          connectionEstablished = true;
          // Close immediately after connection
          setTimeout(() => ws.close(), 100);
        };

        ws.onclose = (event) => {
          resolve({
            connectionEstablished,
            closedCleanly: event.wasClean,
          });
        };

        ws.onerror = () => {
          resolve({
            connectionEstablished: false,
            closedCleanly: false,
          });
        };

        setTimeout(() => {
          if (ws.readyState !== WebSocket.CLOSED) {
            ws.close();
          }
          resolve({
            connectionEstablished,
            closedCleanly: false,
          });
        }, 5000);
      });
    }, { url: wsUrl });

    expect(result.connectionEstablished).toBe(true);
    expect(result.closedCleanly).toBe(true);
  });

  test('multiple WebSocket connections can coexist', async ({ page, context }) => {
    // Authenticate
    await authenticateViaOAuth2(page, {
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
      baseUrl: BASE_URL,
    });

    // Create multiple WebSocket connections
    const wsUrl = BASE_URL.replace('http://', 'ws://') + '/ws';

    const result = await page.evaluate(async ({ url }: { url: string }) => {
      return new Promise<{ connections: number; allSucceeded: boolean }>((resolve) => {
        const websockets: WebSocket[] = [];
        const welcomeReceived: boolean[] = [false, false, false];
        let connectionCount = 0;

        const checkCompletion = () => {
          if (welcomeReceived.every((received) => received)) {
            // All connections received welcome messages
            websockets.forEach((ws) => ws.close());
            resolve({
              connections: connectionCount,
              allSucceeded: true,
            });
          }
        };

        // Create 3 WebSocket connections
        for (let i = 0; i < 3; i++) {
          const ws = new WebSocket(url);
          const index = i;

          ws.onopen = () => {
            connectionCount++;
          };

          ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'welcome') {
              welcomeReceived[index] = true;
              checkCompletion();
            }
          };

          ws.onerror = () => {
            websockets.forEach((ws) => ws.close());
            resolve({
              connections: connectionCount,
              allSucceeded: false,
            });
          };

          websockets.push(ws);
        }

        // Timeout after 10 seconds
        setTimeout(() => {
          websockets.forEach((ws) => ws.close());
          resolve({
            connections: connectionCount,
            allSucceeded: false,
          });
        }, 10000);
      });
    }, { url: wsUrl });

    expect(result.connections).toBe(3);
    expect(result.allSucceeded).toBe(true);
  });
});
