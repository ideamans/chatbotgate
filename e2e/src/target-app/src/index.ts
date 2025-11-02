import express from 'express';
import morgan from 'morgan';

const app = express();

const PORT = Number(process.env.TARGET_APP_PORT ?? 3000);
const HOST = process.env.TARGET_APP_HOST ?? '0.0.0.0';
const APP_NAME = process.env.TARGET_APP_NAME ?? 'Target Application';

app.use(morgan('dev'));

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

app.get('/', (req, res) => {
  const isAuthenticated = req.header('x-authenticated') === 'true';
  const forwardedEmail = req.header('x-forwarded-email') || '';
  const forwardedName = req.header('x-forwarded-name') || '';
  const forwardedProvider = req.header('x-auth-provider') ?? 'unknown';
  const forwardedUser = req.header('x-forwarded-user') ?? forwardedEmail ?? 'guest';

  if (!isAuthenticated) {
    const body = `
      <header>
        <h1>${APP_NAME}</h1>
        <div class="meta">Status: <strong data-test="app-status">Unauthenticated</strong></div>
      </header>
      <main>
        <div class="card">
          <p>このページは multi-oauth2-proxy の背後で保護されています。<br/>プロキシ経由でアクセスするとユーザー情報が表示されます。</p>
          <p><a href="/_auth/login" data-test="app-login-link">認証ページへ移動</a></p>
        </div>
      </main>
    `;
    res.status(401).send(renderPage(APP_NAME, body));
    return;
  }

  // Email might be empty for OAuth2 providers that don't provide email
  const emailDisplay = forwardedEmail || '(email not provided)';
  const nameDisplay = forwardedName || '(name not provided)';

  const body = `
    <header>
      <h1>${APP_NAME}</h1>
      <div class="meta">認証済みユーザー: <strong data-test="app-user-email">${emailDisplay}</strong></div>
    </header>
    <main>
      <div class="card" data-test="app-content">
        <p data-test="app-user-name">Name: ${nameDisplay}</p>
        <p data-test="app-user-login">Login: ${forwardedUser}</p>
        <p data-test="app-auth-provider">Provider: ${forwardedProvider}</p>
        <form method="post" action="/_auth/logout">
          <button type="submit" data-test="oauth-signout">サインアウト</button>
        </form>
      </div>
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
