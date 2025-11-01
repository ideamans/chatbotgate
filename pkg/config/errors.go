package config

import "errors"

var (
	// ErrServiceNameRequired is returned when service name is not provided
	ErrServiceNameRequired = errors.New("service name is required")

	// ErrInvalidPort is returned when port is invalid
	ErrInvalidPort = errors.New("port must be between 1 and 65535")

	// ErrUpstreamRequired is returned when upstream is not provided
	ErrUpstreamRequired = errors.New("proxy upstream is required")

	// ErrCookieSecretRequired is returned when cookie secret is not provided
	ErrCookieSecretRequired = errors.New("cookie secret is required")

	// ErrCookieSecretTooShort is returned when cookie secret is too short
	ErrCookieSecretTooShort = errors.New("cookie secret must be at least 32 characters")

	// ErrNoEnabledProviders is returned when no OAuth2 providers are enabled
	ErrNoEnabledProviders = errors.New("at least one OAuth2 provider must be enabled")

	// ErrConfigFileNotFound is returned when config file is not found
	ErrConfigFileNotFound = errors.New("configuration file not found")
)
