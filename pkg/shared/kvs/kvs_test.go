package kvs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew tests the New function with different store types
func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errContains string
		description string
	}{
		{
			name: "memory store with empty type",
			config: Config{
				Type: "",
			},
			expectError: false,
			description: "Empty type should default to memory",
		},
		{
			name: "memory store explicitly",
			config: Config{
				Type: "memory",
			},
			expectError: false,
			description: "Should create memory store",
		},
		{
			name: "leveldb store",
			config: Config{
				Type: "leveldb",
			},
			expectError: false,
			description: "Should create leveldb store",
		},
		{
			name: "unsupported store type",
			config: Config{
				Type: "invalid-type",
			},
			expectError: true,
			errContains: "unsupported store type",
			description: "Should return error for unsupported type",
		},
		{
			name: "unknown store type",
			config: Config{
				Type: "postgres",
			},
			expectError: true,
			errContains: "unsupported store type",
			description: "Should return error for unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := New(tt.config)

			if tt.expectError {
				require.Error(t, err, tt.description)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "Error should contain expected text")
				}
				assert.Nil(t, store, "Store should be nil on error")
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, store, "Store should not be nil")
				defer func() { _ = store.Close() }()
			}
		})
	}
}

// TestNewWithNamespace tests creating stores with different namespaces
func TestNewWithNamespace(t *testing.T) {
	tests := []struct {
		name      string
		storeType string
		namespace string
	}{
		{
			name:      "memory store with namespace",
			storeType: "memory",
			namespace: "test-namespace",
		},
		{
			name:      "memory store without namespace",
			storeType: "memory",
			namespace: "",
		},
		{
			name:      "leveldb store with namespace",
			storeType: "leveldb",
			namespace: "test-leveldb",
		},
		{
			name:      "leveldb store with special characters in namespace",
			storeType: "leveldb",
			namespace: "test@namespace#123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Type:      tt.storeType,
				Namespace: tt.namespace,
			}

			store, err := New(config)
			require.NoError(t, err, "Should create store with namespace")
			require.NotNil(t, store, "Store should not be nil")
			defer func() { _ = store.Close() }()
		})
	}
}
