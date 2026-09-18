package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStaticHandlerEmbedded verifies the production path: with dir=""
// (the default), staticHandler serves the embedded build, and every file
// build.mjs is expected to produce is actually reachable.
func TestStaticHandlerEmbedded(t *testing.T) {
	h, err := staticHandler("")
	if err != nil {
		t.Fatalf("staticHandler(\"\"): %v", err)
	}

	// "/" not "/index.html": http.FileServer 301-redirects the literal
	// /index.html path to / (its directory-index canonicalization), so
	// requesting it directly would test the redirect, not the content.
	for _, path := range []string{"/", "/style.css", "/bundle.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

// TestStaticHandlerEmbeddedIndexContent verifies the embedded index.html is
// the real GUI shell (has the #app mount point and loads bundle.js), not an
// accidental empty/placeholder file.
func TestStaticHandlerEmbeddedIndexContent(t *testing.T) {
	h, err := staticHandler("")
	if err != nil {
		t.Fatalf("staticHandler(\"\"): %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	if !strings.Contains(html, `id="app"`) {
		t.Error("index.html missing #app mount point")
	}
	if !strings.Contains(html, "bundle.js") {
		t.Error("index.html does not load bundle.js")
	}
}

// TestStaticHandlerDevMode verifies the --static-dir path: staticHandler
// serves from disk when a directory is given, independent of the embedded
// build.
func TestStaticHandlerDevMode(t *testing.T) {
	dir := t.TempDir()
	const content = "dev-mode marker content"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	h, err := staticHandler(dir)
	if err != nil {
		t.Fatalf("staticHandler(%q): %v", dir, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != content {
		t.Errorf("body = %q, want %q (dev-mode must serve from disk, not the embedded build)", got, content)
	}
}
