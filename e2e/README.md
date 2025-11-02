# E2E Harness

This directory contains the end-to-end test harness for `chatbotgate`.

## Components

- `src/target-app/`: Express ベースの保護対象アプリ (ポート 3000)。`chatbotgate` の背後で動作します。
- `src/stub-auth/`: OAuth2/OIDC スタブプロバイダー (ポート 3001)。ブラウザから `http://localhost:3001` で直接ログイン画面を確認できます。
- `config/`: プロキシ・ターゲットアプリ・スタブプロバイダー向けの環境設定。
- `docker/`: Dockerfile および Compose スタック (`make dev`, `make test`)。
- `src/tests/`: Playwright の E2E シナリオ (OAuth2・パスワードレス)。
- `tmp/`: OTP JSONL の共有ボリューム (`passwordless-otp.jsonl`)。
- `test-results/`: Playwright のアーティファクト (スクリーンショット、トレース、レポート)。

## Typical workflow

From this directory (`e2e/`):

```bash
make dev        # proxy + stub for manual verification
make dev-down   # stop the stack
make test       # build images and run Playwright flows (headless by default)
make test-down  # tear down containers and volumes from make test
```

**Note:** The `stub-auth` and `target-app` services implement graceful shutdown handlers, allowing containers to terminate quickly (< 1 second) when receiving SIGTERM signals.

`make test` mounts `e2e/tmp` and `e2e/test-results` so OTPs and Playwright reports remain on the host for inspection. Clean up artifacts with `make clean-e2e` if required.

To observe the browser while running tests, pass `HEADLESS=false` (or `PLAYWRIGHT_HEADLESS=false`) when invoking `make test`:

```bash
make test HEADLESS=false
```

### ポート割り当て

- `4180`: chatbotgate (フロントエンド)
- `3000`: 保護対象アプリ (`target-app`)
- `3001`: OAuth2 プロバイダー (`stub-auth`) — `http://localhost:3001` で直接アクセス可能
