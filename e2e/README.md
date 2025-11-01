# E2E Harness

This directory contains the end-to-end test harness for `multi-oauth2-proxy`.

## Components

- `src/target-app/`: Express ベースの保護対象アプリ (ポート 3000)。`multi-oauth2-proxy` の背後で動作します。
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
make test       # build images and run Playwright flows
make test-down  # tear down containers and volumes from make test
```

`make test` mounts `e2e/tmp` and `e2e/test-results` so OTPs and Playwright reports remain on the host for inspection. Clean up artifacts with `make clean-e2e` if required.

### ポート割り当て

- `4180`: multi-oauth2-proxy (フロントエンド)
- `3000`: 保護対象アプリ (`target-app`)
- `3001`: OAuth2 プロバイダー (`stub-auth`) — `http://localhost:3001` で直接アクセス可能
