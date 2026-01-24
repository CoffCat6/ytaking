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

	// 合并公开路由和管理路由到同一个服务器
	// Fly.io 只支持单端口，管理后台通过 /admin/* 路径访问
	mux := http.NewServeMux()

	// 注册公开路由
	publicMux := server.PublicRoutes()
	mux.Handle("/", publicMux)

	// 注册管理路由 (已有 /admin/ 前缀)
	adminMux := server.AdminRoutes()
	mux.Handle("/admin/", adminMux)

	log.Printf("Server listening on %s (public + admin)", cfg.PublicAddr)
	log.Fatal(http.ListenAndServe(cfg.PublicAddr, mux))
}
