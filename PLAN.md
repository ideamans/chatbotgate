# multi-oauth2-proxy 設計書

## 概要

複数のOAuth2プロバイダーとメール認証を統合した認証プロキシサーバー。
[oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/)の実装を参考にしつつ、
Firebase Authenticationのような柔軟な認証機能をセルフホストで実現する独自実装。

## プロジェクト目標

- 複数のOAuth2プロバイダー（Google, GitHub, Microsoft等）を統一的に扱う（Phase 1: Google, Phase 3: 複数対応）
- メールリンクによるパスワードレス認証を提供（Phase 2）
- メールアドレスベースのアクセス制御（個別アドレス・ドメイン単位）
- Hostヘッダーベースのマルチテナント対応ルーティング（Phase 3）
- サービス名・説明のカスタマイズ（GUIとメールに反映）
- 日英対応（環境変数から自動検出、Phase 3）
- YAML設定ファイルによる柔軟な設定管理
- 設定変更の自動検知・即座の反映（Phase 3）
- インターフェース設計によるテスト容易性の確保
- モックを活用した高品質な単体テスト（段階的に60% → 70% → 80%+）
- 将来的なSSL/TLS自動化（Let's Encrypt + lego）への対応（Phase 5）
- モジュール化されたアーキテクチャで拡張性を確保

## 機能要件

### 1. 認証機能

#### 1.1 OAuth2認証
- 複数プロバイダーの同時サポート
  - Google
  - GitHub
  - Microsoft Azure AD
  - その他OIDC互換プロバイダー
- プロバイダーごとの個別設定
- 認証後のメールアドレスによる認可チェック

#### 1.2 メール認証
- メールアドレス入力フォーム
- ワンタイムパスワード（マジックリンク）の生成
- メール送信
  - SMTP経由（TLS/STARTTLS対応）
  - SendGrid API v3経由
  - メールテンプレートは固定（設定不要）
  - 日英自動切替（LANGUAGE/LANG環境変数から判定）
  - サービス名を含む本文
- トークンの有効期限管理（デフォルト15分）
- 使い捨てトークンの実装（1回使用で無効化）

#### 1.3 認可（Authorization）
- メールアドレスベースのホワイトリスト
  - 個別アドレス指定（例：`miyanaga@gmail.com`）
  - ドメイン指定（例：`@ideamans.com`）
- 設定ファイルでの柔軟な許可リスト管理

### 2. プロキシ機能

- リバースプロキシとしての動作
- メインアップストリーム（必須）
  - デフォルトのプロキシ先バックエンド
- Hostヘッダーベースのルーティング（オプション）
  - リクエストのHostヘッダーに応じて異なるバックエンドへルーティング
  - マルチテナント対応
- 認証済みリクエストのバックエンドへの転送
- 認証情報のヘッダーへの付与
  - `X-Forwarded-User`: ユーザーのメールアドレス
  - `X-Forwarded-Email`: メールアドレス
  - `X-Auth-Provider`: 認証プロバイダー名

### 3. セッション管理

- Cookieベースのセッション管理
- セッションストレージ
  - インメモリ（デフォルト）
  - Redis（オプション、将来的な拡張）
- セッション有効期限の設定可能化

### 4. 設定管理

- YAML設定ファイル
- ファイル監視による自動リロード
- 設定変更時の即座の反映（サーバー再起動不要）
- 設定バリデーション

### 5. UI

- water.cssによるシンプルなデザイン
- 日英対応（LANGUAGE/LANG環境変数から自動検出）
- サービス名・説明を設定から取得して表示
- 提供するページ
  - ログイン選択画面（OAuth2プロバイダー + メール認証）
  - メールアドレス入力フォーム
  - 認証完了画面
  - エラー画面
  - ログアウト画面

## 技術スタック

### プログラミング言語
- **Go 1.21+**
  - 高性能なHTTPサーバー
  - 豊富な標準ライブラリ
  - 強力な並行処理機能
  - 優れたテスト機能とインターフェース設計

### 主要ライブラリ

#### 認証・プロキシ
- `golang.org/x/oauth2` - OAuth2クライアント実装
- `github.com/coreos/go-oidc/v3` - OIDC実装
- 注: oauth2-proxyの実装を参考にするが、独自実装として開発

#### Web・HTTP
- `github.com/go-chi/chi/v5` - HTTPルーター（軽量でメンテナンス継続中）
- 標準ライブラリ `net/http`, `net/http/httputil`
- セッション管理は独自実装（pkg/session）

#### 設定・ファイル監視
- `gopkg.in/yaml.v3` - YAML設定ファイルのパース
- `github.com/fsnotify/fsnotify` - ファイル変更の監視

#### メール送信
- `github.com/wneessen/go-mail` - SMTP メール送信
- `github.com/sendgrid/sendgrid-go` - SendGrid API v3

#### セキュリティ・暗号
- 標準ライブラリ `crypto/rand`, `crypto/hmac`
- `golang.org/x/crypto/bcrypt` - トークンハッシュ化

#### ログ・国際化
- `github.com/fatih/color` - カラー出力（TTY検出付き）
- `github.com/ideamans/go-l10n` - 国際化（i18n）対応
- 標準ライブラリ `log` - ベースロガー

### フロントエンド
- **water.css** - クラスレスCSSフレームワーク
- テンプレートエンジン: `html/template`（Go標準）

## アーキテクチャ設計

### 全体構成

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTPS (将来)
       │ HTTP
┌──────▼───────────────────────────────┐
│   multi-oauth2-proxy                 │
│                                      │
│  ┌────────────────────────────────┐ │
│  │  HTTP Server (Chi Router)     │ │
│  └───────────┬────────────────────┘ │
│              │                       │
│  ┌───────────▼────────────────────┐ │
│  │  Authentication Middleware    │ │
│  └───────────┬────────────────────┘ │
│              │                       │
│     ┌────────┴────────┐             │
│     │                 │             │
│  ┌──▼──────┐   ┌──────▼─────┐      │
│  │ OAuth2  │   │   Email    │      │
│  │ Handler │   │   Handler  │      │
│  └──┬──────┘   └──────┬─────┘      │
│     │                 │             │
│  ┌──▼─────────────────▼─────┐      │
│  │  Authorization Check     │      │
│  │  (Email Whitelist)       │      │
│  └──┬───────────────────────┘      │
│     │                               │
│  ┌──▼───────────────────────┐      │
│  │  Reverse Proxy           │      │
│  └──────────────────────────┘      │
│                                      │
│  ┌────────────────────────────────┐ │
│  │  Config Manager (YAML Watch)  │ │
│  └────────────────────────────────┘ │
│                                      │
│  ┌────────────────────────────────┐ │
│  │  Session Store                 │ │
│  └────────────────────────────────┘ │
└──────────┬───────────────────────────┘
           │
   ┌───────▼────────┐
   │  Backend App   │
   └────────────────┘
```

### モジュール構成

```
multi-oauth2-proxy/
├── cmd/
│   └── multi-oauth2-proxy/     # メインエントリポイント
│       └── main.go
│
├── pkg/
│   ├── config/                 # 設定管理
│   │   ├── config.go           # 設定構造体
│   │   ├── loader.go           # YAML読み込み
│   │   ├── watcher.go          # ファイル監視
│   │   └── validator.go        # 設定バリデーション
│   │
│   ├── auth/                   # 認証モジュール
│   │   ├── manager.go          # 認証マネージャー
│   │   ├── oauth2/             # OAuth2認証
│   │   │   ├── provider.go    # プロバイダー抽象化
│   │   │   ├── google.go      # Google実装
│   │   │   ├── github.go      # GitHub実装
│   │   │   └── oidc.go        # 汎用OIDC実装
│   │   │
│   │   └── email/              # メール認証
│   │       ├── handler.go     # メール認証ハンドラー
│   │       ├── token.go       # トークン生成・検証
│   │       └── sender.go      # メール送信（SMTP/SendGrid）
│   │
│   ├── authz/                  # 認可モジュール
│   │   ├── checker.go          # メールベース認可
│   │   └── rules.go            # 認可ルール
│   │
│   ├── session/                # セッション管理
│   │   ├── store.go            # セッションストア抽象化
│   │   ├── memory.go           # インメモリ実装
│   │   └── redis.go            # Redis実装（将来）
│   │
│   ├── proxy/                  # プロキシ機能
│   │   ├── handler.go          # リバースプロキシハンドラー
│   │   ├── router.go           # Hostベースルーティング
│   │   └── headers.go          # ヘッダー付与
│   │
│   ├── server/                 # HTTPサーバー
│   │   ├── server.go           # サーバー起動
│   │   ├── middleware.go       # 認証ミドルウェア
│   │   └── routes.go           # ルート定義
│   │
│   ├── ui/                     # UI関連
│   │   ├── templates/          # HTMLテンプレート
│   │   │   ├── login.html
│   │   │   ├── email.html
│   │   │   ├── success.html
│   │   │   └── error.html
│   │   └── handler.go          # UIハンドラー
│   │
│   ├── logging/                # ロギング
│   │   ├── logger.go           # ロガーインターフェース
│   │   ├── color.go            # カラー出力実装
│   │   └── translations.go     # 翻訳定義（go-l10n）
│   │
│   ├── testutil/               # テストユーティリティ
│   │   ├── helpers.go          # テストヘルパー関数
│   │   ├── mocks.go            # 共通モック
│   │   └── config.go           # テスト用設定
│   │
│   └── tls/                    # TLS管理（将来実装）
│       ├── manager.go          # 証明書管理
│       └── lego.go             # lego統合
│
├── web/
│   └── static/
│       └── water.css           # CSSファイル
│
├── config.yaml                 # デフォルト設定ファイル
├── config.example.yaml         # 設定例
├── .github/
│   └── workflows/
│       └── test.yml            # CI/CDテストワークフロー
├── go.mod
├── go.sum
├── LICENSE                     # MITライセンス
├── README.md
└── SPEC.md                     # 本ドキュメント
```

**注記:**
- 各パッケージには対応する `*_test.go` ファイルが含まれます
- テストファイルは実装ファイルと同じディレクトリに配置
- モックは各パッケージ内で定義（例：`pkg/auth/email/mock.go`）
- 共通のテストユーティリティは `pkg/testutil/` に配置

## 設定ファイル仕様

### config.yaml

```yaml
# サービス設定
service:
  name: "Multi OAuth2 Proxy"  # サービス名（GUIとメールに表示）
  description: "統合認証サービス"  # サービス説明（GUIに表示）

# サーバー設定
server:
  host: "0.0.0.0"
  port: 4180
  # 将来的なTLS設定
  # tls:
  #   enabled: true
  #   auto: true  # Let's Encrypt自動取得
  #   email: "admin@example.com"

# プロキシ設定
proxy:
  # メインアップストリーム（必須）
  upstream: "http://localhost:8080"

  # Hostヘッダーベースのルーティング（オプション）
  # リクエストのHostヘッダーに一致するルートがあればそちらを使用
  # 一致しない場合はメインアップストリームにフォールバック
  routes:
    - host: "app1.example.com"
      upstream: "http://localhost:8081"

    - host: "app2.example.com"
      upstream: "http://localhost:8082"

    - host: "api.example.com"
      upstream: "http://localhost:8083"

# セッション設定
session:
  cookie_name: "_oauth2_proxy"
  cookie_secret: "ランダム文字列（32文字以上推奨）"
  cookie_expire: "168h"  # 7日間
  cookie_secure: false   # HTTPSの場合true
  cookie_httponly: true
  cookie_samesite: "lax"

# OAuth2プロバイダー設定
oauth2:
  providers:
    - name: "google"
      display_name: "Google"
      client_id: "your-google-client-id"
      client_secret: "your-google-client-secret"
      enabled: true

    - name: "github"
      display_name: "GitHub"
      client_id: "your-github-client-id"
      client_secret: "your-github-client-secret"
      enabled: true

    - name: "microsoft"
      display_name: "Microsoft"
      client_id: "your-microsoft-client-id"
      client_secret: "your-microsoft-client-secret"
      tenant: "common"  # Azure AD Tenant
      enabled: false

# メール認証設定
email_auth:
  enabled: true

  # メール送信方法: "smtp" または "sendgrid"
  sender_type: "smtp"

  # SMTP設定（sender_type: "smtp" の場合）
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from: "noreply@example.com"
    from_name: "Multi OAuth2 Proxy"  # 送信者名（オプション）
    # TLS設定
    tls: true
    # STARTTLS設定
    starttls: true

  # SendGrid API設定（sender_type: "sendgrid" の場合）
  sendgrid:
    api_key: "SG.xxxxxxxxxxxxxxxxxxxx"  # SendGrid APIキー
    from: "noreply@example.com"
    from_name: "Multi OAuth2 Proxy"  # 送信者名（オプション）

  token:
    expire: "15m"  # トークン有効期限

  # メールテンプレートは固定（日英自動切替）
  # GUIの言語設定に応じて自動的に日本語または英語のメールを送信

# 認可設定（メールホワイトリスト）
authorization:
  # 個別メールアドレス
  allowed_emails:
    - "miyanaga@gmail.com"
    - "user@example.com"

  # ドメイン単位の許可
  allowed_domains:
    - "@ideamans.com"
    - "@example.org"

# ログ設定
logging:
  # デフォルトログレベル（メインモジュール用）
  level: "info"  # debug, info, warn, error

  # サブモジュールのデフォルトレベル
  module_level: "debug"

  # カラー出力（TTY自動検出、パイプ時は無効化）
  color: true

  # 言語はLANGUAGE/LANG環境変数から自動検出
  # go-l10nが自動判別（ja: 日本語, en: 英語）

  # 構造化データの出力（将来実装）
  # structured: false
```

## サービス設定仕様

### サービス名と説明

設定ファイルの `service.name` と `service.description` は、以下の場所で使用されます：

**GUIでの表示:**
- ログイン画面のヘッダー
- メールアドレス入力フォームのヘッダー
- 認証完了画面
- エラー画面

**メールテンプレート:**
- メール件名: 「ログインリンク - {{service.name}}」
- メール本文: サービス名を含む説明文

**例:**
```yaml
service:
  name: "My Application"
  description: "マイアプリケーションの統合認証"
```

上記の設定で、ログイン画面には「My Application」が表示され、
メールの件名は「ログインリンク - My Application」（日本語環境）または
「Login Link - My Application」（英語環境）となります。

### 多言語対応の動作

**言語の自動検出:**
- `LANGUAGE` または `LANG` 環境変数から言語を検出
- `ja` で始まる場合: 日本語
- その他: 英語（デフォルト）

**例:**
```bash
# 日本語でサーバーを起動
LANGUAGE=ja ./multi-oauth2-proxy

# 英語でサーバーを起動
LANGUAGE=en ./multi-oauth2-proxy
```

**GUIとメールの連動:**
- サーバーの言語設定に応じて、GUIとメールの言語が統一される
- ユーザーは環境変数で設定された言語でUIを表示し、同じ言語でメールを受け取る

## プロキシルーティング仕様

### ルーティングロジック

認証後のリクエストは以下のロジックでバックエンドにルーティングされます：

1. **Hostヘッダーの確認**
   - リクエストの `Host` ヘッダーを取得
   - 設定された `proxy.routes` から一致するルートを検索

2. **ルート選択**
   - 一致するルートが見つかった場合：そのルートの `upstream` を使用
   - 一致するルートがない場合：メイン `upstream` にフォールバック

3. **プロキシ実行**
   - 選択されたアップストリームにリクエストを転送
   - 認証情報ヘッダーを付与

### ルーティング例

#### 設定
```yaml
proxy:
  upstream: "http://localhost:8080"  # デフォルト
  routes:
    - host: "app1.example.com"
      upstream: "http://localhost:8081"
    - host: "app2.example.com"
      upstream: "http://localhost:8082"
```

#### リクエスト例

| リクエストHost | ルーティング先 | 説明 |
|---------------|---------------|------|
| `app1.example.com` | `http://localhost:8081` | ルート一致 |
| `app2.example.com` | `http://localhost:8082` | ルート一致 |
| `unknown.example.com` | `http://localhost:8080` | フォールバック |
| `localhost:4180` | `http://localhost:8080` | フォールバック |

### ユースケース

#### マルチテナントSaaS
```yaml
proxy:
  upstream: "http://default-app:8080"
  routes:
    - host: "tenant1.saas.com"
      upstream: "http://tenant1-backend:8080"
    - host: "tenant2.saas.com"
      upstream: "http://tenant2-backend:8080"
```

#### 複数サービスの統合認証
```yaml
proxy:
  upstream: "http://main-app:8080"
  routes:
    - host: "api.example.com"
      upstream: "http://api-server:8080"
    - host: "admin.example.com"
      upstream: "http://admin-panel:8080"
    - host: "docs.example.com"
      upstream: "http://documentation:8080"
```

## ロギング仕様

### ログレベル

- **DEBUG**: デバッグ情報（サブモジュール用、開発時のみ）
- **INFO**: 通常の情報ログ（メインモジュールのデフォルト）
- **WARN**: 警告（処理は継続するが注意が必要）
- **ERROR**: エラー（処理に失敗したが継続可能）
- **FATAL**: 致命的エラー（プログラム終了）

### ログフォーマット

#### 基本フォーマット
```
時刻 [モジュール] レベル: メッセージ
```

#### 例
```
2025-11-01 15:04:05 [server] INFO: サーバーを起動しました port=4180
2025-11-01 15:04:06 [oauth2] DEBUG: Google認証を初期化しました
2025-11-01 15:04:10 [email] INFO: ログインリンクを送信しました to=user@example.com
2025-11-01 15:04:15 [authz] WARN: 認可に失敗しました email=unknown@example.com
2025-11-01 15:04:20 [proxy] ERROR: バックエンド接続エラー upstream=http://localhost:8080
```

### カラー出力

TTY（端末）に出力する場合のみカラー表示を有効化。パイプやリダイレクト時は自動的に無効化。

**カラーマッピング:**
- モジュール名（`[xxx]`）: **シアン（太字）**
- DEBUG: グレー
- INFO: 緑
- WARN: 黄色
- ERROR: 赤
- FATAL: 赤（太字）

### モジュール名

各パッケージで以下のモジュール名を使用：

| パッケージ | モジュール名 |
|-----------|-------------|
| `cmd/multi-oauth2-proxy` | `main` |
| `pkg/server` | `server` |
| `pkg/auth/oauth2` | `oauth2` |
| `pkg/auth/email` | `email` |
| `pkg/authz` | `authz` |
| `pkg/session` | `session` |
| `pkg/proxy` | `proxy` |
| `pkg/config` | `config` |
| `pkg/tls` | `tls` |

### ログレベル設定

**メインモジュール（オーケストレーション）:**
- `main`, `server`: **INFO** 以上

**サブモジュール:**
- その他すべて: **DEBUG** 以上（開発時）、**INFO** 以上（本番時）

### 国際化（i18n）

`github.com/ideamans/go-l10n` を使用して日英対応。

**特徴:**
- 翻訳ファイルは不要
- ソースコードに直接翻訳を記述
- LANGUAGE/LANG環境変数から自動で言語を検出
- テンプレート（Go template）による動的な値の埋め込み

**ログメッセージの翻訳例:**
```go
import "github.com/ideamans/go-l10n"

// 翻訳をソースコードに定義
var translations = l10n.Translations{
    "server.started": {
        "ja": "サーバーを起動しました port={{.port}}",
        "en": "Server started port={{.port}}",
    },
    "oauth2.initialized": {
        "ja": "{{.provider}}認証を初期化しました",
        "en": "{{.provider}} authentication initialized",
    },
    "email.link_sent": {
        "ja": "ログインリンクを送信しました to={{.to}}",
        "en": "Login link sent to={{.to}}",
    },
    "authz.denied": {
        "ja": "認可に失敗しました email={{.email}}",
        "en": "Authorization denied email={{.email}}",
    },
}

// 使用例
translator := l10n.New(translations)
message := translator.T("server.started", map[string]interface{}{
    "port": 4180,
})
logger.Info(message)
```

**メールテンプレートの翻訳例:**
```go
var emailTemplates = l10n.Translations{
    "email.subject": {
        "ja": "ログインリンク - {{.serviceName}}",
        "en": "Login Link - {{.serviceName}}",
    },
    "email.body": {
        "ja": `以下のリンクをクリックしてログインしてください。
このリンクは{{.expire}}分間有効です。

{{.loginURL}}

サービス: {{.serviceName}}

このメールに心当たりがない場合は、無視してください。`,
        "en": `Click the link below to log in.
This link is valid for {{.expire}} minutes.

{{.loginURL}}

Service: {{.serviceName}}

If you did not request this email, please ignore it.`,
    },
}
```

### ロガーインターフェース

```go
package logging

type Logger interface {
    // 基本ログメソッド
    Debug(key string, args ...interface{})
    Info(key string, args ...interface{})
    Warn(key string, args ...interface{})
    Error(key string, args ...interface{})
    Fatal(key string, args ...interface{})

    // モジュール名を指定したロガーを作成
    WithModule(module string) Logger
}
```

### 使用例

```go
// メインモジュール
logger := logging.New(config.Logging)
logger.Info("server.started", "port", config.Port)

// サブモジュール
emailLogger := logger.WithModule("email")
emailLogger.Debug("token.generated", "expires", "15m")
emailLogger.Info("email.link_sent", "to", email)
```

### 構造化データの出力（将来実装）

現在は構造化データの出力は保留。将来的に JSON 形式での出力をサポート予定。

```json
{
  "timestamp": "2025-11-01T15:04:05+09:00",
  "level": "INFO",
  "module": "server",
  "message": "サーバーを起動しました",
  "port": 4180
}
```

## 認証フロー

### OAuth2認証フロー

```
1. ユーザーがプロテクトされたURLにアクセス
   ↓
2. 認証チェック（Cookie確認）
   ↓ 未認証
3. /login にリダイレクト
   ↓
4. ユーザーがプロバイダーを選択（例：Google）
   ↓
5. /oauth2/start/{provider} にリクエスト
   ↓
6. OAuth2プロバイダーにリダイレクト
   ↓
7. ユーザーが承認
   ↓
8. /oauth2/callback にリダイレクト
   ↓
9. 認証コードをトークンに交換
   ↓
10. メールアドレスを取得
    ↓
11. 認可チェック（ホワイトリスト確認）
    ↓ OK
12. セッションCookieを設定
    ↓
13. 元のURLにリダイレクト
    ↓
14. バックエンドにプロキシ
```

### メール認証フロー

```
1. ユーザーがプロテクトされたURLにアクセス
   ↓
2. 認証チェック（Cookie確認）
   ↓ 未認証
3. /login にリダイレクト
   ↓
4. ユーザーが「メールでログイン」を選択
   ↓
5. メールアドレス入力フォーム表示
   ↓
6. POST /auth/email/send
   ↓
7. メールアドレスの認可チェック（ホワイトリスト）
   ↓ OK
8. ワンタイムトークン生成（HMAC-SHA256）
   ↓
9. トークンをインメモリストアに保存（有効期限付き）
   ↓
10. マジックリンクをメール送信
    - GUIの言語設定（LANGUAGE/LANG環境変数）に応じて日英自動切替
    - サービス名を含むメール本文
    ↓
11. ユーザーがメール内のリンクをクリック
    ↓
12. GET /auth/email/verify?token=xxx
    ↓
13. トークン検証（有効期限・使用済みチェック）
    ↓ OK
14. トークンを無効化（使い捨て）
    ↓
15. セッションCookieを設定
    ↓
16. 元のURLまたはホームにリダイレクト
```

## API エンドポイント

### 認証関連

| エンドポイント | メソッド | 説明 |
|----------------|----------|------|
| `/login` | GET | ログイン選択画面 |
| `/logout` | GET/POST | ログアウト |
| `/oauth2/start/{provider}` | GET | OAuth2認証開始 |
| `/oauth2/callback` | GET | OAuth2コールバック |
| `/auth/email` | GET | メールアドレス入力フォーム |
| `/auth/email/send` | POST | マジックリンク送信 |
| `/auth/email/verify` | GET | トークン検証・ログイン |

### 管理・ヘルスチェック

| エンドポイント | メソッド | 説明 |
|----------------|----------|------|
| `/health` | GET | ヘルスチェック |
| `/ready` | GET | Readinessチェック |
| `/api/config/reload` | POST | 設定再読み込み（手動）|

### プロキシ

| エンドポイント | メソッド | 説明 |
|----------------|----------|------|
| `/*` | ALL | バックエンドへのプロキシ（認証済み）|

## セキュリティ考慮事項

### 1. トークン管理
- ワンタイムトークンはHMAC-SHA256で生成
- トークンは1回使用で即座に無効化
- 有効期限はデフォルト15分（設定可能）
- トークンはセキュアランダム値を含む

### 2. セッション管理
- Cookie は HttpOnly フラグ必須
- HTTPS使用時は Secure フラグ必須
- SameSite=Lax でCSRF対策
- セッション有効期限の適切な設定

### 3. 設定ファイル
- 機密情報（クライアントシークレット、パスワード等）は環境変数でも設定可能
- 設定ファイルのパーミッション管理（600推奨）

### 4. OAuth2
- State パラメータでCSRF対策
- Nonce パラメータでリプレイ攻撃対策（OIDC）
- PKCE（Proof Key for Code Exchange）の使用を推奨

### 5. レート制限
- メール送信のレート制限（将来実装）
- ログイン試行のレート制限（将来実装）

## テスト設計

### インターフェース設計の原則

すべての外部依存とモジュールをインターフェース化し、モックによる単体テストを容易にする。

### 主要なインターフェース

#### 1. メール送信インターフェース

```go
package email

type Sender interface {
    Send(to, subject, body string) error
}

// SMTP実装
type SMTPSender struct {
    config SMTPConfig
}

func (s *SMTPSender) Send(to, subject, body string) error {
    // SMTP送信の実装
}

// SendGrid実装
type SendGridSender struct {
    apiKey string
}

func (s *SendGridSender) Send(to, subject, body string) error {
    // SendGrid API呼び出し
}

// モック（テスト用）
type MockSender struct {
    SendFunc func(to, subject, body string) error
    Calls    []SendCall
}

type SendCall struct {
    To      string
    Subject string
    Body    string
}

func (m *MockSender) Send(to, subject, body string) error {
    m.Calls = append(m.Calls, SendCall{to, subject, body})
    if m.SendFunc != nil {
        return m.SendFunc(to, subject, body)
    }
    return nil
}
```

#### 2. OAuth2プロバイダーインターフェース

```go
package oauth2

type Provider interface {
    Name() string
    AuthURL(state string) string
    Exchange(code string) (*Token, error)
    GetUserEmail(token *Token) (string, error)
}

// モック（テスト用）
type MockProvider struct {
    NameFunc         func() string
    AuthURLFunc      func(state string) string
    ExchangeFunc     func(code string) (*Token, error)
    GetUserEmailFunc func(token *Token) (string, error)
}
```

#### 3. セッションストアインターフェース

```go
package session

type Store interface {
    Get(id string) (*Session, error)
    Set(id string, session *Session) error
    Delete(id string) error
}

// インメモリ実装
type MemoryStore struct {
    sessions map[string]*Session
    mu       sync.RWMutex
}

// モック（テスト用）
type MockStore struct {
    GetFunc    func(id string) (*Session, error)
    SetFunc    func(id string, session *Session) error
    DeleteFunc func(id string) error
}
```

#### 4. 設定ローダーインターフェース

```go
package config

type Loader interface {
    Load() (*Config, error)
    Watch(callback func(*Config)) error
}

// YAML実装
type YAMLLoader struct {
    path string
}

// モック（テスト用）
type MockLoader struct {
    LoadFunc  func() (*Config, error)
    WatchFunc func(callback func(*Config)) error
}
```

#### 5. HTTPクライアントインターフェース

```go
package http

type Client interface {
    Do(req *http.Request) (*http.Response, error)
}

// 標準ライブラリのhttp.Clientをラップ
type DefaultClient struct {
    client *http.Client
}

// モック（テスト用）
type MockClient struct {
    DoFunc func(req *http.Request) (*http.Response, error)
    Calls  []*http.Request
}
```

#### 6. 認可チェッカーインターフェース

```go
package authz

type Checker interface {
    IsAllowed(email string) bool
}

// 実装
type EmailChecker struct {
    allowedEmails  []string
    allowedDomains []string
}

// モック（テスト用）
type MockChecker struct {
    IsAllowedFunc func(email string) bool
}
```

### テストの方針

#### 1. 単体テスト（Unit Tests）

**対象:**
- 各パッケージの個別機能
- ビジネスロジック
- バリデーション

**原則:**
- すべての外部依存はモックを使用
- テストカバレッジ目標: 80%以上
- テーブル駆動テスト（Table-Driven Tests）を推奨

**例:**
```go
func TestEmailChecker_IsAllowed(t *testing.T) {
    tests := []struct {
        name           string
        allowedEmails  []string
        allowedDomains []string
        email          string
        want           bool
    }{
        {
            name:          "個別アドレス許可",
            allowedEmails: []string{"user@example.com"},
            email:         "user@example.com",
            want:          true,
        },
        {
            name:           "ドメイン許可",
            allowedDomains: []string{"@example.com"},
            email:          "anyone@example.com",
            want:           true,
        },
        {
            name:          "許可なし",
            allowedEmails: []string{"user@example.com"},
            email:         "other@example.com",
            want:          false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            checker := &EmailChecker{
                allowedEmails:  tt.allowedEmails,
                allowedDomains: tt.allowedDomains,
            }
            if got := checker.IsAllowed(tt.email); got != tt.want {
                t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### 2. モックを使用したテスト例

```go
func TestEmailHandler_SendLoginLink(t *testing.T) {
    // モックメール送信者
    mockSender := &email.MockSender{}

    // モック認可チェッカー
    mockChecker := &authz.MockChecker{
        IsAllowedFunc: func(email string) bool {
            return email == "allowed@example.com"
        },
    }

    handler := &EmailHandler{
        sender:  mockSender,
        checker: mockChecker,
    }

    // テスト実行
    err := handler.SendLoginLink("allowed@example.com")

    // 検証
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(mockSender.Calls) != 1 {
        t.Fatalf("expected 1 email sent, got %d", len(mockSender.Calls))
    }

    if mockSender.Calls[0].To != "allowed@example.com" {
        t.Errorf("email sent to wrong address: %s", mockSender.Calls[0].To)
    }
}
```

#### 3. 統合テスト（Integration Tests）

**対象:**
- HTTPエンドポイント
- 認証フロー全体
- データベース連携（Redis等）

**ツール:**
- `httptest` パッケージを使用
- テスト用の軽量な依存関係（テスト用設定）

**例:**
```go
func TestLoginFlow(t *testing.T) {
    // テスト用サーバーのセットアップ
    mockSender := &email.MockSender{}
    server := setupTestServer(mockSender)
    defer server.Close()

    // ログインページにアクセス
    resp, err := http.Get(server.URL + "/login")
    if err != nil {
        t.Fatal(err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }
}
```

#### 4. テストユーティリティ

**共通のヘルパー関数:**
```go
// pkg/testing/helpers.go
package testing

func NewTestConfig() *config.Config {
    return &config.Config{
        Service: config.ServiceConfig{
            Name:        "Test Service",
            Description: "Test Description",
        },
        // ... テスト用設定
    }
}

func NewMockDependencies() *Dependencies {
    return &Dependencies{
        Sender:  &email.MockSender{},
        Store:   &session.MockStore{},
        Checker: &authz.MockChecker{},
        Logger:  &logging.MockLogger{},
    }
}
```

### テストの実行

```bash
# すべてのテストを実行
go test ./...

# カバレッジ付きでテスト
go test -cover ./...

# カバレッジレポート生成
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 特定のパッケージのテスト
go test ./pkg/auth/email

# verbose モード
go test -v ./...
```

### CI/CD統合

**GitHub Actions例:**
```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out
```

## 今後の拡張計画

### Phase 1: コア基盤構築（最小限の動作するプロダクト）
- [x] 設計書作成
- [ ] インターフェース設計とモック実装の基礎
- [ ] 単一OAuth2プロバイダー（Googleのみ）
- [ ] 基本的なプロキシ機能（単一アップストリーム）
- [ ] 静的YAML設定（リロードなし）
- [ ] インメモリセッション管理
- [ ] 最小限のUI（ログイン・ログアウトのみ）
- [ ] 基本的なロガー（英語のみ、シンプルフォーマット）
- [ ] 単体テスト基盤（カバレッジ60%以上）
- [ ] 認可機能（メールアドレス・ドメインホワイトリスト）

**目標**: 単一OAuth2プロバイダーで動作する最小限の認証プロキシを実現

### Phase 2: メール認証とセキュリティ強化
- [ ] メール認証機能（SMTP/SendGrid API対応）
- [ ] ワンタイムトークン管理（生成・検証・無効化）
- [ ] レート制限機能（メール送信・ログイン試行）
- [ ] セキュリティ強化（CSRF対策、ノンス）
- [ ] メールUI（アドレス入力フォーム、送信完了画面）
- [ ] テストカバレッジ向上（70%以上）

**目標**: パスワードレス認証を追加し、セキュリティを強化

### Phase 3: マルチプロバイダーと設定管理
- [ ] 複数OAuth2プロバイダー対応（GitHub, Microsoft等）
- [ ] Hostヘッダーベースルーティング
- [ ] 設定ファイルの自動リロード（fsnotify）
- [ ] 国際化対応（日本語・英語切替）
- [ ] カラー出力対応ロガー（TTY自動検出）
- [ ] UIの多言語化
- [ ] テストカバレッジ向上（80%以上）

**目標**: マルチテナント対応と柔軟な設定管理を実現

### Phase 4: 運用機能強化
- [ ] Redis セッションストア
- [ ] メトリクス収集（Prometheus対応）
- [ ] ログの構造化・レベル管理
- [ ] 管理API（認証ユーザー一覧、統計等）
- [ ] ヘルスチェック・Readinessプローブ
- [ ] Docker/Kubernetes対応
- [ ] アクセスログ・監査ログ

**目標**: 本番運用に必要な監視・管理機能を提供

### Phase 5: SSL/TLSと高度な機能
- [ ] lego統合によるLet's Encrypt自動化
- [ ] 証明書の自動更新
- [ ] HTTPS強制リダイレクト・HSTS対応
- [ ] MFA（多要素認証）サポート（オプション）
- [ ] WebAuthn/FIDO2対応（オプション）
- [ ] ユーザーグループ・ロール管理（オプション）
- [ ] より細かいアクセス制御（URL・パス単位）

**目標**: エンタープライズレベルのセキュリティと管理機能を実現

## 参考資料

### 認証・OAuth2
- [oauth2-proxy Documentation](https://oauth2-proxy.github.io/oauth2-proxy/)
- [OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [Go OAuth2 Package](https://pkg.go.dev/golang.org/x/oauth2)

### Go開発・テスト
- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests in Go](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### その他
- [water.css](https://watercss.kognise.dev/)
- [lego - Let's Encrypt Client](https://go-acme.github.io/lego/)
- [github.com/ideamans/go-l10n](https://github.com/ideamans/go-l10n)

## ライセンス

このプロジェクトは MIT ライセンスの下でライセンスされています。

```
MIT License

Copyright (c) 2025 multi-oauth2-proxy contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## 貢献

### コントリビューションガイドライン

このプロジェクトへの貢献を歓迎します！以下のガイドラインに従ってください。

#### プルリクエストの手順

1. **フォークとクローン**
   ```bash
   git clone https://github.com/yourusername/multi-oauth2-proxy.git
   cd multi-oauth2-proxy
   ```

2. **ブランチを作成**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **コードを実装**
   - インターフェースを適切に使用
   - 単体テストを追加（カバレッジ80%以上）
   - コメントを適切に記述

4. **テストを実行**
   ```bash
   go test -v -race -cover ./...
   ```

5. **コミットとプッシュ**
   ```bash
   git add .
   git commit -m "Add feature: your feature description"
   git push origin feature/your-feature-name
   ```

6. **プルリクエストを作成**
   - 変更内容を明確に説明
   - 関連するIssueをリンク

#### コーディング規約

- **Go標準に従う**: `gofmt`, `golint`, `go vet` を使用
- **インターフェース優先**: 外部依存は必ずインターフェース化
- **テスト必須**: 新機能には必ず単体テストを追加
- **ドキュメント**: 公開関数・型にはGoDocコメントを記述
- **エラーハンドリング**: エラーは適切にラップして返す

#### テストの要件

- すべてのパブリック関数にテストを追加
- テーブル駆動テストを推奨
- モックを使用して外部依存を分離
- カバレッジ80%以上を維持

#### レビュープロセス

1. CI/CDチェックが通過すること
2. コードレビューで承認されること
3. テストカバレッジが基準を満たすこと
