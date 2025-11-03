# ChatbotGate リファクタリング計画

## 設計方針

### コアアーキテクチャ

ChatbotGateのアーキテクチャは以下の3層構造：

```
外部（標準 http.Handler）
    ↓
Manager（http.Handler） - Middleware実体のルーター
    ↓
Middleware実体 - 認証ロジックの本体
    ↓
next http.Handler（Proxy等）
```

### Managerの役割

**Manager = ただのhttp.Handlerラッパー**

- **本質的な責務**: リクエストを現在のMiddleware実体に委譲するだけ
- **ServeHTTP**: `current.Load().(*middleware.Middleware).ServeHTTP(w, r)`
- **それ以上でもそれ以下でもない**

**Reload機能について:**
- 本質的な機能ではない
- たまたま実装した便利メソッド
- インターフェース化は不要
- 使いたければ使う、使いたくなければ使わない

**Watcherについて:**
- 完全にオプショナルな機能
- 本質ではない
- ファイル監視してReload()を呼ぶだけのユーティリティ

### 依存性注入（DI）の方針

**Factory = 簡易的なDIコンテナ**

- すべてのコンポーネント生成ロジックを集約
- `DefaultFactory`がベースライン実装
- カスタム実装は`DefaultFactory`に委譲し、特定メソッドだけオーバーライド
- serve.goから初期化ロジックを排除

**インターフェース化の原則:**

✅ **インターフェース化すべき:**
- 低レイヤーコンポーネント（Store, Logger, Checker等）
- 複数の実装が明確に想定されるもの
- テストでモック化が必要なもの

❌ **インターフェース化不要:**
- 高レイヤーの統合コンポーネント（Manager, Watcher）
- 単一責務が明確なもの
- 具体型で十分なもの

### モジュール階層

```
pkg/
├── factory/          # DIコンテナ（すべてのコンポーネント生成）
│   ├── factory.go    # Factoryインターフェース
│   ├── default.go    # DefaultFactory実装
│   └── builder.go    # ApplicationBuilder（高レベルAPI）
│
├── manager/          # Middleware実体のルーター
│   ├── single_domain.go   # 単一ドメイン実装
│   └── multi_domain.go    # マルチドメイン実装（将来）
│
├── middleware/       # 認証ロジック本体
│   ├── middleware.go      # Middleware実体
│   ├── handlers.go        # 認証エンドポイント
│   ├── validate.go        # バリデーション
│   └── helpers.go         # ヘルパー関数
│
├── auth/             # 認証機能
│   ├── oauth2/       # OAuth2認証
│   └── email/        # メール認証
│
├── authz/            # 認可（アクセス制御）
├── session/          # セッション管理
├── kvs/              # KVSストレージ抽象化
│
├── forwarding/       # ユーザー情報転送
├── passthrough/      # 認証バイパス
├── proxy/            # リバースプロキシ（差し替え可能）
│
├── config/           # 設定管理
├── watcher/          # 設定ファイル監視（オプショナル）
├── i18n/             # 国際化
├── logging/          # ロギング
└── ui/               # UIテンプレート
```

---

## リファクタリング計画

### フェーズ1: インターフェース整理【優先度: 高】

#### 1.1 Forwardingのインターフェース化

**現状:**
```go
type Middleware struct {
    forwarder *forwarding.Forwarder  // 具体型依存
}
```

**目標:**
```go
// pkg/forwarding/forwarder.go
type Forwarder interface {
    AddToHeaders(headers http.Header, userInfo *UserInfo) http.Header
    AddToQueryString(targetURL string, userInfo *UserInfo) (string, error)
}

type DefaultForwarder struct {
    // 現在の実装
}

// pkg/middleware/middleware.go
type Middleware struct {
    forwarder forwarding.Forwarder  // インターフェース型
}
```

**タスク:**
- [ ] `pkg/forwarding/forwarder.go`にForwarderインターフェース定義
- [ ] 既存実装を`DefaultForwarder`にリネーム
- [ ] `middleware.Middleware`の型をインターフェースに変更
- [ ] Factoryで`DefaultForwarder`を生成
- [ ] テスト追加

#### 1.2 Passthroughのインターフェース化

**現状:**
```go
type Middleware struct {
    passthroughMatcher *passthrough.Matcher  // 具体型依存
}
```

**目標:**
```go
// pkg/passthrough/passthrough.go
type Matcher interface {
    Match(requestPath string) bool
    HasErrors() bool
    Errors() []error
}

type PatternMatcher struct {
    // 現在の実装
}

// pkg/middleware/middleware.go
type Middleware struct {
    passthroughMatcher passthrough.Matcher  // インターフェース型
}
```

**タスク:**
- [ ] `pkg/passthrough/passthrough.go`にMatcherインターフェース定義
- [ ] 既存実装を`PatternMatcher`にリネーム
- [ ] `middleware.Middleware`の型をインターフェースに変更
- [ ] Factoryで`PatternMatcher`を生成
- [ ] テスト追加

#### 1.3 Managerのインターフェース削除

**現状:**
```go
// Reloadableインターフェースが存在（不要）
type Reloadable interface {
    Reload(cfg *config.Config) error
    GetConfig() *config.Config
}
```

**目標:**
```go
// pkg/manager/single_domain.go
type SingleDomainManager struct {
    current atomic.Value
    factory factory.Factory
    // ...
}

func (m *SingleDomainManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    mw := m.current.Load().(*middleware.Middleware)
    mw.ServeHTTP(w, r)
}

// Reload is just a convenience method - not part of any interface
func (m *SingleDomainManager) Reload(newConfig *config.Config) error {
    // ...
}
```

**タスク:**
- [ ] `Reloadable`インターフェース削除
- [ ] `manager.MiddlewareManager`を`manager.SingleDomainManager`にリネーム
- [ ] `Reload`メソッドを通常のメソッドとして保持
- [ ] `GetConfig`メソッド削除（不要）
- [ ] Watcherを具体型依存に変更
- [ ] テスト更新

---

### フェーズ2: Factory導入【優先度: 高】✅

#### 2.1 Factoryインターフェース設計 ✅

**実装完了:**
```go
// pkg/factory/factory.go
type Factory interface {
    CreateMiddleware(cfg, sessionStore, proxyHandler, logger) (*middleware.Middleware, error)
    CreateOAuth2Manager(cfg, host, port) *oauth2.Manager
    CreateEmailHandler(cfg, host, port, authzChecker, translator, tokenKVS, rateLimitKVS) (*email.Handler, error)
    CreateAuthzChecker(cfg) authz.Checker
    CreateForwarder(cfg) forwarding.Forwarder
    CreatePassthroughMatcher(cfg) passthrough.Matcher
    CreateTranslator() *i18n.Translator
    CreateTokenKVS() kvs.Store
    CreateRateLimitKVS() kvs.Store
}
```

**タスク:**
- [x] `pkg/factory/factory.go`作成（インターフェース定義）
- [x] `pkg/factory/default.go`作成（DefaultFactory実装）
- [x] `manager.createMiddleware`のロジック（145行）をFactoryに移動
- [x] OAuth2プロバイダー設定ロジックをFactoryに移動
- [x] ManagerをFactory依存に変更
- [x] すべてのテスト更新（manager: 7件, watcher: 6件）
- [x] ビルド確認

#### 2.2 Factoryメソッド拡張によるserve.go簡素化 ✅

**実装方針変更:**
ApplicationBuilderパターンはインポートサイクルの問題があるため、代わりにFactoryインターフェースにメソッドを追加してserve.goを簡素化。

**実装内容:**
```go
// pkg/factory/factory.go
type Factory interface {
    // 既存メソッド...

    // 追加メソッド
    CreateKVSStores(cfg *config.Config) (session, token, rateLimit kvs.Store, err error)
    CreateSessionStore(kvsStore kvs.Store) session.Store
    CreateProxyHandler(cfg *config.Config) (*proxy.Handler, error)
}
```

**成果:**
- `pkg/factory/default.go`に3つの新メソッドを実装
  - `CreateKVSStores()`: 100行以上のKVS初期化ロジックをカプセル化
  - `CreateSessionStore()`: KVSからSessionStoreを生成
  - `CreateProxyHandler()`: Proxy設定からHandlerを生成
- `cmd/chatbotgate/cmd/serve.go`を250行→153行に削減（約40%減）
- すべてのテスト合格

**タスク:**
- [x] Factoryインターフェースに3つのメソッドを追加
- [x] DefaultFactoryに実装追加
- [x] serve.goでFactoryメソッドを使用
- [x] 不要なインポートを削除
- [x] ビルド確認
- [x] テスト実行

#### 2.3 serve.goの簡素化結果

**変更前:** 250行
**変更後:** 153行（40%削減）

**目標:**
```go
func runServe(cmd *cobra.Command, args []string) error {
    // Build application using factory
    app, err := factory.NewApplicationBuilder(cfgFile, host, port).Build()
    if err != nil {
        return err
    }
    defer app.Close()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start optional config watcher
    if watchConfig {
        go app.ConfigWatcher.Watch(ctx)
    }

    // Create HTTP server
    httpServer := &http.Server{
        Addr:    fmt.Sprintf("%s:%d", host, port),
        Handler: app.Manager,
    }

    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        cancel()
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer shutdownCancel()
        httpServer.Shutdown(shutdownCtx)
    }()

    return httpServer.ListenAndServe()
}
```

**タスク:**
- [ ] `cmd/chatbotgate/cmd/serve.go`をリファクタリング
- [ ] 初期化ロジックを`ApplicationBuilder`に委譲
- [ ] 150行→50行以下に削減
- [ ] テスト追加

---

### フェーズ3: テスト環境整備【優先度: 中】✅

#### 3.1 TestingFactory実装 ✅

**実装完了:**
```go
// pkg/factory/testing_factory.go
type TestingFactory struct {
    *DefaultFactory
}

func NewTestingFactory(host string, port int) *TestingFactory
func NewTestingFactoryWithLogger(host string, port int, logger logging.Logger) *TestingFactory

// Override to always use in-memory KVS
func (f *TestingFactory) CreateKVSStores(cfg *config.Config) (session, token, rateLimit kvs.Store, err error)
func (f *TestingFactory) CreateTokenKVS() kvs.Store
func (f *TestingFactory) CreateRateLimitKVS() kvs.Store

// Test helper functions
func CreateTestConfig() *config.Config
func CreateTestConfigWithOAuth2() *config.Config
func CreateTestConfigWithEmail() *config.Config
```

**成果:**
- **TestingFactory**: DefaultFactoryを埋め込み、KVSメソッドをオーバーライドして常にメモリストアを使用
- **テストヘルパー**: 3つの設定作成関数を実装
  - `CreateTestConfig()`: 基本的なテスト設定
  - `CreateTestConfigWithOAuth2()`: OAuth2プロバイダー付き設定
  - `CreateTestConfigWithEmail()`: メール認証付き設定
- **包括的テスト**:
  - `default_test.go`: 13テストケース（DefaultFactoryの全メソッド）
  - `testing_factory_test.go`: 9テストケース（TestingFactoryと設定ヘルパー）
  - すべてのテストが合格

**タスク:**
- [x] `pkg/factory/testing_factory.go`作成
- [x] メモリKVS強制（CreateKVSStoresをオーバーライド）
- [x] テストヘルパー追加（CreateTestConfig系）
- [x] Factory包括的テスト作成
- [x] 全テスト合格確認

---

### フェーズ4: マルチドメイン対応準備【優先度: 低】

#### 4.1 MultiDomainManager設計

**目標:**
```go
// pkg/manager/multi_domain.go
type MultiDomainManager struct {
    domains      map[string]*middleware.Middleware
    defaultDomain *middleware.Middleware
    factory      factory.Factory
    mu           sync.RWMutex
}

func (m *MultiDomainManager) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (m *MultiDomainManager) Reload(configs map[string]*config.Config) error
func (m *MultiDomainManager) ReloadDomain(domain string, cfg *config.Config) error
```

**タスク:**
- [ ] `pkg/manager/multi_domain.go`作成（スケルトン）
- [ ] Hostベースルーティング実装
- [ ] 設定ファイルフォーマット検討
- [ ] テスト追加

#### 4.2 MultiDomainFactory実装

**目標:**
```go
// pkg/factory/multi_domain_factory.go
type MultiDomainFactory struct {
    *DefaultFactory
    domainConfigs map[string]*config.Config
}

func (f *MultiDomainFactory) CreateManager(...) (http.Handler, error) {
    // MultiDomainManagerを返す
}
```

**タスク:**
- [ ] `pkg/factory/multi_domain_factory.go`作成
- [ ] マルチドメイン設定ローダー実装
- [ ] テスト追加

---

### フェーズ5: ドキュメント整備【優先度: 中】

#### 5.1 アーキテクチャドキュメント

**タスク:**
- [ ] `docs/architecture.md` - アーキテクチャ図と説明
- [ ] `docs/factory.md` - Factory使用方法とカスタマイズ例
- [ ] `docs/testing.md` - テスト方法とTestingFactory使用例
- [ ] `docs/multi_domain.md` - マルチドメイン対応の設計（将来）

#### 5.2 カスタマイズ例

**タスク:**
- [ ] `examples/custom_factory/` - カスタムFactory実装例
- [ ] `examples/custom_forwarder/` - カスタムForwarder実装例
- [ ] `examples/production_setup/` - 本番環境セットアップ例

---

## 実装順序

### Step 1: インターフェース整理（1-2週間）
1. Forwardingインターフェース化
2. Passthroughインターフェース化
3. Managerインターフェース削除

### Step 2: Factory導入（2-3週間）
1. Factoryインターフェース実装
2. ApplicationBuilder実装
3. serve.go簡素化

### Step 3: テスト環境整備（1週間）
1. TestingFactory実装
2. 既存テスト移行

### Step 4: ドキュメント整備（1週間）
1. アーキテクチャドキュメント
2. カスタマイズ例

### Step 5: マルチドメイン対応（将来）
1. MultiDomainManager実装
2. MultiDomainFactory実装

---

## マイルストーン

### M1: 基盤整備（4週間）
- フェーズ1完了（インターフェース整理）
- フェーズ2完了（Factory導入）

### M2: テスト・ドキュメント（2週間）
- フェーズ3完了（テスト環境）
- フェーズ5完了（ドキュメント）

### M3: 拡張対応（将来）
- フェーズ4完了（マルチドメイン）

---

## 後方互換性

### 破壊的変更
- `manager.MiddlewareManager` → `manager.SingleDomainManager` (型名変更)
- `Reloadable`インターフェース削除
- `forwarding.Forwarder` → `forwarding.DefaultForwarder` (具体型名変更)
- `passthrough.Matcher` → `passthrough.PatternMatcher` (具体型名変更)

### 互換性維持
- 設定ファイルフォーマット（変更なし）
- HTTPエンドポイント（変更なし）
- 認証フロー（変更なし）
- 外部からのAPI（http.Handlerとして動作）

### 移行パス
1. v1.x → v2.0: Factory導入、型名変更（メジャーバージョンアップ）
2. 移行ガイド提供
3. 既存ユーザーへのアナウンス

---

## 品質基準

### コードカバレッジ
- 目標: 80%以上
- 重点: Factory、Manager、Middleware

### パフォーマンス
- Reload時のダウンタイム: 0ms（atomic swap）
- メモリオーバーヘッド: 最小限
- ベンチマーク追加

### ドキュメント
- すべてのパブリックインターフェースにGoDoc
- アーキテクチャ図
- カスタマイズ例

---

## リスク管理

### リスク1: 破壊的変更による影響
- **対策**: メジャーバージョンアップ、移行ガイド提供

### リスク2: リファクタリング中のバグ混入
- **対策**: テストカバレッジ向上、段階的リリース

### リスク3: スケジュール遅延
- **対策**: フェーズごとにリリース、優先度調整可能
