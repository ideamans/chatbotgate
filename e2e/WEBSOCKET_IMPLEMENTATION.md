# WebSocket Proxying E2E Tests Implementation

## Summary

Implemented comprehensive WebSocket proxying tests to verify that ChatbotGate correctly proxies WebSocket connections with authentication, as promised in the README.

## Problem

The README and documentation claim WebSocket support:
> "Reverse proxy with **WebSocket support**"

However, there were NO E2E tests validating this critical feature. This implementation gap was identified as a high-priority issue in the E2E test review.

## Solution

### 1. Added WebSocket Support to Target App

**Modified Files:**
- `e2e/src/target-app/package.json` - Added `ws` and `@types/ws` dependencies
- `e2e/src/target-app/src/index.ts` - Implemented WebSocket server

**WebSocket Server Features:**
- Endpoint: `ws://localhost:3000/ws`
- Receives authentication headers from proxy (`x-authenticated`, `x-auth-provider`, `x-chatbotgate-email`)
- Sends welcome message with authentication info
- Echoes back all messages with metadata
- Supports graceful shutdown
- Handles errors gracefully

**Implementation Details:**

```typescript
// WebSocket server on existing HTTP server
const wss = new WebSocketServer({
  server,
  path: '/ws'
});

wss.on('connection', (ws: WebSocket, request: http.IncomingMessage) => {
  // Extract authentication info from headers (forwarded by proxy)
  const isAuthenticated = request.headers['x-authenticated'] === 'true';
  const authProvider = request.headers['x-auth-provider'] || 'unknown';
  const userEmail = request.headers['x-chatbotgate-email'] || 'unknown';

  // Send welcome message
  ws.send(JSON.stringify({
    type: 'welcome',
    authenticated: isAuthenticated,
    provider: authProvider,
    email: userEmail
  }));

  // Echo messages back with metadata
  ws.on('message', (data) => {
    const response = JSON.stringify({
      type: 'echo',
      original: JSON.parse(data.toString()),
      authenticated: isAuthenticated,
      provider: authProvider,
      email: userEmail
    });
    ws.send(response);
  });
});
```

### 2. Created Comprehensive E2E Tests

**New File:** `e2e/src/tests/websocket.spec.ts` (366 lines)

**Test Coverage (6 test scenarios):**

1. **Authenticated user can connect to WebSocket**
   - Authenticates via OAuth2
   - Connects to WebSocket through proxy
   - Verifies welcome message contains authentication info
   - Sends test message and verifies echo response
   - Confirms authentication headers are forwarded to upstream

2. **Unauthenticated user cannot connect to WebSocket**
   - Attempts WebSocket connection without authentication
   - Verifies connection fails (proxy rejects unauthenticated connections)

3. **WebSocket supports bidirectional communication**
   - Establishes WebSocket connection
   - Sends 3 sequential messages
   - Verifies all messages are echoed back
   - Confirms message ordering is preserved

4. **WebSocket forwards authentication headers to upstream**
   - Authenticates with specific user email
   - Connects to WebSocket
   - Verifies upstream receives user email in headers
   - Confirms authentication context is maintained

5. **WebSocket connection closes gracefully**
   - Establishes connection
   - Closes connection cleanly
   - Verifies clean shutdown (wasClean flag)

6. **Multiple WebSocket connections can coexist**
   - Creates 3 simultaneous WebSocket connections
   - Verifies all receive welcome messages
   - Confirms connections don't interfere with each other

## Test Implementation Strategy

### Browser-Based WebSocket Testing

Since Playwright controls a browser, WebSocket connections are tested using the browser's native `WebSocket` API via `page.evaluate()`:

```typescript
async function testWebSocketConnection(
  page: any,
  wsUrl: string,
  testMessage: any
): Promise<WebSocketTestResult> {
  return await page.evaluate(
    async ({ url, msg }: { url: string; msg: any }) => {
      return new Promise<WebSocketTestResult>((resolve) => {
        const ws = new WebSocket(url);

        ws.onopen = () => {
          console.log('Connected');
        };

        ws.onmessage = (event) => {
          const data = JSON.parse(event.data);
          // Handle messages...
        };

        ws.onerror = (error) => {
          resolve({ success: false, error });
        };
      });
    },
    { url: wsUrl, msg: testMessage }
  );
}
```

**Why This Approach:**
- Browser automatically sends authentication cookies with WebSocket upgrade request
- Mimics real-world browser WebSocket usage
- Tests the complete proxy chain including cookie handling
- No need for separate WebSocket client library

### Authentication Flow Testing

```
User → OAuth2 Auth → Authenticated Page → WebSocket Connection
                                          ↓
                                    (Cookies sent automatically)
                                          ↓
                         Proxy → Validates Auth → Forwards to Upstream
                                          ↓
                         Upstream receives: x-authenticated: true
                                           x-auth-provider: stub-auth
                                           x-chatbotgate-email: user@example.com
```

## Proxy Implementation Verification

The tests verify that `pkg/proxy/core/handler.go` correctly:
1. Preserves WebSocket upgrade headers (lines 92-95)
2. Forwards authentication state to upstream
3. Maintains WebSocket connection stability
4. Handles graceful shutdown

**Relevant Proxy Code:**
```go
// Preserve WebSocket upgrade headers
if strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
    req.Header.Set("Connection", "Upgrade")
    req.Header.Set("Upgrade", "websocket")
}
```

## Test Results

All 6 WebSocket tests validate:
- ✅ WebSocket connections work through proxy
- ✅ Authentication is required for WebSocket connections
- ✅ Authentication headers are forwarded to upstream
- ✅ Bidirectional communication works correctly
- ✅ Multiple concurrent connections are supported
- ✅ Graceful shutdown works properly

## Benefits

1. **Closes Critical Test Gap** - README claims WebSocket support, now validated
2. **Prevents Regressions** - Future proxy changes won't break WebSocket support
3. **Validates Real-World Usage** - Tests actual browser WebSocket behavior
4. **Documents Expected Behavior** - Tests serve as executable documentation

## Metrics

- **New test file**: 366 lines
- **Test scenarios**: 6 comprehensive tests
- **Coverage areas**: Authentication, bidirectional communication, concurrency, error handling
- **Lines of target-app code added**: ~120 lines (WebSocket server implementation)

## Next Steps (From E2E Review)

**Completed ✅**
1. Common helper functions
2. Custom assertions
3. **WebSocket proxying tests** (this implementation)

**Remaining High Priority 🔥**
1. OAuth2 error handling tests (user denial, invalid code, timeouts)
2. Health check endpoint tests
3. Concurrent access tests

**Medium Priority ⏳**
1. Server-Sent Events (SSE) tests
2. Session timeout tests
3. Large payload forwarding tests

## Dependencies Added

```json
{
  "dependencies": {
    "ws": "^8.18.0"
  },
  "devDependencies": {
    "@types/ws": "^8.5.12"
  }
}
```

## Running the Tests

```bash
# Run all e2e tests including WebSocket
cd e2e
make test

# Run only WebSocket tests
cd e2e
npx playwright test websocket.spec.ts

# Run with visible browser
cd e2e
npx playwright test websocket.spec.ts --headed
```

## Implementation Notes

1. **Timeouts**: All WebSocket operations have 10-second timeouts to prevent hanging tests
2. **Message Protocol**: Uses JSON for structured communication (type + payload)
3. **Error Handling**: Comprehensive error handling in both client and server
4. **Cleanup**: All WebSocket connections are properly closed after tests
5. **Concurrent Tests**: Tests are isolated using unique email addresses

## Verification

The implementation successfully tests the complete WebSocket proxying chain:
```
Browser WebSocket Client
    ↓ (with auth cookies)
ChatbotGate Proxy
    ↓ (upgrade headers preserved)
    ↓ (auth headers forwarded)
Target App WebSocket Server
    ↓ (receives auth context)
    ↓ (echo response)
ChatbotGate Proxy
    ↓ (transparent passthrough)
Browser WebSocket Client
    ↓ (receives response)
Test Assertion ✅
```

This validates that ChatbotGate fulfills its promise of WebSocket support as documented in the README.
