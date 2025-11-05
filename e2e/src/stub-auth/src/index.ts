import cookieParser from 'cookie-parser';
import express, { NextFunction, Request, Response } from 'express';
import session from 'express-session';
import morgan from 'morgan';
import { nanoid } from 'nanoid';
import jwt from 'jsonwebtoken';
import { createHash, createPublicKey } from 'crypto';

declare module 'express-session' {
  interface SessionData {
    user?: StubUserSession;
    authorizeRequest?: AuthorizeRequest;
    returnTo?: string;
  }
}

type StubUserSession = {
  email: string;
  name: string;
};

type AuthorizeRequest = {
  responseType: string;
  clientId: string;
  redirectUri: string;
  scope?: string;
  state?: string;
  codeChallenge?: string;
  codeChallengeMethod?: string;
  nonce?: string;
};

type AuthorizationCode = {
  code: string;
  clientId: string;
  redirectUri: string;
  userEmail: string;
  userName: string;
  scope?: string;
  createdAt: Date;
  codeChallenge?: string;
  codeChallengeMethod?: string;
  nonce?: string;
};

type AccessToken = {
  token: string;
  clientId: string;
  userEmail: string;
  userName: string;
  scope?: string;
  createdAt: Date;
  expiresAt: Date;
};

const PORT = Number(process.env.STUB_AUTH_PORT ?? 3001);
const HOST = process.env.STUB_AUTH_HOST ?? '0.0.0.0';
const BASE_URL = process.env.STUB_PUBLIC_URL ?? `http://localhost:${PORT}`;
const CLIENT_ID = process.env.STUB_CLIENT_ID ?? 'stub-client-id';
const CLIENT_SECRET = process.env.STUB_CLIENT_SECRET ?? 'stub-client-secret';
const REDIRECT_URI = process.env.STUB_REDIRECT_URI ?? 'http://proxy-app:4180/_auth/oauth2/callback';
const REDIRECT_URIS = (process.env.STUB_REDIRECT_URIS ?? '')
  .split(',')
  .map((uri) => uri.trim())
  .filter((uri) => uri.length > 0);
const SESSION_SECRET = process.env.STUB_SESSION_SECRET ?? 'stub-session-secret';
const ISSUER = process.env.STUB_ISSUER ?? BASE_URL;
const TOKEN_EXPIRY_SECONDS = Number(process.env.STUB_TOKEN_EXPIRY ?? 300);
const AUTH_CODE_TTL_SECONDS = Number(process.env.STUB_AUTH_CODE_TTL ?? 120);

const JWKS_PATH = '/oauth/jwks';
const JWKS_URI = `${BASE_URL}${JWKS_PATH}`;

const TEST_USER_EMAIL = 'someone@example.com';
const TEST_USER_PASSWORD = 'password';
const TEST_USER_NAME = 'Test User';

// Special test user that doesn't provide email in userinfo endpoint
const NO_EMAIL_USER_EMAIL = 'noemail@example.com';
const NO_EMAIL_USER_PASSWORD = 'password';
const NO_EMAIL_USER_NAME = 'No Email User';

// Test users for whitelist testing
const ALLOWED_EMAIL_USER = 'allowed@example.com';
const ALLOWED_EMAIL_PASSWORD = 'password';
const ALLOWED_EMAIL_NAME = 'Allowed User';

const ALLOWED_DOMAIN_USER = 'user@allowed.example.com';
const ALLOWED_DOMAIN_PASSWORD = 'password';
const ALLOWED_DOMAIN_NAME = 'Allowed Domain User';

const DENIED_USER_EMAIL = 'denied@example.com';
const DENIED_USER_PASSWORD = 'password';
const DENIED_USER_NAME = 'Denied User';

const RSA_PRIVATE_KEY = `-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA0THIgvDUKvAQ7l/7MUoLZ0hjLTzQavdr0lsg5uS0oY3Pic/O\n3nK7K0UBWoFeFmnlYlSPUw59f455EJQvyAdz/pOrCIrumMmSC+mjunBb2pj/JSKW\nIFbksuOf4MydkDWrZPK+Z6mSHo2VVn6GY62HqxqVZkMafVwY1q5VgdJ6W1OyYunj\neSxBJb1u6lBsCkQPwJ3/0u5c8yc0f1jGSCbrMu3kWhufxLQLXaaS+QsvUAZfXri4\n3YdCgwnC0x2vL49vj+tXd2AM3QlQq+Mht0NgWYnG+WfhzzPZIgXOGuhfQdhrEtt6\ndy9yFuz6mO4VIxwvf5jsXt4thkQ0VW17O95aAwIDAQABAoIBAGQgd+B0dJi4nuH5\nfrlwv1SICTA102vfUPQ2OeFJxkstHRDRLiq6r2tec+9lzCreNLSD1LXkUZ0kDD4r\nL1OGfbZz54EHPnxSvlyFT6CE9vICGN0lWMXR0VTuLi/iv+euSILgzNHBD/cfvULQ\n/HHpNO5ooul3ZM5rrlfSyYqBu57J6tqF7ydpEnXmPR0fiHw8OhsuHieSiWSFqMqx\nfebagvjLPoCOwx6V6JlBRASIQI//2DrZRB/qMpXgfTlmxBNDr3u0kB5kj7HCOc0q\nquzQ/UvYWJvksTKXSMJu0s8VSp/8d/pwmJwOmMK+i3sAxgrsTCXCckavF6DaS/qr\nI1NzcoECgYEA891/lzWkMPnRhCMUyq3HyUyYXVaI51qZKbOtrm9B5Q9DXThS1DMY\nFVMVfNrE4/qlTLmdxwBW16/FcoacLdbfq8mOiXMP7pDBCiMSve7bhl9QhRi6KOUM\nBh5U3Z6DyYg6Q8SE+bMHIuF5fCbSn/2nRdRlj1GFmQN8DKhaNnQkCAkCgYEA25qg\nvFJ7Ht6+YsC7ObdUdlCy1jbKx95Kr3PmkSb6OmjBxtLOcgNmfuUTS70ngFi0pDft\n1pA5/pMgo+tSlS2IPgBv50J+4ZetDawZYfV9HlyA6KhbFu/2X3Aj78jpTe3Kna9J\nvyUeaVBKcPvEnTQxaTpC5u8rzilzPT8R2UKMHKsCgYEAzeoYFGv86kXnffXJVqKK\nchU1CotJKmE7txS68PGM6IeM0CgA+KD0Ev2GxVhMrFw2O6T37tMAgTswM9YqBiLL\n1thofPMlXsHn3lFjP/Fyd/H/oYMRnfpZvsjZzBBPI1reJ97GkblzqyZMWGLHssSR\n+8quvueNMXjZxC5bjmNfEVECgYEArAdegQgv8MfW5q9KO3VVEfY3kj2L7rRBV154\nsR6SiO0FV4ZOONxXD3LOAdfkuNNEdxxlEV8cP0PsHty6bagkgUWAY+4gTQKvivVV\nUPqpD/6w8RDpgndqTesgC7gco3Jy9cGaCMXAJAnEtutTYz6+skr0m8miTDcGUmU0\nyzgpYE8CgYB6LLMJriBen5+2CoBj4nETZSfDl4XofMhiqI0HyvAiD4tpfJ3+AT0q\nZa1ZtLKK+EAGWT1bt1TGsG8nAbpDC524p5zGqOGWUQdWFwChLgzV86ElZ+LM10PZ\nLeDNpGVH3/5YX3jZxC1M8J3j/ncqLpPoyJgco69bfkgmWneBmgc3yQ==\n-----END RSA PRIVATE KEY-----\n`;

const RSA_PUBLIC_KEY = `-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0THIgvDUKvAQ7l/7MUoL\nZ0hjLTzQavdr0lsg5uS0oY3Pic/O3nK7K0UBWoFeFmnlYlSPUw59f455EJQvyAdz\n/pOrCIrumMmSC+mjunBb2pj/JSKWIFbksuOf4MydkDWrZPK+Z6mSHo2VVn6GY62H\nqxqVZkMafVwY1q5VgdJ6W1OyYunjeSxBJb1u6lBsCkQPwJ3/0u5c8yc0f1jGSCbr\nMu3kWhufxLQLXaaS+QsvUAZfXri43YdCgwnC0x2vL49vj+tXd2AM3QlQq+Mht0Ng\nWYnG+WfhzzPZIgXOGuhfQdhrEtt6dy9yFuz6mO4VIxwvf5jsXt4thkQ0VW17O95a\nAwIDAQAB\n-----END PUBLIC KEY-----\n`;

const defaultRedirects = REDIRECT_URIS.length > 0 ? REDIRECT_URIS : [REDIRECT_URI];

const clients: ClientMetadata[] = [
  {
    clientId: CLIENT_ID,
    clientSecret: CLIENT_SECRET,
    redirectUris: defaultRedirects,
  },
];

const authorizationCodes = new Map<string, AuthorizationCode>();
const accessTokens = new Map<string, AccessToken>();

const app = express();

app.set('trust proxy', true);
app.use(morgan('dev'));
app.use(express.urlencoded({ extended: true }));
app.use(express.json());
app.use(cookieParser());
app.use(
  session({
    name: 'stub_session',
    secret: SESSION_SECRET,
    resave: false,
    saveUninitialized: false,
    cookie: {
      httpOnly: true,
      sameSite: 'lax',
    },
  })
);

app.get('/health', (_req, res) => {
  res.status(200).send('OK');
});

app.get('/', (_req, res) => {
  res.redirect('/login');
});

app.get(
  '/login',
  asyncHandler((req, res) => {
    const error = req.query.error ? 'メールアドレスまたはパスワードが正しくありません。' : undefined;
    const body = `
      <h1>Stub OAuth2 Provider</h1>
      <p>この画面は OAuth2 認証用のテストサイトです。以下の資格情報を使用してください。</p>
      <div class="notice">
        <p><strong>通常ユーザー:</strong><br>
          メール: <strong>${TEST_USER_EMAIL}</strong> / パスワード: <strong>${TEST_USER_PASSWORD}</strong>
        </p>
        <p><strong>ホワイトリストテスト用ユーザー:</strong><br>
          許可(メール): <strong>${ALLOWED_EMAIL_USER}</strong> / パスワード: <strong>${ALLOWED_EMAIL_PASSWORD}</strong><br>
          許可(ドメイン): <strong>${ALLOWED_DOMAIN_USER}</strong> / パスワード: <strong>${ALLOWED_DOMAIN_PASSWORD}</strong><br>
          拒否: <strong>${DENIED_USER_EMAIL}</strong> / パスワード: <strong>${DENIED_USER_PASSWORD}</strong>
        </p>
        <p><strong>メールアドレスなしユーザー:</strong><br>
          メール: <strong>${NO_EMAIL_USER_EMAIL}</strong> / パスワード: <strong>${NO_EMAIL_USER_PASSWORD}</strong><br>
          <small>※このユーザーは userinfo エンドポイントでメールアドレスを返しません</small>
        </p>
      </div>
      <form method="post" action="/login">
        <label>メールアドレス
          <input name="email" type="email" required autocomplete="email" value="${TEST_USER_EMAIL}" data-test="login-email" />
        </label>
        <label>パスワード
          <input name="password" type="password" required autocomplete="current-password" value="${TEST_USER_PASSWORD}" data-test="login-password" />
        </label>
        <button type="submit" data-test="login-submit">ログイン</button>
        ${error ? `<div class="error" data-test="login-error">${error}</div>` : ''}
      </form>
    `;
    res.send(renderPage('OAuth2 Login', body));
  })
);

app.post(
  '/login',
  asyncHandler((req, res) => {
    const { email, password } = req.body as { email?: string; password?: string };

    // Check for valid test users
    let validUser: StubUserSession | null = null;
    if (email === TEST_USER_EMAIL && password === TEST_USER_PASSWORD) {
      validUser = { email: TEST_USER_EMAIL, name: TEST_USER_NAME };
    } else if (email === NO_EMAIL_USER_EMAIL && password === NO_EMAIL_USER_PASSWORD) {
      validUser = { email: NO_EMAIL_USER_EMAIL, name: NO_EMAIL_USER_NAME };
    } else if (email === ALLOWED_EMAIL_USER && password === ALLOWED_EMAIL_PASSWORD) {
      validUser = { email: ALLOWED_EMAIL_USER, name: ALLOWED_EMAIL_NAME };
    } else if (email === ALLOWED_DOMAIN_USER && password === ALLOWED_DOMAIN_PASSWORD) {
      validUser = { email: ALLOWED_DOMAIN_USER, name: ALLOWED_DOMAIN_NAME };
    } else if (email === DENIED_USER_EMAIL && password === DENIED_USER_PASSWORD) {
      validUser = { email: DENIED_USER_EMAIL, name: DENIED_USER_NAME };
    }

    if (validUser) {
      req.session.user = validUser;
      const redirectTarget =
        req.session.returnTo ??
        (req.session.authorizeRequest ? buildAuthorizePath(req.session.authorizeRequest) : '/');
      req.session.returnTo = undefined;
      return res.redirect(redirectTarget);
    }
    res.redirect('/login?error=1');
  })
);

app.post(
  '/logout',
  asyncHandler((req, res, next) => {
    req.session.destroy((err) => {
      if (err) {
        return next(err);
      }
      res.clearCookie('stub_session');
      res.redirect('/login');
    });
  })
);

app.get(
  '/oauth/authorize',
  asyncHandler((req, res) => {
    const responseType = req.query.response_type as string | undefined;
    const clientId = req.query.client_id as string | undefined;
    const redirectUri = req.query.redirect_uri as string | undefined;
    const scope = req.query.scope as string | undefined;
    const state = req.query.state as string | undefined;
    const codeChallenge = req.query.code_challenge as string | undefined;
    const codeChallengeMethod = req.query.code_challenge_method as string | undefined;
    const nonce = req.query.nonce as string | undefined;

    if (!responseType || responseType !== 'code') {
      return res.status(400).json({ error: 'unsupported_response_type' });
    }
    if (!clientId || !findClient(clientId)) {
      return res.status(400).json({ error: 'invalid_client' });
    }
    if (!redirectUri || !isRedirectUriAllowed(clientId, redirectUri)) {
      return res.status(400).json({ error: 'invalid_redirect_uri' });
    }

    req.session.authorizeRequest = {
      responseType,
      clientId,
      redirectUri,
      scope,
      state,
      codeChallenge,
      codeChallengeMethod,
      nonce,
    };

    if (!req.session.user) {
      req.session.returnTo = req.originalUrl;
      return res.redirect('/login');
    }

    const body = `
      <h1>アクセス許可</h1>
      <p><strong>${clientId}</strong> があなたのアカウントへのアクセスを求めています。</p>
      <form method="post" action="/oauth/authorize">
        <input type="hidden" name="decision" value="allow" />
        <button type="submit" data-test="authorize-allow">許可する</button>
      </form>
      <form method="post" action="/oauth/authorize" style="margin-top: 0.5rem;">
        <input type="hidden" name="decision" value="deny" />
        <button type="submit">拒否する</button>
      </form>
    `;
    res.send(renderPage('Authorize Access', body));
  })
);

app.post(
  '/oauth/authorize',
  requireLogin,
  asyncHandler((req, res) => {
    const decision = (req.body.decision as string | undefined) ?? 'deny';
    const authRequest = req.session.authorizeRequest;
    if (!authRequest) {
      return res.status(400).json({ error: 'session_expired' });
    }

    if (decision !== 'allow') {
      const redirectUrl = new URL(authRequest.redirectUri);
      redirectUrl.searchParams.set('error', 'access_denied');
      if (authRequest.state) {
        redirectUrl.searchParams.set('state', authRequest.state);
      }
      req.session.authorizeRequest = undefined;
      return res.redirect(redirectUrl.toString());
    }

    const code = nanoid(32);
    authorizationCodes.set(code, {
      code,
      clientId: authRequest.clientId,
      redirectUri: authRequest.redirectUri,
      userEmail: req.session.user!.email,
      userName: req.session.user!.name,
      scope: authRequest.scope,
      createdAt: new Date(),
      codeChallenge: authRequest.codeChallenge,
      codeChallengeMethod: authRequest.codeChallengeMethod,
      nonce: authRequest.nonce,
    });

    const redirectUrl = new URL(authRequest.redirectUri);
    redirectUrl.searchParams.set('code', code);
    if (authRequest.state) {
      redirectUrl.searchParams.set('state', authRequest.state);
    }
    req.session.authorizeRequest = undefined;
    req.session.returnTo = undefined;
    res.redirect(redirectUrl.toString());

    setTimeout(() => {
      const record = authorizationCodes.get(code);
      if (record && record.createdAt.getTime() + AUTH_CODE_TTL_SECONDS * 1000 < Date.now()) {
        authorizationCodes.delete(code);
      }
    }, AUTH_CODE_TTL_SECONDS * 1000 + 1000);
  })
);

app.post(
  '/oauth/token',
  asyncHandler((req, res) => {
    const {
      grant_type: grantType,
      code,
      redirect_uri: redirectUri,
      client_id: clientId,
      client_secret: clientSecret,
      code_verifier: codeVerifier,
    } = req.body as Record<string, string | undefined>;

    if (grantType !== 'authorization_code') {
      return res.status(400).json({ error: 'unsupported_grant_type' });
    }
    if (!code || !redirectUri) {
      return res.status(400).json({ error: 'invalid_request' });
    }

    const authCode = authorizationCodes.get(code);
    if (!authCode) {
      return res.status(400).json({ error: 'invalid_grant' });
    }

    if (authCode.redirectUri !== redirectUri) {
      return res.status(400).json({ error: 'invalid_grant' });
    }

    const basicAuth = parseBasicAuthorization(req.header('authorization'));
    const effectiveClientId = clientId ?? basicAuth?.clientId ?? authCode.clientId;
    const validatedClient = effectiveClientId ? findClient(effectiveClientId) : undefined;
    if (!validatedClient) {
      return res.status(401).json({ error: 'invalid_client' });
    }

    if (validatedClient.clientId !== authCode.clientId) {
      return res.status(400).json({ error: 'invalid_grant' });
    }

    const providedSecret = clientSecret ?? basicAuth?.clientSecret;
    if (validatedClient.clientSecret !== providedSecret) {
      return res.status(401).json({ error: 'invalid_client' });
    }

    if (authCode.codeChallenge) {
      const expectedChallenge = deriveCodeChallenge(codeVerifier, authCode.codeChallengeMethod);
      if (!codeVerifier || expectedChallenge !== authCode.codeChallenge) {
        return res.status(400).json({ error: 'invalid_grant', error_description: 'PKCE verification failed' });
      }
    }

    authorizationCodes.delete(code);

    const accessToken = nanoid(32);
    const createdAt = new Date();
    const expiresAt = new Date(createdAt.getTime() + TOKEN_EXPIRY_SECONDS * 1000);
    accessTokens.set(accessToken, {
      token: accessToken,
      clientId: validatedClient.clientId,
      userEmail: authCode.userEmail,
      userName: authCode.userName,
      scope: authCode.scope,
      createdAt,
      expiresAt,
    });

    const idToken = jwt.sign(
      {
        iss: ISSUER,
        sub: authCode.userEmail,
        aud: validatedClient.clientId,
        exp: Math.floor(expiresAt.getTime() / 1000),
        iat: Math.floor(createdAt.getTime() / 1000),
        email: authCode.userEmail,
        nonce: authCode.nonce,
      },
      RSA_PRIVATE_KEY,
      {
        algorithm: 'RS256',
        keyid: 'stub-1',
      }
    );

    res.json({
      access_token: accessToken,
      token_type: 'Bearer',
      expires_in: TOKEN_EXPIRY_SECONDS,
      id_token: idToken,
      scope: authCode.scope ?? 'openid email profile',
    });
  })
);

app.get(
  '/oauth/userinfo',
  asyncHandler((req, res) => {
    const authHeader = req.header('authorization');
    if (!authHeader) {
      return res.status(401).json({ error: 'invalid_request' });
    }

    const [scheme, token] = authHeader.split(' ');
    if (scheme !== 'Bearer' || !token) {
      return res.status(401).json({ error: 'invalid_request' });
    }

    const sessionToken = accessTokens.get(token);
    if (!sessionToken || sessionToken.expiresAt.getTime() < Date.now()) {
      return res.status(401).json({ error: 'invalid_token' });
    }

    // Base response
    const response: any = {
      sub: sessionToken.userEmail,
      name: sessionToken.userName,
    };

    // Special case: noemail@example.com user doesn't provide email
    // This simulates OAuth2 providers that don't provide email address
    if (sessionToken.userEmail !== NO_EMAIL_USER_EMAIL) {
      response.email = sessionToken.userEmail;
      response.email_verified = true;
    }

    // Add additional fields based on scopes
    const scopes = sessionToken.scope?.split(' ') || [];

    // If 'analytics' scope is requested, add custom analytics field
    if (scopes.includes('analytics')) {
      response.secrets = {
        access_token: 'secret-analytics-token-' + nanoid(16),
        refresh_token: 'secret-refresh-token-' + nanoid(16),
      };
      response.analytics = {
        user_id: 'analytics-user-' + sessionToken.userEmail.split('@')[0],
        tier: 'premium',
      };
    }

    // If 'profile' scope is requested, add additional profile fields
    if (scopes.includes('profile')) {
      response.locale = 'ja-JP';
      response.timezone = 'Asia/Tokyo';
    }

    res.json(response);
  })
);

app.get(
  '/.well-known/openid-configuration',
  asyncHandler((_req, res) => {
    res.json({
      issuer: ISSUER,
      authorization_endpoint: `${BASE_URL}/oauth/authorize`,
      token_endpoint: `${BASE_URL}/oauth/token`,
      userinfo_endpoint: `${BASE_URL}/oauth/userinfo`,
      jwks_uri: JWKS_URI,
      response_types_supported: ['code'],
      subject_types_supported: ['public'],
      id_token_signing_alg_values_supported: ['RS256'],
      scopes_supported: ['openid', 'email', 'profile'],
      token_endpoint_auth_methods_supported: ['client_secret_post', 'client_secret_basic'],
      claims_supported: ['sub', 'email', 'email_verified', 'name'],
    });
  })
);

app.get(
  JWKS_PATH,
  asyncHandler((_req, res) => {
    const jwk = createPublicKey(RSA_PUBLIC_KEY).export({ format: 'jwk' });
    res.json({
      keys: [
        {
          ...jwk,
          use: 'sig',
          kid: 'stub-1',
          alg: 'RS256',
        },
      ],
    });
  })
);

app.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
  console.error('Unhandled error', err);
  res.status(500).json({ error: 'server_error', message: 'Internal Server Error' });
});

type ClientMetadata = {
  clientId: string;
  clientSecret: string;
  redirectUris: string[];
};

function findClient(clientId: string): ClientMetadata | undefined {
  return clients.find((candidate) => candidate.clientId === clientId);
}

function isRedirectUriAllowed(clientId: string, redirectUri: string): boolean {
  const client = findClient(clientId);
  if (!client) {
    return false;
  }
  return client.redirectUris.includes(redirectUri);
}

function parseBasicAuthorization(header: string | undefined):
  | { clientId: string; clientSecret: string }
  | undefined {
  if (!header || !header.startsWith('Basic ')) {
    return undefined;
  }
  const base64Credentials = header.slice('Basic '.length);
  const decoded = Buffer.from(base64Credentials, 'base64').toString('utf8');
  const separatorIndex = decoded.indexOf(':');
  if (separatorIndex === -1) {
    return undefined;
  }
  return {
    clientId: decoded.slice(0, separatorIndex),
    clientSecret: decoded.slice(separatorIndex + 1),
  };
}

function deriveCodeChallenge(codeVerifier: string | undefined, method: string | undefined) {
  if (!codeVerifier) {
    return undefined;
  }
  if (!method || method === 'plain') {
    return codeVerifier;
  }
  if (method === 'S256') {
    return createHash('sha256').update(codeVerifier).digest('base64url');
  }
  return undefined;
}

function buildAuthorizePath(request: AuthorizeRequest): string {
  const params = new URLSearchParams();
  params.set('response_type', request.responseType);
  params.set('client_id', request.clientId);
  params.set('redirect_uri', request.redirectUri);

  if (request.scope) {
    params.set('scope', request.scope);
  }
  if (request.state) {
    params.set('state', request.state);
  }
  if (request.codeChallenge) {
    params.set('code_challenge', request.codeChallenge);
  }
  if (request.codeChallengeMethod) {
    params.set('code_challenge_method', request.codeChallengeMethod);
  }
  if (request.nonce) {
    params.set('nonce', request.nonce);
  }

  return `/oauth/authorize?${params.toString()}`;
}

function requireLogin(req: Request, res: Response, next: NextFunction) {
  if (req.session.user) {
    return next();
  }
  req.session.returnTo = req.originalUrl;
  res.redirect('/login');
}

type AsyncHandler = (req: Request, res: Response, next: NextFunction) => Promise<unknown> | unknown;

function asyncHandler(handler: AsyncHandler): AsyncHandler {
  return async (req, res, next) => {
    try {
      await handler(req, res, next);
    } catch (error) {
      next(error);
    }
  };
}

function renderPage(title: string, body: string): string {
  return `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8" />
  <title>${title}</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { font-family: system-ui, sans-serif; max-width: 480px; margin: 2rem auto; padding: 0 1rem; }
    h1 { font-size: 1.6rem; }
    form { display: grid; gap: 0.75rem; margin-top: 1rem; }
    label { display: grid; gap: 0.25rem; font-weight: 600; }
    input[type="email"], input[type="password"] { padding: 0.5rem; font-size: 1rem; }
    button { padding: 0.6rem 1rem; font-size: 1rem; font-weight: 600; cursor: pointer; }
    .notice { background: #f4f6fb; padding: 1rem; border-radius: 0.5rem; }
    .error { color: #c0392b; }
  </style>
</head>
<body>
  ${body}
</body>
</html>`;
}

const server = app.listen(PORT, HOST, () => {
  console.log(`Stub auth provider listening at ${BASE_URL} (host ${HOST})`);
  console.log(`Test users:`);
  console.log(`  - Regular user: ${TEST_USER_EMAIL} / ${TEST_USER_PASSWORD}`);
  console.log(`  - No-email user: ${NO_EMAIL_USER_EMAIL} / ${NO_EMAIL_USER_PASSWORD} (doesn't provide email in userinfo)`);
  console.log(`Client credentials: ${CLIENT_ID} / ${CLIENT_SECRET}`);
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
