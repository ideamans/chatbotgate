# Target App

Protected sample application consumed by the end-to-end environment. It renders the user identity forwarded by `multi-oauth2-proxy` and exposes a sign-out button that posts to `/_auth/logout`.

## Usage

```bash
cd e2e/src/target-app
npm install
npm run build
npm start
```

The server listens on `http://localhost:3000` by default and expects to be accessed via the proxy. Set the following environment variables when required:

- `TARGET_APP_PORT`
- `TARGET_APP_HOST`
- `TARGET_APP_NAME`
