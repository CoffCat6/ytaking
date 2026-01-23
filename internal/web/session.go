package web

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

const sessionCookieName = "admin_session"

type sessionEntry struct {
	expiresAt time.Time
	csrfToken string
}

var (
	sessionMu sync.Mutex
	sessions  = map[string]sessionEntry{}
)

func createSession() (string, error) {
	// 生成随机 token 作为会话标识
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	// 生成 CSRF token
	csrfBuf := make([]byte, 32)
	if _, err := rand.Read(csrfBuf); err != nil {
		return "", err
	}
	csrfToken := base64.RawURLEncoding.EncodeToString(csrfBuf)

	sessionMu.Lock()
	sessions[token] = sessionEntry{
		expiresAt: time.Now().Add(24 * time.Hour),
		csrfToken: csrfToken,
	}
	sessionMu.Unlock()

	return token, nil
}

func getCsrfToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()

	entry, ok := sessions[cookie.Value]
	if !ok {
		return ""
	}
	// 如果会话过期，视为无效，不返回 CSRF token
	if time.Now().After(entry.expiresAt) {
		return ""
	}
	return entry.csrfToken
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()

	entry, ok := sessions[cookie.Value]
	if !ok {
		return false
	}
	if time.Now().After(entry.expiresAt) {
		delete(sessions, cookie.Value)
		return false
	}
	return true
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
