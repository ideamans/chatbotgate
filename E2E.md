# E2Eテスト計画

## 目的
- multi-oauth2-proxy が OAuth2 認証およびメールパスワードレス認証の双方で、想定するフローをエンドツーエンドで満たすことを確認する。
- ローカル環境で開発者や Playwright が追加設定なしに動作確認できるコンテナベースの検証環境を用意する。

## 検証対象の要求事項
- 任意の OAuth2 プロバイダーを設定できる構成が存在し、テスト用 OAuth2 サーバーと連携できる。
- someone@example.com / password でテストサーバーにログインし、OAuth2 経由で保護されたページへ遷移できる。
- サインアウト後に保護ページへアクセスすると再度認証画面へリダイレクトされる。
- メールパスワードレス認証で発行されたワンタイムパスワード(OTP)がファイルに保存され、Playwright が読み取ってログインできる。
- 目的のページに OAuth2 サインアウトボタンが表示され、操作するとセッションが破棄される。

## テスト環境全体像
- **multi-oauth2-proxy 本体サーバー**: 認証プロキシ。テスト設定ファイルを読み込み、テストサーバーを OAuth2 プロバイダーとして扱う。メール認証の OTP を e2e ディレクトリ配下の一時ファイルへ出力する。
- **テストサーバー (stub-auth)**: TypeScript/Express で実装する簡易アプリ。
  - 目的ページ (`/app`) とログインフォーム (`/login`) を提供。
  - OAuth2 プロバイダーとして `/oauth/authorize`・`/oauth/token`・`/oauth/userinfo` を実装。
  - `someone@example.com` / `password` のみ認証成功。
  - `/logout` でセッション破棄し、目的ページに OAuth2 サインアウトボタンを表示。
- **Playwright ランナー**: TypeScript で E2E テストを記述し、コンテナ上でブラウザ操作を自動化。
- **共有ボリューム**: OTP ファイルなどを `e2e/tmp` に書き出し、本体・Playwright 双方が参照。

## 想定ディレクトリ構成
```
e2e/
  config/
    proxy.e2e.yaml        # multi-oauth2-proxy 用のテスト設定
    stub-auth.env         # テストサーバー用環境変数
  docker/
    docker-compose.e2e.yaml            # 手動確認用 (proxy + stub)
    docker-compose.e2e.playwright.yaml # Playwright 実行サービスを追加
    Dockerfile.stub-auth               # テストサーバー用
    Dockerfile.playwright              # Playwright ランナー用
  playwright.config.ts
  src/
    stub-auth/                         # TypeScript サーバー実装
    tests/
      oauth2.spec.ts
      passwordless.spec.ts
    support/
      otp-reader.ts                    # OTP ファイル読取ヘルパー
  tmp/
    passwordless-otp.json              # OTP 保存先 (実行時に生成)
```

## Docker Compose 設計
- `docker/docker-compose.e2e.yaml`
  - ネットワーク: `e2e-net`
  - サービス `proxy-app`: multi-oauth2-proxy をビルドまたはローカルバイナリで起動。`config/proxy.e2e.yaml` をマウントし、`/otp` ボリュームとして `e2e/tmp` をマウント。
  - サービス `stub-auth`: `Dockerfile.stub-auth` からビルド。ポート 3000 を公開。セッションはサーバー内メモリ、サインアウト時に Cookie を削除。
  - 共有ボリューム: `otp-files` → `../tmp` へバインド。
- `docker/docker-compose.e2e.playwright.yaml`
  - 上記ファイルを `extends` しつつ Playwright サービス(`playwright-runner`)を追加。
  - `playwright-runner` は `Dockerfile.playwright` で構築し、`npm ci && npx playwright test` をエントリーポイントに設定。
  - Playwright サービスには `DISPLAY` 等は不要。`proxy-app` の 4180 ポートへ内部アクセスし、OTP ファイルを `../tmp/passwordless-otp.json` から読み取る。
  - テスト結果は `e2e/test-results` に出力 (バインドマウント)。

## テストサーバー (stub-auth) の詳細
- **使用技術**: Node.js + TypeScript + Express + passport-like ライブラリなしの簡易実装。
- **エンドポイント**
  - `GET /app` : 認証済みであれば保護ページを表示。OAuth2 サインアウトボタンを設置。
  - `GET /login` : ログインフォーム。Playwright がメール/パスワードを入力できるように `data-test` 属性を付与。
  - `POST /login` : フォーム送信。成功時はセッションを生成し `/app` へリダイレクト。失敗時はエラー表示。
  - `POST /logout` : セッション破棄後 `/login` にリダイレクト。
  - `GET /oauth/authorize` : Authorization Code フローの認可画面。ログイン済みであれば確認画面を表示し、承認で `code` を付与。
  - `POST /oauth/token` : 認可コードを検証し、`access_token`・`id_token` 相当の JSON を返す。
  - `GET /oauth/userinfo` : `access_token` からメールアドレスを返却。
  - `GET /.well-known/openid-configuration` : OIDC メタデータを固定値で返す。
- **セッション管理**: `express-session` 相当の仕組みで Cookie `stub_session` を発行。Docker 再起動毎にリセット可。
- **サインアウトボタン**: `/app` 内に `<form action="/logout" method="post">` を配置し、Playwright がクリック可能な `data-test="oauth-signout"` を設定。

## 本体サーバーへの追加要件
- **カスタム OAuth2 プロバイダー定義**
  - 設定ファイルで `provider.type: custom` を指定し、`auth_url`・`token_url`・`userinfo_url`・`jwks_url` を外部から与えられるようにする。
  - client_id/client_secret は `docker compose` の `.env` から注入。
  - テストサーバーの self-signed 証明書は用意しない前提で HTTP を許容する (環境変数で制御)。
- **パスワードレス OTP のファイル出力**
  - メール送信の代わりに OTP 情報 (メールアドレス・トークン・有効期限) を JSON で `e2e/tmp/passwordless-otp.json` へ追記。
  - 権限競合を避けるためファイルロック or append-safe な書き方を検討。
  - OTP 取り出し用 CLI や HTTP は不要。Playwright がファイルを直接読む。
- **ログ/デバッグ**
  - テスト時には認証の成否を標準出力に INFO ログで記録し、Playwright が失敗原因を追跡しやすくする。

## テスト設定ファイル (`e2e/config/proxy.e2e.yaml`) の想定項目
- `upstream` : `http://stub-auth:3000/app`
- `oauth2.providers[0]` : custom プロバイダー設定 (authorize/token/userinfo/oidc discovery)。
- `oauth2.redirect_url` : `http://proxy-app:4180/oauth2/callback`。
- `oauth2.allowed_emails` : `["someone@example.com"]`。
- `passwordless` : 有効化フラグ、OTP ファイル出力先、トークン有効時間短縮 (例: 3 分)。
- `session` : Cookie 名や暗号キー (テスト用固定値)。
- `log` : DEBUG レベル。

## Playwright テスト方針
- `playwright.config.ts`
  - ベース URL: `http://proxy-app:4180`。
  - WebKit/Firefox までは不要、Chromium のみ。
  - 失敗時のスクリーンショット・トレースを `e2e/test-results` へ保存。
- `oauth2.spec.ts`
  1. トップページから OAuth2 ログインを選択。
  2. stub-auth のログインフォームに `someone@example.com` / `password` を入力。
  3. 同意画面で承認し、目的ページ `/app` に到達することを確認。
  4. OAuth サインアウトボタンをクリックし、proxy 経由でセッションが終了。
  5. 直後に `/app` を再訪し、ログイン画面に戻されることを検証。
- `passwordless.spec.ts`
  1. メールログインを選択し、`someone@example.com` を送信。
  2. `tmp/passwordless-otp.json` をポーリングして最新 OTP を取得 (サポートスクリプトで実装)。
  3. OTP 入力画面にコードを投入し、目的ページに遷移できることを検証。
  4. OTP を再使用できないことを確認 (再入力でエラーになることを期待)。

## 実行手順 (想定)
1. `make e2e-build` などのコマンドで multi-oauth2-proxy バイナリを用意し、Docker から参照できるようにする。
2. 手動確認: `docker compose -f e2e/docker/docker-compose.e2e.yaml up --build` を起動し、ブラウザで `http://localhost:4180/` にアクセスしてフローを確認。
3. 自動確認: `docker compose -f e2e/docker/docker-compose.e2e.yaml -f e2e/docker/docker-compose.e2e.playwright.yaml up --build playwright-runner` を実行。Playwright が完了後に終了コードで合否を判定。
4. 実行後、`e2e/test-results` と `e2e/tmp/passwordless-otp.json` を確認し、不要であればクリーンアップ。

## リスクと検討事項
- ローカルで使用するポート (4180, 3000) が競合しないか確認が必要。
- OTP ファイルのローテーションをどう行うか (テストごとにクリアする仕組みが必要)。
- Docker ビルド時間短縮のため、Playwright イメージは公式 base (`mcr.microsoft.com/playwright`) を利用する案も検討。
- Windows でもパスが正しくマウントされるか事前検証が必要。

## 次のアクション
- テストサーバーと Playwright 環境の雛形を `e2e/` に追加。
- multi-oauth2-proxy 側の設定対応と OTP ファイル出力機能を実装。
- Docker Compose で相互接続を確認し、テストを走らせて調整。
