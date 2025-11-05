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

- `4180`: chatbotgate (デフォルト設定)
- `4181`: chatbotgate (ホワイトリスト有効)
- `4182`: chatbotgate (ユーザー情報転送機能有効・暗号化あり)
- `3000`: 保護対象アプリ (`target-app`)
- `3001`: OAuth2 プロバイダー (`stub-auth`) — `http://localhost:3001` で直接アクセス可能
- `8025`: Mailpit Web UI (メール確認用)

## ユーザー情報転送機能のテスト (Forwarding Feature Tests)

ポート `4182` で動作するプロキシインスタンスは、ユーザー情報転送機能が有効になっており、以下の機能をテストできます:

### 機能概要

1. **QueryString転送**: 認証後のリダイレクト時に、ユーザー名とメールアドレスを暗号化してURLクエリパラメータ `_user` に追加
2. **Header転送**: すべてのプロキシリクエストに、暗号化されたユーザー情報を `X-Chatbotgate-User` ヘッダーとして追加
3. **暗号化**: AES-256-GCM方式でユーザー情報を暗号化

### 設定ファイル

- `config/proxy.e2e.with-forwarding.yaml`: 転送機能有効の設定
- 暗号化キー: `e2e-test-encryption-key-32-chars-long-1234567890`

### ターゲットアプリの対応

`target-app` は転送されたユーザー情報を復号化して表示します:

- QueryStringからの復号化 (`_user` パラメータ)
- Headerからの復号化 (`X-Chatbotgate-User` ヘッダー)
- 復号化したユーザー名とメールアドレスを画面に表示

### テストシナリオ

`src/tests/forwarding.spec.ts` には以下のテストが含まれています:

1. **OAuth2認証でのユーザー情報転送**: OAuth2プロバイダー経由でログインし、ユーザー名とメールアドレスの両方が転送されることを確認
2. **メール認証でのユーザー情報転送**: メール認証でログインし、ユーザー名が空でメールアドレスのみが転送されることを確認

### 動作確認

```bash
# E2Eテストを実行
make test

# または、ブラウザを表示して確認
make test HEADLESS=false

# 手動確認（開発環境）
make dev
# ブラウザで http://localhost:4182 にアクセス
```
