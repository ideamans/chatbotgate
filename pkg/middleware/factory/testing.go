package factory

import (
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestingFactory is a factory implementation for testing purposes.
// It embeds DefaultFactory and overrides specific methods to use
// in-memory stores and mock components suitable for testing.
type TestingFactory struct {
	*DefaultFactory
}

// NewTestingFactory creates a new TestingFactory with test-friendly defaults.
// It always uses in-memory KVS stores and a simple logger.
func NewTestingFactory(host string, port int) *TestingFactory {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	return &TestingFactory{
		DefaultFactory: NewDefaultFactory(host, port, logger),
	}
}

// NewTestingFactoryWithLogger creates a TestingFactory with a custom logger.
func NewTestingFactoryWithLogger(host string, port int, logger logging.Logger) *TestingFactory {
	return &TestingFactory{
		DefaultFactory: NewDefaultFactory(host, port, logger),
	}
}

// CreateKVSStores creates all in-memory KVS stores for testing.
// Unlike the production implementation, this always uses memory stores
// regardless of configuration, making tests faster and isolated.
func (f *TestingFactory) CreateKVSStores(cfg *config.Config) (session kvs.Store, token kvs.Store, rateLimit kvs.Store, err error) {
	// Set default namespace names
	cfg.KVS.Namespaces.SetDefaults()

	// Always use memory stores for testing
	sessionCfg := kvs.Config{
		Type:      "memory",
		Namespace: cfg.KVS.Namespaces.Session,
	}
	session, err = kvs.New(sessionCfg)
	if err != nil {
		return nil, nil, nil, err
	}

	tokenCfg := kvs.Config{
		Type:      "memory",
		Namespace: cfg.KVS.Namespaces.Token,
	}
	token, err = kvs.New(tokenCfg)
	if err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	rateLimitCfg := kvs.Config{
		Type:      "memory",
		Namespace: cfg.KVS.Namespaces.RateLimit,
	}
	rateLimit, err = kvs.New(rateLimitCfg)
	if err != nil {
		session.Close()
		token.Close()
		return nil, nil, nil, err
	}

	return session, token, rateLimit, nil
}

// CreateTokenKVS creates an in-memory KVS for email tokens with short cleanup interval
func (f *TestingFactory) CreateTokenKVS() kvs.Store {
	store, _ := kvs.NewMemoryStore("test-tokens", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Second, // Faster cleanup for tests
	})
	return store
}

// CreateRateLimitKVS creates an in-memory KVS for rate limiting with short cleanup interval
func (f *TestingFactory) CreateRateLimitKVS() kvs.Store {
	store, _ := kvs.NewMemoryStore("test-ratelimit", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Second, // Faster cleanup for tests
	})
	return store
}
