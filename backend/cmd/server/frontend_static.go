package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func isAllowedOrigin(origin string) bool {
	switch origin {
	case "http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://localhost:5174",
		"http://localhost:3000",
		"https://localhost:5173":
		return true
	}

	return strings.HasPrefix(origin, "chrome-extension://")
}

func detectFrontendWebRoot() string {
	candidates := []string{"web"}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "web"))
	}

	for _, candidate := range candidates {
		if hasFrontendIndex(candidate) {
			return candidate
		}
	}

	return ""
}

func hasFrontendIndex(webRoot string) bool {
	info, err := os.Stat(filepath.Join(webRoot, "index.html"))
	return err == nil && !info.IsDir()
}

func configureFrontendStatic(r *gin.Engine, webRoot string) {
	indexPath := filepath.Join(webRoot, "index.html")

	log.Printf("Serving frontend assets from %s", webRoot)

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"message": "resource not found"})
			return
		}

		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		if assetPath, ok := resolveFrontendAssetPath(webRoot, c.Request.URL.Path); ok {
			c.File(assetPath)
			return
		}

		c.File(indexPath)
	})
}

func resolveFrontendAssetPath(webRoot, requestPath string) (string, bool) {
	cleanedPath := path.Clean("/" + requestPath)
	if cleanedPath == "/" {
		return "", false
	}

	relativePath := strings.TrimPrefix(cleanedPath, "/")
	candidatePath := filepath.Join(webRoot, filepath.FromSlash(relativePath))
	relativeCandidate, err := filepath.Rel(webRoot, candidatePath)
	if err != nil || strings.HasPrefix(relativeCandidate, "..") {
		return "", false
	}

	info, err := os.Stat(candidatePath)
	if err != nil || info.IsDir() {
		return "", false
	}

	return candidatePath, true
}
