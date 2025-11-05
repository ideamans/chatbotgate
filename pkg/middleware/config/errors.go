package config

import "errors"

var (
	// ErrServiceNameRequired is returned when service name is not provided
	ErrServiceNameRequired = errors.New("service name is required")

	// ErrCookieSecretRequired is returned when cookie secret is not provided
	ErrCookieSecretRequired = errors.New("cookie secret is required")

	// ErrCookieSecretTooShort is returned when cookie secret is too short
	ErrCookieSecretTooShort = errors.New("cookie secret must be at least 32 characters")

	// ErrNoEnabledProviders is returned when no OAuth2 providers are enabled
	ErrNoEnabledProviders = errors.New("at least one OAuth2 provider must be enabled")

	// ErrConfigFileNotFound is returned when config file is not found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrEncryptionKeyRequired is returned when encryption is enabled but key is not provided
	ErrEncryptionKeyRequired = errors.New("encryption key is required when encryption is enabled")

	// ErrEncryptionKeyTooShort is returned when encryption key is too short
	ErrEncryptionKeyTooShort = errors.New("encryption key must be at least 32 characters")

	// ErrEncryptionConfigRequired is returned when encrypt filter is used but encryption config is not provided
	ErrEncryptionConfigRequired = errors.New("encryption configuration is required when 'encrypt' filter is used")
)
