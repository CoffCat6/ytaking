package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"myblog/internal/blog"
	"myblog/internal/config"
	"myblog/internal/web"
)

var (
	handler http.Handler
	once    sync.Once
)

// Initialize the application
func initApp() {
	// For Vercel, we need to adjust paths and config
	// Set default environment variables if not present
	if os.Getenv("DATA_DIR") == "" {
		// Vercel is read-only except for /tmp, but using /tmp for SQLite means data loss on restart.
		// For a real Vercel deployment, you MUST use an external database (Postgres/MySQL/Turso).
		// For demonstration/read-only purposes, we might load from a file included in the repo,
		// but writes will fail or be lost.
		// For now, let's assume read-only or ephemeral mode using /tmp
		os.Setenv("DATA_DIR", "/tmp")
	}

	cfg := config.Load()

	// WARNING: SQLite on Vercel is ephemeral! Data will disappear after function execution.
	// You should switch to a cloud database (like Turso, Neon, or Supabase) for persistence.
	// Here we just use SQLite to keep the app running, but it will be reset frequently.
	dbPath := filepath.Join(cfg.DataDir, "blog.db")
	store, err := blog.NewSQLiteStore(dbPath)
	if err != nil {
		log.Printf("Error initializing SQLite store: %v", err)
		// Fallback or panic?
	}

	// Site config (read-only from repo if possible, or /tmp)
	// If site.json is in the repo, we should read it from there.
	// Assuming site.json is in ./data in the repo
	repoDataDir := "data" 
	if _, err := os.Stat(repoDataDir); err == nil {
		// If local data dir exists (e.g. deployed with source), use it for reading if possible
		// But NewSiteStore might try to write.
	}
	
	siteStore, err := blog.NewSiteStore(filepath.Join(cfg.DataDir, "site.json"))
	if err != nil {
		log.Printf("Error initializing site store: %v", err)
	}

	// Initialize Server
	server := web.NewServer(cfg, store, siteStore)

	// Combine Public and Admin routes into one mux for Vercel (single entry point)
	mux := http.NewServeMux()
	
	// Handle static files
	// Vercel handles static files automatically from 'public' or 'static' folder if configured,
	// but our Go app also serves them. We'll let Go serve them for now or rely on Vercel routing.
	// Note: Vercel routes static files before hitting this function usually.
	
	// Register routes
	// We need to strip the prefix if Vercel forwards requests differently, 
	// but usually it forwards the full path.
	
	// Merge routes: 
	// Public routes usually at /
	// Admin routes usually at /admin
	
	publicHandler := server.PublicRoutes()
	adminHandler := server.AdminRoutes()

	mux.Handle("/admin/", adminHandler)
	mux.Handle("/", publicHandler)

	handler = mux
}

// Handler is the entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(initApp)
	handler.ServeHTTP(w, r)
}
