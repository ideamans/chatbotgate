# ChatbotGate

[![CI](https://github.com/ideamans/chatbotgate/actions/workflows/ci.yml/badge.svg)](https://github.com/ideamans/chatbotgate/actions/workflows/ci.yml)
[![Release](https://github.com/ideamans/chatbotgate/actions/workflows/release.yml/badge.svg)](https://github.com/ideamans/chatbotgate/actions/workflows/release.yml)
[![Docker Hub](https://img.shields.io/docker/v/ideamans/chatbotgate?label=docker&logo=docker)](https://hub.docker.com/r/ideamans/chatbotgate)
[![Go Report Card](https://goreportcard.com/badge/github.com/ideamans/chatbotgate)](https://goreportcard.com/report/github.com/ideamans/chatbotgate)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[English](README.md) | [日本語](#chatbotgate)

**ChatbotGate**は、軽量で柔軟な認証リバースプロキシです。アップストリームアプリケーションの前段に配置し、複数のOAuth2プロバイダーとパスワードレスメール認証を通じて統合認証機能を提供します。

## 主な機能

### 🔐 複数の認証方式
- **OAuth2/OIDC**: Google、GitHub、Microsoft、カスタムOIDCプロバイダー
- **パスワードレスメール認証**: SMTP、SendGrid、sendmail経由のマジックリンク認証
- 異なるユーザーグループ向けにプロバイダーを組み合わせ可能

### 🛡️ 柔軟なアクセス制御
- メールアドレス・ドメインベースのホワイトリスト
- パスベースのアクセスルール（許可、認証必須、拒否）
- パターンマッチング（完全一致、プレフィックス、正規表現、minimatch）
- 最初にマッチしたルールが優先

### 🔄 シームレスなリバースプロキシ
- HTTP/WebSocketリクエストの透過的なプロキシ
- Server-Sent Events (SSE) ストリーミングサポート
- X-Forwardedヘッダー（X-Real-IP、X-Forwarded-For、X-Forwarded-Proto、X-Forwarded-Host）
- 32KBバッファプールによる大容量ファイル処理
- 設定可能な認証パスプレフィックス（デフォルト: `/_auth`）
- ホストベースルーティングによるマルチテナント対応
- アップストリームへの自動シークレットヘッダー付与

### 📦 複数のストレージバックエンド
- **Memory**: 開発用の高速エフェメラルストレージ
- **LevelDB**: 永続化された組み込みデータベース
- **Redis**: 本番環境向けの分散スケーラブルストレージ
- 名前空間分離を備えた統一KVSインターフェース

### 🎨 使いやすいインターフェース
- クリーンでレスポンシブな認証UI
- 多言語サポート（英語/日本語）
- テーマ切り替え（自動/ライト/ダーク）
- カスタマイズ可能なブランディング（ロゴ、アイコン、カラー）

### 🔌 ユーザー情報転送
- 認証済みユーザーデータをアップストリームアプリに転送
- 柔軟なフィールドマッピング（メール、ユーザー名、プロバイダーなど）
- 暗号化・圧縮サポート（AES-256-GCM、gzip）
- クエリパラメータとHTTPヘッダー

### ⚙️ 本番環境対応
- **環境変数展開**: 設定ファイル内で`${VAR:-default}`形式をサポート
- ライブ設定リロード（ほとんどの設定）
- 設定検証ツール（`test-config`）
- シェル補完（bash、zsh、fish、powershell）
- ヘルスチェックエンドポイント（`/_auth/health`）
- 構造化ロギングと設定可能なログレベル
- メール送信レート制限（マジックリンクメールの悪用防止）
- 包括的なテストカバレッジ
- マルチアーキテクチャ対応Dockerイメージ（amd64/arm64）

## クイックスタート

### インストール

**ソースから:**
```bash
git clone https://github.com/ideamans/chatbotgate.git
cd chatbotgate
go build -o chatbotgate ./cmd/chatbotgate
```

**Dockerを使用:**
```bash
# 最新版をpull（マルチアーキテクチャ: amd64/arm64）
docker pull ideamans/chatbotgate:latest

# または特定バージョンをpull
docker pull ideamans/chatbotgate:v1.0.0
```

Dockerイメージは[Docker Hub](https://hub.docker.com/r/ideamans/chatbotgate)でリリース毎に自動ビルド・公開されます。

### 基本設定

`config.yaml`ファイルを作成します。環境変数は`${VAR}`または`${VAR:-default}`構文で使用できます:

```yaml
service:
  name: "My App Auth"

server:
  host: "0.0.0.0"
  port: 4180
  # OAuth2コールバック用のベースURL（自動生成: {base_url}/_auth/oauth2/callback）
  # リバースプロキシ配下やHTTPS使用時に設定
  # base_url: "https://your-domain.com"

proxy:
  upstream:
    # フォールバック付き環境変数を使用
    url: "${UPSTREAM_URL:-http://localhost:8080}"

session:
  cookie:
    # シークレットは環境変数から取得（推奨）
    secret: "${COOKIE_SECRET:-CHANGE-THIS-TO-A-RANDOM-SECRET}"
    expire: "168h"

oauth2:
  providers:
    - id: "google"
      type: "google"
      # 環境変数から認証情報を取得
      client_id: "${GOOGLE_CLIENT_ID}"
      client_secret: "${GOOGLE_CLIENT_SECRET}"

access_control:
  emails:
    - "@example.com"  # @example.comのすべてのメールを許可
```

**環境変数の設定:**

```bash
export COOKIE_SECRET="$(openssl rand -base64 32)"
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"
export UPSTREAM_URL="http://localhost:8080"
```

### 設定の検証

起動前に設定ファイルを検証します:

```bash
./chatbotgate test-config -c config.yaml
```

### サーバーの起動

```bash
./chatbotgate -config config.yaml
```

`http://localhost:4180`にアクセスして認証フローを確認できます。

### シェル補完（オプション）

CLI操作を簡単にするシェル補完を生成:

```bash
# Bash
./chatbotgate completion bash > /etc/bash_completion.d/chatbotgate

# Zsh
./chatbotgate completion zsh > ~/.zsh/completion/_chatbotgate

# Fish
./chatbotgate completion fish > ~/.config/fish/completions/chatbotgate.fish

# PowerShell
./chatbotgate completion powershell > chatbotgate.ps1
```

## ドキュメント

- **[ユーザーガイド (GUIDE.md)](GUIDE.md)** - ChatbotGateの展開・設定に関する完全ガイド（英語）
- **[モジュールガイド (MODULE.md)](MODULE.md)** - GoモジュールとしてChatbotGateを使用する開発者向けガイド（英語）
- **[Examplesディレクトリ](examples/)** - 本番環境対応のデプロイ例（Docker、systemd、完全な設定）

## プロジェクト構造

```
chatbotgate/
├── cmd/
│   └── chatbotgate/          # メインエントリポイントとCLI
├── pkg/
│   ├── middleware/           # 認証ミドルウェア
│   │   ├── auth/             # OAuth2とメール認証
│   │   ├── authz/            # 認可（ホワイトリスト）
│   │   ├── session/          # セッション管理
│   │   ├── rules/            # パスベースアクセス制御
│   │   ├── forwarding/       # ユーザー情報転送
│   │   └── ...
│   ├── proxy/                # リバースプロキシ
│   └── shared/               # 共有コンポーネント
│       ├── kvs/              # Key-Valueストアインターフェース
│       ├── i18n/             # 国際化
│       └── logging/          # 構造化ロギング
├── web/                      # Webアセット（HTML、CSS、JS）
├── email/                    # メールテンプレート
├── e2e/                      # End-to-Endテスト
├── config.example.yaml       # 設定例
└── README.md                 # このファイル
```

## 動作の仕組み

```
┌─────────┐      ┌──────────────┐      ┌──────────┐
│  ユーザー │─────▶│ ChatbotGate  │─────▶│アップスト │
│ブラウザ  │      │    (認証)    │      │ リーム   │
└─────────┘      └──────────────┘      └──────────┘
     ▲                  │
     │                  ▼
     │           ┌─────────────┐
     └───────────│   OAuth2    │
                 │  Provider   │
                 └─────────────┘
```

1. **ユーザーリクエスト** → ChatbotGateが認証をチェック
2. **未認証の場合** → `/_auth/login`にリダイレクト
3. **ユーザーが選択** → OAuth2またはメール認証
4. **認証成功** → セッション作成、元のURLにリダイレクト
5. **認証済みリクエスト** → ユーザー情報ヘッダー付きでアップストリームにプロキシ

## 開発

### 前提条件

- Go 1.21以降
- Node.js 20+（webアセットとe2eテスト用）
- Docker & Docker Compose（オプション、コンテナ開発用）
- Redis（オプション、分散セッション用）

### ビルド

```bash
# すべてをビルド（webアセット + Goバイナリ）
make build

# Goバイナリのみ
make build-go

# webアセットのみ
make build-web
```

### コード品質

```bash
# コードフォーマット
make fmt

# フォーマットチェック（CI）
make fmt-check

# Linterを実行
make lint

# すべてのCIチェックを実行（format + lint + test）
make ci
```

### テスト実行

```bash
# すべてのユニットテストを実行
make test

# カバレッジレポート付きテスト
make test-coverage

# 特定パッケージのテスト
go test ./pkg/middleware/auth/oauth2

# 詳細出力
go test -v ./pkg/...

# e2eテスト（Dockerが必要）
cd e2e && make test
```

### Dockerビルド

```bash
# イメージをビルド
docker build -t chatbotgate .

# docker-composeで実行
docker-compose up
```

## 設定

ChatbotGateはYAMLで設定し、環境変数展開（`${VAR:-default}`）をサポートします。

**基本例:**

```yaml
service:
  name: "My App"

server:
  port: 4180

session:
  cookie:
    secret: "${COOKIE_SECRET}"  # 環境変数
    expire: "168h"

oauth2:
  providers:
    - id: "google"
      type: "google"
      client_id: "${GOOGLE_CLIENT_ID}"
      client_secret: "${GOOGLE_CLIENT_SECRET}"

proxy:
  upstream:
    url: "http://localhost:8080"
```

**完全な設定については:**
- [config.example.yaml](config.example.yaml) - すべてのオプションを含む包括的な例
- [GUIDE.md - Configuration](GUIDE.md#configuration) - 詳細な設定ガイド（英語）
- [examples/](examples/) - 本番環境対応のデプロイ例

## ユースケース

- **チャットボットウィジェット認証**: OAuth2またはメール認証でチャットボットインターフェース（Dify、Rasaなど）を保護
- **内部ツールアクセス制御**: 独自の認証システムを持たない内部ツールに認証を追加
- **マルチテナントアプリケーション**: ホスト名に基づいて異なるアップストリームバックエンドにルーティング
- **認証付きAPIゲートウェイ**: リバースプロキシと認証をマイクロサービス向けに組み合わせ

## ロギング

ChatbotGateは複数のバックエンドで構造化ロギングをサポート:

```bash
# systemd/journald（本番環境推奨）
journalctl -u chatbotgate -f

# Dockerログ
docker logs -f chatbotgate

# ファイルロギング（非systemd環境向け）
# config.yamlのlogging.fileで設定
```

**完全なロギングドキュメント**については、[GUIDE.md - Logging](GUIDE.md#logging)を参照してください（英語）

## ヘルスチェック

ChatbotGateはコンテナオーケストレーション用のヘルスチェックエンドポイントを提供:

- **Readiness**: `GET /_auth/health`（準備完了時200、起動中/終了中503）
- **Liveness**: `GET /_auth/health?probe=live`（プロセス生存時200）

**Dockerヘルスチェック例:**
```yaml
healthcheck:
  test: ["CMD-SHELL", "curl -fsS http://localhost:4180/_auth/health || exit 1"]
  interval: 5s
  timeout: 2s
  retries: 12
```

**完全なヘルスチェックドキュメント**については、[GUIDE.md - Health Checks](GUIDE.md#health-check-endpoints)を参照してください（英語）

## セキュリティに関する考慮事項

- **Cookie Secret**: 強力なランダムシークレットを使用（32文字以上）
- **HTTPS**: 本番環境では常にHTTPSを使用（`cookie_secure: true`を設定）
- **シークレットの保管**: 機密データには環境変数またはシークレットマネージャーを使用
- **アップストリームシークレット**: シークレットヘッダーでアップストリームへの直接アクセスを保護
- **ホワイトリスト**: 可能な限りメール/ドメインでアクセスを制限
- **メールレート制限**: マジックリンクメールの悪用を防ぐため`email_auth.limit_per_minute`を設定（デフォルト: 5/分）

**包括的なセキュリティガイド**については、[GUIDE.md - Security Best Practices](GUIDE.md#security-best-practices)を参照してください（英語）

## コントリビューション

コントリビューションを歓迎します！以下の手順で:

1. リポジトリをFork
2. フィーチャーブランチを作成（`git checkout -b feature/amazing-feature`）
3. 変更をコミット（`git commit -m 'Add amazing feature'`）
4. ブランチにプッシュ（`git push origin feature/amazing-feature`）
5. プルリクエストを開く

### 開発ガイドライン

- Goのベストプラクティスとイディオムに従う
- 新機能にはテストを書く（カバレッジ80%以上を目標）
- ユーザー向け変更にはドキュメントを更新
- コミット前に`make ci`を実行してすべてのチェックが通ることを確認
- `make fmt`でコードをフォーマット
- コミットはアトミックに保ち、明確なコミットメッセージを書く

**モジュール開発**については、[MODULE.md](MODULE.md)を参照してください - GoモジュールとしてChatbotGateを使用する開発者向けガイド（英語）

### CI/CDパイプライン

ChatbotGateはGitHub Actionsを使用した継続的インテグレーションとデプロイ:

- **CI**（push/PR時）: Linting、フォーマットチェック、ユニットテスト、e2eテストを実行
- **Release**（tag時）: GoReleaserでバイナリをビルドし、Docker HubにDockerイメージを公開
- **Dockerイメージ**: マルチアーキテクチャ（amd64/arm64）イメージを`ideamans/chatbotgate`に公開

## ライセンス

このプロジェクトはMITライセンスの下でライセンスされています - 詳細は[LICENSE](LICENSE)ファイルを参照してください。

## サポート

- **Issues**: [GitHub Issues](https://github.com/ideamans/chatbotgate/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ideamans/chatbotgate/discussions)

## 謝辞

- [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/)にインスパイアされました
- [Go](https://golang.org/)、[fsnotify](https://github.com/fsnotify/fsnotify)、および[多くの優れたライブラリ](go.mod)で構築

---

**認証に課題を抱えるWebのために❤️を込めて作られました**
