package assets

import (
	"embed"
)

// Embedded CSS from web/dist/styles.css
// Build: cp web/dist/styles.css pkg/assets/static/styles.css
//
//go:embed static/styles.css
var embeddedCSS string

// GetEmbeddedCSS returns the embedded CSS content
func GetEmbeddedCSS() string {
	return embeddedCSS
}

// Embedded icons from web/public/icons/*.svg
// Build: cp web/public/icons/*.svg pkg/assets/static/icons/
//
//go:embed static/icons/*.svg
var embeddedIcons embed.FS

// GetEmbeddedIcons returns the embedded icons filesystem
func GetEmbeddedIcons() embed.FS {
	return embeddedIcons
}
