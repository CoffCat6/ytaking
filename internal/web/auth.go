package web

import (
	"net/http"
)

func adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 放行静态资源、登录与退出
		if r.URL.Path == "/admin/login" || r.URL.Path == "/admin/logout" || len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/static" {
			next.ServeHTTP(w, r)
			return
		}

		// 通过会话校验后台权限
		if !isAuthenticated(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		// CSRF Check for state-changing requests
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			token := r.FormValue("csrf_token")
			validToken := getCsrfToken(r)
			if validToken == "" || token != validToken {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
