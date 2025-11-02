# Mailpit を使った E2E テスト

## 概要

E2E環境では、実際のメール送信をテストするために [Mailpit](https://github.com/axllent/mailpit) を使用しています。
Mailpitは軽量なメールサーバーで、以下の機能を提供します：

- SMTPサーバー（ポート1025）でメールを受信
- Web UI（http://localhost:8025）でメールをプレビュー
- REST APIでメール内容をプログラムから取得

## 手動テスト

### 1. E2E環境の起動

```bash
docker compose -f e2e/docker/docker-compose.e2e.yaml up --build
```

### 2. ブラウザでテスト

1. プロキシにアクセス: http://localhost:4180
2. メールアドレスを入力してログインリンクを送信
3. **Mailpit Web UI** を開く: http://localhost:8025
4. 受信したメールを確認
5. メール内のログインリンクをクリックしてログイン

### 3. Mailpit Web UIの使い方

- **メール一覧**: 送信されたすべてのメールが表示されます
- **メール詳細**: メールをクリックすると、HTML/テキスト両方のプレビューが可能
- **リンクのクリック**: メール内のリンクをクリックして直接アクセス可能
- **削除**: 個別メールまたは全メールを削除可能

## Playwright テストでの使用方法

### 基本的な使い方

```typescript
import { waitForLoginEmail, clearAllMessages } from '../support/mailpit-helper';

test('email login flow', async ({ page }) => {
  // テスト開始前にメールをクリア
  await clearAllMessages();

  // メール送信をトリガー
  await page.getByLabel('Email Address').fill('test@example.com');
  await page.getByRole('button', { name: 'Send Login Link' }).click();

  // メールを待機してログインURLを取得
  const loginUrl = await waitForLoginEmail('test@example.com');

  // URLに直接アクセス
  await page.goto(loginUrl);

  // ログインできたことを確認
  await expect(page.locator('[data-test="app-user-email"]')).toContainText('test@example.com');
});
```

### トークンの抽出

```typescript
import { waitForLoginEmail } from '../support/mailpit-helper';

test('extract token from email', async ({ page }) => {
  // メール送信
  await page.getByLabel('Email Address').fill('test@example.com');
  await page.getByRole('button', { name: 'Send Login Link' }).click();

  // ログインURLを取得
  const loginUrl = await waitForLoginEmail('test@example.com');

  // URLからトークンを抽出
  const url = new URL(loginUrl);
  const token = url.searchParams.get('token');

  console.log('Token:', token);

  // トークンを使用してログイン
  await page.goto(`/_auth/email/verify?token=${token}`);
});
```

### メール内容の検証

```typescript
import { waitForMessage, getMessage } from '../support/mailpit-helper';

test('verify email content', async ({ page }) => {
  // メール送信
  await page.getByLabel('Email Address').fill('test@example.com');
  await page.getByRole('button', { name: 'Send Login Link' }).click();

  // メールを待機
  const message = await waitForMessage('test@example.com');

  // メールの詳細を取得
  const detail = await getMessage(message.ID);

  // メール内容を検証
  expect(message.Subject).toContain('Login Link');
  expect(message.From.Address).toBe('noreply@example.com');
  expect(detail.Text).toContain('/_auth/email/verify?token=');
  expect(detail.HTML).toContain('Log In');
});
```

## 利用可能なヘルパー関数

### `waitForLoginEmail(email, options?)`

メールを待機してログインURLを自動的に抽出します。

```typescript
const loginUrl = await waitForLoginEmail('test@example.com', {
  timeoutMs: 10_000,      // タイムアウト（デフォルト: 30秒）
  pollIntervalMs: 500,    // ポーリング間隔（デフォルト: 500ms）
  mailpitUrl: 'http://mailpit:8025'  // Mailpit URL（デフォルト: 環境変数またはデフォルトURL）
});
```

### `waitForMessage(email, options?)`

指定したメールアドレス宛のメールを待機します。

```typescript
const message = await waitForMessage('test@example.com', {
  timeoutMs: 10_000,
  pollIntervalMs: 500,
});
```

### `getMessage(id, mailpitUrl?)`

メールIDから詳細情報を取得します。

```typescript
const detail = await getMessage(message.ID);
console.log('Subject:', detail.Subject);
console.log('Text:', detail.Text);
console.log('HTML:', detail.HTML);
```

### `getMessages(mailpitUrl?)`

すべてのメールを取得します。

```typescript
const messages = await getMessages();
console.log(`Total messages: ${messages.length}`);
```

### `clearAllMessages(mailpitUrl?)`

すべてのメールを削除します（テスト前のクリーンアップ用）。

```typescript
await clearAllMessages();
```

### `extractLoginUrl(messageText)`

メール本文からログインURLを抽出します。

```typescript
const loginUrl = extractLoginUrl(detail.Text);
// または
const loginUrl = extractLoginUrl(detail.HTML);
```

## トラブルシューティング

### メールが届かない

1. Mailpit Web UI (http://localhost:8025) で受信状況を確認
2. プロキシのログを確認: `docker logs e2e-proxy-app`
3. Mailpitのログを確認: `docker logs e2e-mailpit`
4. 設定ファイルで `otp_output_file` がコメントアウトされているか確認

### URLが抽出できない

1. Mailpit Web UIでメール内容を確認
2. URL形式が正しいか確認（`/_auth/email/verify?token=...`）
3. `extractLoginUrl()` のパターンマッチングを確認

### タイムアウトエラー

```typescript
// タイムアウトを延長
const loginUrl = await waitForLoginEmail('test@example.com', {
  timeoutMs: 30_000,  // 30秒に延長
});
```

## 従来のファイルベース方式との併用

`otp_output_file` を設定した場合は、従来のファイルベース方式も使用できます：

```typescript
import { waitForOtp, clearOtpFile } from '../support/otp-reader';

test('file-based OTP', async ({ page }) => {
  await clearOtpFile();

  // メール送信
  await page.getByLabel('Email Address').fill('test@example.com');
  await page.getByRole('button', { name: 'Send Login Link' }).click();

  // ファイルからOTPを読み取り
  const otp = await waitForOtp('test@example.com');

  // ログインURLにアクセス
  await page.goto(otp.login_url);
});
```

**注意**: Mailpitとファイル出力は排他的です。設定ファイルで `otp_output_file` を設定すると、
メールは送信されず、ファイルにのみ出力されます。

## 環境変数

Playwrightテストで以下の環境変数を設定できます：

```bash
# Mailpit URL（デフォルト: http://mailpit:8025）
export MAILPIT_URL=http://localhost:8025

# OTPファイルパス（ファイルベース方式を使う場合）
export OTP_FILE=/path/to/otp.jsonl
```

## 参考資料

- [Mailpit GitHub](https://github.com/axllent/mailpit)
- [Mailpit API ドキュメント](https://github.com/axllent/mailpit/blob/develop/docs/apiv1/README.md)
- [E2E テスト計画](./E2E.md)
