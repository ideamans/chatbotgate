package assets

import (
	"strings"
	"testing"
)

// TestGetEmbeddedCSS tests that embedded main CSS is available
func TestGetEmbeddedCSS(t *testing.T) {
	css := GetEmbeddedCSS()

	if len(css) == 0 {
		t.Error("GetEmbeddedCSS() returned empty string")
	}

	// Basic check that it looks like CSS
	if !strings.Contains(css, "{") || !strings.Contains(css, "}") {
		t.Error("GetEmbeddedCSS() does not appear to contain valid CSS")
	}
}

// TestGetEmbeddedDifyCSS tests that embedded Dify CSS is available
func TestGetEmbeddedDifyCSS(t *testing.T) {
	css := GetEmbeddedDifyCSS()

	if len(css) == 0 {
		t.Error("GetEmbeddedDifyCSS() returned empty string")
	}

	// Basic check that it looks like CSS
	if !strings.Contains(css, "{") || !strings.Contains(css, "}") {
		t.Error("GetEmbeddedDifyCSS() does not appear to contain valid CSS")
	}
}

// TestGetEmbeddedIcons tests that embedded icons filesystem is available
func TestGetEmbeddedIcons(t *testing.T) {
	fs := GetEmbeddedIcons()

	// Try to read the icons directory
	entries, err := fs.ReadDir("static/icons")
	if err != nil {
		t.Fatalf("Failed to read icons directory: %v", err)
	}

	// Should have at least some icon files
	if len(entries) == 0 {
		t.Error("GetEmbeddedIcons() returned empty directory")
	}

	// Check that all entries are .svg files
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".svg") {
			t.Errorf("Found non-SVG file in icons directory: %s", entry.Name())
		}
	}
}

// TestDifyCSSContent tests that Dify CSS contains expected iframe optimizations
func TestDifyCSSContent(t *testing.T) {
	css := GetEmbeddedDifyCSS()

	// Check for key Dify CSS features based on dify.css content
	expectedPatterns := []string{
		"body",         // Should style body element
		"background",   // Should have background styling
		"in-iframe",    // Should have in-iframe class references
		"@media",       // Should have media queries for responsive design
		"auth-card",    // Should style auth-card element
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(css, pattern) {
			t.Errorf("Dify CSS should contain '%s' for iframe optimizations", pattern)
		}
	}
}

// TestMainCSSContent tests that main CSS contains expected content
func TestMainCSSContent(t *testing.T) {
	css := GetEmbeddedCSS()

	// Main CSS should be significantly larger than Dify CSS
	difyCss := GetEmbeddedDifyCSS()
	if len(css) < len(difyCss) {
		t.Error("Main CSS should be larger than Dify CSS")
	}

	// Main CSS should contain water.css or similar styling
	if !strings.Contains(css, "body") {
		t.Error("Main CSS should contain body styling")
	}
}

// TestCSSFilesAreDifferent tests that main.css and dify.css are different files
func TestCSSFilesAreDifferent(t *testing.T) {
	mainCSS := GetEmbeddedCSS()
	difyCSS := GetEmbeddedDifyCSS()

	if mainCSS == difyCSS {
		t.Error("Main CSS and Dify CSS should be different files")
	}

	if len(mainCSS) == len(difyCSS) {
		t.Error("Main CSS and Dify CSS should have different lengths")
	}
}
