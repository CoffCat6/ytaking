package web

import (
	"html/template"
	"myblog/internal/blog"
	"myblog/internal/config"
)

type Server struct {
	Config        *config.Config
	Store         blog.Store
	SiteStore     *blog.SiteStore
	TemplateCache map[string]*template.Template
}

func NewServer(cfg *config.Config, store blog.Store, siteStore *blog.SiteStore) *Server {
	return &Server{
		Config:        cfg,
		Store:         store,
		SiteStore:     siteStore,
		TemplateCache: make(map[string]*template.Template),
	}
}
