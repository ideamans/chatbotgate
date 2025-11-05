package assets

import (
	"embed"
)

// Embedded CSS from web/
// Build: cd web && yarn build
// The build automatically outputs to pkg/middleware/assets/static/
//
//go:embed static/main.css
var embeddedCSS string

// GetEmbeddedCSS returns the embedded CSS content
func GetEmbeddedCSS() string {
	return embeddedCSS
}

// Embedded Dify CSS for iframe optimizations
//
//go:embed static/dify.css
var embeddedDifyCSS string

// GetEmbeddedDifyCSS returns the embedded Dify CSS content
func GetEmbeddedDifyCSS() string {
	return embeddedDifyCSS
}

// Embedded icons from web/public/icons/
// Build: cd web && yarn build (automatically copies icons)
//
//go:embed static/icons/*.svg
var embeddedIcons embed.FS

// GetEmbeddedIcons returns the embedded icons filesystem
func GetEmbeddedIcons() embed.FS {
	return embeddedIcons
}
