package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"myblog/internal/blog"
	"myblog/internal/config"
	"myblog/internal/web"
)

func main() {
	// 1. Load config and store
	cfg := config.Load()

	// 强制设置 SiteBaseURL 为环境变量或默认值，但在本地生成时可能需要根据部署目标调整
	// 这里假设用户会在运行生成器前设置好环境变量，或者我们在这里可以覆盖
	// 如果发布到 GitHub Pages，通常子路径是 /repo-name/
	// 我们暂时保持 cfg 读取的逻辑，用户需自行配置 SITE_BASE_URL

	dbPath := filepath.Join(cfg.DataDir, "blog.db")
	store, err := blog.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}

	sitePath := filepath.Join(cfg.DataDir, "site.json")
	siteStore, err := blog.NewSiteStore(sitePath)
	if err != nil {
		log.Fatalf("Failed to open site store: %v", err)
	}

	// 2. Initialize Server
	srv := web.NewServer(cfg, store, siteStore)

	// 3. Prepare output directory
	outputDir := "dist"
	if err := os.RemoveAll(outputDir); err != nil {
		log.Printf("Warning: failed to clean dist dir: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create dist dir: %v", err)
	}

	// 4. Define routes to crawl
	routes := []string{"/"} // Index

	// Add all posts
	posts := store.ListPublished()
	for _, p := range posts {
		routes = append(routes, "/posts/"+p.Slug)
	}

	// Add archive page
	routes = append(routes, "/archive")

	// 5. Generate pages
	mux := srv.PublicRoutes()

	for _, route := range routes {
		fmt.Printf("Generating %s...\n", route)
		req := httptest.NewRequest("GET", route, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			log.Printf("Error generating %s: status %d", route, resp.StatusCode)
			continue
		}

		// Determine output file path
		// / -> index.html
		// /posts/slug -> posts/slug/index.html (for clean URLs) OR posts/slug.html
		// Let's use clean URLs: posts/slug/index.html
		relPath := route
		if relPath == "/" {
			relPath = "index.html"
		} else {
			relPath = relPath + "/index.html"
		}
		// Remove leading slash for filepath.Join
		relPath = filepath.Clean(relPath) // trims leading / on windows sometimes? handled better manually
		if relPath[0] == '\\' || relPath[0] == '/' {
			relPath = relPath[1:]
		}

		outPath := filepath.Join(outputDir, relPath)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			log.Fatalf("Failed to create dir for %s: %v", outPath, err)
		}

		f, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("Failed to create file %s: %v", outPath, err)
		}

		body, _ := io.ReadAll(resp.Body)
		f.Write(body)
		f.Close()
		resp.Body.Close()
	}

	// 6. Copy static assets
	fmt.Println("Copying static assets...")
	copyDir("static", filepath.Join(outputDir, "static"))
	copyDir("uploads", filepath.Join(outputDir, "uploads"))

	fmt.Println("Done! Static site generated in 'dist' directory.")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		destFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		return err
	})
}
