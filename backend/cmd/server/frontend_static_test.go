package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsAllowedOrigin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "vite localhost", origin: "http://localhost:5173", want: true},
		{name: "chrome extension", origin: "chrome-extension://abcdefghijklmnop", want: true},
		{name: "unsupported https origin", origin: "https://example.com", want: false},
		{name: "empty", origin: "", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isAllowedOrigin(tc.origin); got != tc.want {
				t.Fatalf("isAllowedOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}

func TestConfigureFrontendStatic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	webRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(webRoot, "index.html"), "<html><body>frontend shell</body></html>")
	mustWriteFile(t, filepath.Join(webRoot, "assets", "app.js"), "console.log('frontend asset')")

	router := gin.New()
	configureFrontendStatic(router, webRoot)

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
	}{
		{name: "spa route falls back to index", method: http.MethodGet, target: "/login", wantStatus: http.StatusOK, wantBody: "frontend shell"},
		{name: "asset served directly", method: http.MethodGet, target: "/assets/app.js", wantStatus: http.StatusOK, wantBody: "frontend asset"},
		{name: "api path keeps 404", method: http.MethodGet, target: "/api/v1/missing", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "post non api path stays 404", method: http.MethodPost, target: "/login", wantStatus: http.StatusNotFound, wantBody: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && !contains(recorder.Body.String(), tc.wantBody) {
				t.Fatalf("body %q does not contain %q", recorder.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestResolveFrontendAssetPathRejectsTraversal(t *testing.T) {
	t.Parallel()

	webRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(webRoot, "index.html"), "ok")

	if _, ok := resolveFrontendAssetPath(webRoot, "/../secret.txt"); ok {
		t.Fatalf("expected traversal path to be rejected")
	}
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || strings.Contains(haystack, needle)
}
