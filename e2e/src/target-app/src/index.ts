import express from 'express';
import morgan from 'morgan';
import crypto from 'crypto';

const app = express();

const PORT = Number(process.env.TARGET_APP_PORT ?? 3000);
const HOST = process.env.TARGET_APP_HOST ?? '0.0.0.0';
const APP_NAME = process.env.TARGET_APP_NAME ?? 'Target Application';

app.use(morgan('dev'));

// Encryption key for decrypting forwarded user info
const ENCRYPTION_KEY = process.env.FORWARDING_ENCRYPTION_KEY ?? 'e2e-test-encryption-key-32-chars-long-1234567890';

interface UserInfo {
  username?: string;
  email?: string;
}

/**
 * Decrypt AES-256-GCM encrypted data
 * Format: base64(nonce + ciphertext + tag)
 * This matches the Go implementation in pkg/forwarding/encryption.go
 */
function decrypt(encryptedBase64: string): string {
  try {
    // Hash the key with SHA-256 to get a consistent 32-byte key
    const keyHash = crypto.createHash('sha256').update(ENCRYPTION_KEY).digest();

    // Decode base64
    const encrypted = Buffer.from(encryptedBase64, 'base64');

    // Extract nonce (12 bytes), ciphertext, and tag (16 bytes)
    const nonceSize = 12;
    const tagSize = 16;

    if (encrypted.length < nonceSize + tagSize) {
      throw new Error('Encrypted data too short');
    }

    const nonce = encrypted.subarray(0, nonceSize);
    const ciphertext = encrypted.subarray(nonceSize, encrypted.length - tagSize);
    const tag = encrypted.subarray(encrypted.length - tagSize);

    // Decrypt using AES-256-GCM
    const decipher = crypto.createDecipheriv('aes-256-gcm', keyHash, nonce);
    decipher.setAuthTag(tag);

    const decrypted = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
    return decrypted.toString('utf8');
  } catch (error) {
    console.error('Decryption failed:', error);
    throw error;
  }
}

/**
 * Decode value - try decryption first, fall back to plain text
 * This handles both encrypted and plain text values
 */
function decodeValue(value: string): string {
  if (!value) return '';

  try {
    // Try decryption first
    return decrypt(value);
  } catch (error) {
    // If decryption fails, assume it's plain text
    return value;
  }
}

const renderPage = (title: string, body: string): string => `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${title}</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 640px; margin: 2rem auto; padding: 0 1rem; }
    header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
    main { display: grid; gap: 1.25rem; }
    .card { padding: 1.25rem; border-radius: 0.75rem; background: #f6f8fb; border: 1px solid #dbe4ff; }
    .meta { font-size: 0.95rem; color: #4a5568; }
    form { display: inline-block; }
    button { cursor: pointer; padding: 0.6rem 1.1rem; font-size: 1rem; border-radius: 9999px; border: none; background: #1d4ed8; color: #fff; }
    button:hover { background: #1e40af; }
    a { color: #1d4ed8; }
  </style>
</head>
<body>
  ${body}
</body>
</html>`;

app.get('/health', (_req, res) => {
  res.status(200).send('OK');
});

// Passthrough test endpoints - these should be accessible without authentication
app.get('/embed.js', (_req, res) => {
  res.setHeader('Content-Type', 'application/javascript');
  res.status(200).send("// Embed widget script\nconsole.log('ChatbotGate embed widget loaded');");
});

app.get('/public/data.json', (_req, res) => {
  res.json({
    message: 'public data',
    status: 'ok',
  });
});

app.get('/static/image.png', (_req, res) => {
  // Return a 1x1 transparent PNG
  const png = Buffer.from([
    0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
    0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
    0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
    0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
    0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
    0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
    0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
    0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
    0x42, 0x60, 0x82,
  ]);
  res.setHeader('Content-Type', 'image/png');
  res.status(200).send(png);
});

app.get('/api/public/info', (_req, res) => {
  res.json({
    api: 'public',
    version: '1.0',
    authenticated: false,
  });
});

// Catch-all route to handle all paths (except /health and passthrough endpoints)
app.get('*', (req, res) => {
  const isAuthenticated = req.header('x-authenticated') === 'true';
  const authProvider = req.header('x-auth-provider') ?? 'unknown';

  if (!isAuthenticated) {
    const body = `
      <header>
        <h1>${APP_NAME}</h1>
        <div class="meta">Status: <strong data-test="app-status">Unauthenticated</strong></div>
      </header>
      <main>
        <div class="card">
          <p>このページは ChatbotGate の背後で保護されています。<br/>プロキシ経由でアクセスするとユーザー情報が表示されます。</p>
          <p><a href="/_auth/login" data-test="app-login-link">認証ページへ移動</a></p>
        </div>
      </main>
    `;
    res.status(401).send(renderPage(APP_NAME, body));
    return;
  }

  // Extract forwarding data from querystring (individual parameters)
  let querystringData: UserInfo | null = null;
  const querystringUser = req.query['chatbotgate.user'] as string | undefined;
  const querystringEmail = req.query['chatbotgate.email'] as string | undefined;

  if (querystringUser || querystringEmail) {
    querystringData = {};
    if (querystringUser) {
      querystringData.username = decodeValue(querystringUser);
    }
    if (querystringEmail) {
      querystringData.email = decodeValue(querystringEmail);
    }
  }

  // Extract forwarding data from headers (individual headers)
  let headerData: UserInfo | null = null;
  const forwardedUserHeader = req.header('x-chatbotgate-user');
  const forwardedEmailHeader = req.header('x-chatbotgate-email');

  if (forwardedUserHeader || forwardedEmailHeader) {
    headerData = {};
    if (forwardedUserHeader) {
      headerData.username = decodeValue(forwardedUserHeader);
    }
    if (forwardedEmailHeader) {
      headerData.email = decodeValue(forwardedEmailHeader);
    }
  }

  // Determine display values from forwarding headers
  const emailDisplay = headerData?.email || '(email not provided)';
  const nameDisplay = headerData?.username || '(name not provided)';

  // Build the page with 3 separate sections
  let infoSections = '';

  // 1. Authentication Status Headers
  infoSections += '<div class="card">';
  infoSections += '<h2>1. Authentication Status Headers</h2>';
  infoSections += '<p><strong>Source:</strong> ChatbotGate sets these headers to indicate authentication status</p>';
  infoSections += `<p data-test="auth-status">Authenticated (X-Authenticated): ${isAuthenticated ? 'true' : 'false'}</p>`;
  infoSections += `<p data-test="auth-provider">Provider (X-Auth-Provider): ${authProvider}</p>`;
  infoSections += '</div>';

  // 2. Forwarding Headers (X-ChatbotGate-*)
  if (headerData) {
    infoSections += '<div class="card">';
    infoSections += '<h2>2. Forwarding Headers (X-ChatbotGate-*)</h2>';
    infoSections += '<p><strong>Source:</strong> X-ChatbotGate-User and X-ChatbotGate-Email headers</p>';
    infoSections += '<p><strong>Note:</strong> Only sent when <code>forwarding.header.enabled: true</code></p>';
    infoSections += '<p><strong>Encryption:</strong> Can be encrypted or plain text depending on configuration</p>';
    infoSections += `<p data-test="forwarding-header-username">Username: ${headerData.username || '(empty)'}</p>`;
    infoSections += `<p data-test="forwarding-header-email">Email: ${headerData.email || '(empty)'}</p>`;
    infoSections += '</div>';
  } else {
    infoSections += '<div class="card">';
    infoSections += '<h2>2. Forwarding Headers (X-ChatbotGate-*)</h2>';
    infoSections += '<p><strong>Source:</strong> X-ChatbotGate-User and X-ChatbotGate-Email headers</p>';
    infoSections += '<p><strong>Note:</strong> Only sent when <code>forwarding.header.enabled: true</code></p>';
    infoSections += '<p data-test="forwarding-header-not-present">(Headers not present or decryption failed)</p>';
    infoSections += '</div>';
  }

  // 3. Forwarding QueryString (chatbotgate.*)
  if (querystringData) {
    infoSections += '<div class="card">';
    infoSections += '<h2>3. Forwarding QueryString (chatbotgate.*)</h2>';
    infoSections += '<p><strong>Source:</strong> chatbotgate.user and chatbotgate.email query parameters</p>';
    infoSections += '<p><strong>Note:</strong> Only sent when <code>forwarding.querystring.enabled: true</code></p>';
    infoSections += '<p><strong>Encryption:</strong> Can be encrypted or plain text depending on configuration</p>';
    infoSections += `<p data-test="forwarding-qs-username">Username: ${querystringData.username || '(empty)'}</p>`;
    infoSections += `<p data-test="forwarding-qs-email">Email: ${querystringData.email || '(empty)'}</p>`;
    infoSections += '</div>';
  } else {
    infoSections += '<div class="card">';
    infoSections += '<h2>3. Forwarding QueryString (chatbotgate.*)</h2>';
    infoSections += '<p><strong>Source:</strong> chatbotgate.user and chatbotgate.email query parameters</p>';
    infoSections += '<p><strong>Note:</strong> Only sent when <code>forwarding.querystring.enabled: true</code></p>';
    infoSections += '<p data-test="forwarding-qs-not-present">(QueryString parameters not present or decryption failed)</p>';
    infoSections += '</div>';
  }

  const body = `
    <header>
      <h1>${APP_NAME}</h1>
      <div class="meta">
        認証済みユーザー: <strong data-test="app-user-email">${emailDisplay}</strong>
        / Name: <strong data-test="app-user-name">${nameDisplay}</strong>
      </div>
    </header>
    <main>
      <div class="card" data-test="app-content">
        <h2>User Information Display</h2>
        <p>This page shows user information from different sources:</p>
        <ul>
          <li><strong>X-Authenticated, X-Auth-Provider</strong>: Authentication status (always plain text)</li>
          <li><strong>X-ChatbotGate-*</strong>: User info forwarding via headers (configured in <code>forwarding.header</code>)</li>
          <li><strong>chatbotgate.*</strong>: User info forwarding via querystring (configured in <code>forwarding.querystring</code>)</li>
        </ul>
        <p><strong>Current user</strong>: ${emailDisplay} (${nameDisplay}), Provider: ${authProvider}</p>
        <form method="post" action="/_auth/logout">
          <button type="submit" data-test="oauth-signout">サインアウト</button>
        </form>
      </div>
      ${infoSections}
    </main>
  `;
  res.send(renderPage(APP_NAME, body));
});

const server = app.listen(PORT, HOST, () => {
  console.log(`Target app listening on http://${HOST}:${PORT}`);
});

// Graceful shutdown handler
const shutdown = (signal: string) => {
  console.log(`\n${signal} received. Shutting down gracefully...`);
  server.close(() => {
    console.log('HTTP server closed');
    process.exit(0);
  });

  // Force shutdown after 5 seconds if graceful shutdown fails
  setTimeout(() => {
    console.error('Could not close connections in time, forcefully shutting down');
    process.exit(1);
  }, 5000);
};

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));
