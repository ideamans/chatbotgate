package forwarding

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
)

// DataType represents the type of data (string or binary)
type DataType int

const (
	DataTypeString DataType = iota
	DataTypeBinary
)

// FilterOutput holds the result of a filter operation
type FilterOutput struct {
	Data []byte
	Type DataType
}

// Filter represents a data transformation filter
type Filter interface {
	// Name returns the filter name
	Name() string
	// InputTypes returns acceptable input types
	InputTypes() []DataType
	// OutputType returns the output type
	OutputType() DataType
	// Apply applies the filter to the input data
	Apply(input *FilterOutput) (*FilterOutput, error)
}

// EncryptFilter applies AES-256-GCM encryption
type EncryptFilter struct {
	encryptor *Encryptor
}

func NewEncryptFilter(encryptor *Encryptor) *EncryptFilter {
	return &EncryptFilter{encryptor: encryptor}
}

func (f *EncryptFilter) Name() string {
	return "encrypt"
}

func (f *EncryptFilter) InputTypes() []DataType {
	return []DataType{DataTypeString}
}

func (f *EncryptFilter) OutputType() DataType {
	return DataTypeBinary
}

func (f *EncryptFilter) Apply(input *FilterOutput) (*FilterOutput, error) {
	// Encrypt returns base64 string, but we treat it as binary for the pipeline
	encrypted, err := f.encryptor.Encrypt(string(input.Data))
	if err != nil {
		return nil, fmt.Errorf("encrypt filter: %w", err)
	}

	// Store encrypted base64 string as bytes (binary type)
	return &FilterOutput{
		Data: []byte(encrypted),
		Type: DataTypeBinary,
	}, nil
}

// ZipFilter applies gzip compression
type ZipFilter struct{}

func NewZipFilter() *ZipFilter {
	return &ZipFilter{}
}

func (f *ZipFilter) Name() string {
	return "zip"
}

func (f *ZipFilter) InputTypes() []DataType {
	return []DataType{DataTypeString, DataTypeBinary}
}

func (f *ZipFilter) OutputType() DataType {
	return DataTypeBinary
}

func (f *ZipFilter) Apply(input *FilterOutput) (*FilterOutput, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(input.Data); err != nil {
		return nil, fmt.Errorf("zip filter: write: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("zip filter: close: %w", err)
	}

	return &FilterOutput{
		Data: buf.Bytes(),
		Type: DataTypeBinary,
	}, nil
}

// Base64Filter applies base64 encoding
type Base64Filter struct{}

func NewBase64Filter() *Base64Filter {
	return &Base64Filter{}
}

func (f *Base64Filter) Name() string {
	return "base64"
}

func (f *Base64Filter) InputTypes() []DataType {
	return []DataType{DataTypeBinary}
}

func (f *Base64Filter) OutputType() DataType {
	return DataTypeString
}

func (f *Base64Filter) Apply(input *FilterOutput) (*FilterOutput, error) {
	encoded := base64.StdEncoding.EncodeToString(input.Data)
	return &FilterOutput{
		Data: []byte(encoded),
		Type: DataTypeString,
	}, nil
}

// FilterChain represents a chain of filters
type FilterChain struct {
	filters []Filter
}

// NewFilterChain creates a new filter chain
func NewFilterChain(filterNames []string, encryptor *Encryptor) (*FilterChain, error) {
	if len(filterNames) == 0 {
		return &FilterChain{filters: []Filter{}}, nil
	}

	filters := make([]Filter, 0, len(filterNames))
	for _, name := range filterNames {
		var filter Filter
		switch name {
		case "encrypt":
			if encryptor == nil {
				return nil, errors.New("encrypt filter requires encryption config")
			}
			filter = NewEncryptFilter(encryptor)
		case "zip":
			filter = NewZipFilter()
		case "base64":
			filter = NewBase64Filter()
		default:
			return nil, fmt.Errorf("unknown filter: %s", name)
		}
		filters = append(filters, filter)
	}

	return &FilterChain{filters: filters}, nil
}

// Apply applies all filters in the chain
func (fc *FilterChain) Apply(input string) (string, error) {
	// Start with string input
	output := &FilterOutput{
		Data: []byte(input),
		Type: DataTypeString,
	}

	// Apply each filter
	for i, filter := range fc.filters {
		// Check input type compatibility
		inputTypes := filter.InputTypes()
		compatible := false
		for _, acceptedType := range inputTypes {
			if output.Type == acceptedType {
				compatible = true
				break
			}
		}

		if !compatible {
			return "", fmt.Errorf("filter %d (%s): incompatible input type (expected %v, got %v)",
				i, filter.Name(), inputTypes, output.Type)
		}

		// Apply filter
		result, err := filter.Apply(output)
		if err != nil {
			return "", fmt.Errorf("filter %d (%s): %w", i, filter.Name(), err)
		}

		output = result
	}

	// Auto-add base64 if final output is binary
	if output.Type == DataTypeBinary {
		base64Filter := NewBase64Filter()
		result, err := base64Filter.Apply(output)
		if err != nil {
			return "", fmt.Errorf("auto base64: %w", err)
		}
		output = result
	}

	return string(output.Data), nil
}
