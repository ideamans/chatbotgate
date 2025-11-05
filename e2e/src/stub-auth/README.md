# Stub Auth Server

OAuth2/OIDC compatible stub provider used for e2e scenarios. The implementation matches the flow described in `../E2E.md` and exposes:

- Login form at `/login`
  - Regular user: `someone@example.com` / `password` (provides email in userinfo)
  - No-email user: `noemail@example.com` / `password` (does NOT provide email in userinfo - for testing email requirement scenarios)
  - Whitelist test users (all provide email in userinfo):
    - Allowed by email: `allowed@example.com` / `password`
    - Allowed by domain: `user@allowed.example.com` / `password`
    - Denied: `denied@example.com` / `password`
- Authorization Code flow endpoints under `/oauth/*`
- OIDC discovery document at `/.well-known/openid-configuration`

## Usage

```bash
cd e2e/src/stub-auth
npm install
npm run build
npm start
```

The server listens on `http://localhost:3001` by default (and the port is published in `make dev`). Configure credentials via environment variables when needed:

- `STUB_AUTH_PORT`, `STUB_AUTH_HOST`
- `STUB_PUBLIC_URL`
- `STUB_CLIENT_ID`, `STUB_CLIENT_SECRET`
- `STUB_REDIRECT_URI`
- `STUB_REDIRECT_URIS` (comma separated list if multiple values are needed)
- `STUB_SESSION_SECRET`, `STUB_ISSUER`

Client and user credentials are fixed defaults for deterministic testing.
