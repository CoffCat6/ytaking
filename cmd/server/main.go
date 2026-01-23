package main

import (
	"log"
	"net/http"
	"path/filepath"

	"myblog/internal/blog"
	"myblog/internal/config"
	"myblog/internal/web"
)

func main() {
	cfg := config.Load()

	store, err := blog.NewSQLiteStore(filepath.Join(cfg.DataDir, "blog.db"))
	if err != nil {
		log.Fatal(err)
	}

	// Migration logic: if SQLite is empty but JSON exists, import data
	jsonPath := filepath.Join(cfg.DataDir, "posts.json")
	if len(store.List()) == 0 {
		if jsonStore, err := blog.NewFileStore(jsonPath); err == nil {
			log.Println("Migrating data from JSON to SQLite...")
			posts := jsonStore.List() // Lists all
			for _, p := range posts {
				// FileStore loads CreatedAt/UpdatedAt, we preserve them
				if err := store.Create(p); err != nil {
					log.Printf("Failed to migrate post %s: %v", p.Slug, err)
				}
			}
			log.Println("Migration completed.")
		}
	}

	siteStore, err := blog.NewSiteStore(filepath.Join(cfg.DataDir, "site.json"))
	if err != nil {
		log.Fatal(err)
	}

	server := web.NewServer(cfg, store, siteStore)

	publicMux := server.PublicRoutes()
	adminMux := server.AdminRoutes()

	errCh := make(chan error, 2)

	go func() {
		log.Printf("public listening on %s", cfg.PublicAddr)
		errCh <- http.ListenAndServe(cfg.PublicAddr, publicMux)
	}()

	go func() {
		log.Printf("admin listening on %s", cfg.AdminAddr)
		errCh <- http.ListenAndServe(cfg.AdminAddr, adminMux)
	}()

	log.Fatal(<-errCh)
}
