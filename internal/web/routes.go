package web

import "net/http"

func (s *Server) PublicRoutes() http.Handler {
	mux := http.NewServeMux()

	// 静态资源（CSS/图片等）
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// 页面
	mux.HandleFunc("/", s.Index)
	mux.HandleFunc("/posts", s.PostsList)
	mux.HandleFunc("/posts/", s.PostDetail)
	mux.HandleFunc("/archive", s.ArchivePage)
	mux.HandleFunc("/search", s.SearchPage)
	mux.HandleFunc("/feed", s.RSS)
	mux.HandleFunc("/sitemap.xml", s.Sitemap)

	return mux
}

func (s *Server) AdminRoutes() http.Handler {
	mux := http.NewServeMux()
	// 静态资源（CSS/图片等）
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
	mux.HandleFunc("/admin/login", s.AdminLogin)
	mux.HandleFunc("/admin/logout", s.AdminLogout)
	mux.HandleFunc("/admin/posts", s.AdminPosts)
	mux.HandleFunc("/admin/posts/new", s.AdminPostNew)
	mux.HandleFunc("/admin/posts/edit", s.AdminPostEdit)
	mux.HandleFunc("/admin/posts/delete", s.AdminPostDelete)
	mux.HandleFunc("/admin/settings", s.AdminSettings)
	mux.HandleFunc("/admin/upload", s.AdminUpload)
	mux.HandleFunc("/admin/subscribers", s.AdminSubscribers)
	mux.HandleFunc("/admin/subscribers/add", s.AdminSubscribe)
	mux.HandleFunc("/admin/subscribers/delete", s.AdminUnsubscribe)
	return adminAuth(mux)
}
