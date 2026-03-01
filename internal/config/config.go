package config

import (
	"net"
	"os"
	"strings"
)

type Config struct {
	PublicAddr   string
	AdminAddr    string
	AdminUser    string
	AdminPass    string
	SiteBaseURL  string
	AdminBaseURL string
	DataDir      string
}

func Load() *Config {
	publicAddr := getEnv("PUBLIC_ADDR", ":8084")
	adminAddr := getEnv("ADMIN_ADDR", ":8085")
	siteBaseURL := strings.TrimRight(getEnv("SITE_BASE_URL", ""), "/")
	if siteBaseURL == "" {
		siteBaseURL = baseURLFromAddr(publicAddr)
	}
	adminBaseURL := strings.TrimRight(getEnv("ADMIN_BASE_URL", ""), "/")
	if adminBaseURL == "" {
		if siteBaseURL != "" {
			adminBaseURL = siteBaseURL + "/admin"
		} else {
			adminBaseURL = baseURLFromAddr(adminAddr)
		}
	}

	return &Config{
		PublicAddr:   publicAddr,
		AdminAddr:    adminAddr,
		AdminUser:    getEnv("ADMIN_USER", "admin"),
		AdminPass:    getEnv("ADMIN_PASS", "admin"),
		SiteBaseURL:  siteBaseURL,
		AdminBaseURL: adminBaseURL,
		DataDir:      getEnv("DATA_DIR", "data"),
	}
}

func baseURLFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}

	host := ""
	port := ""
	if strings.HasPrefix(addr, ":") {
		host = "localhost"
		port = strings.TrimPrefix(addr, ":")
	} else {
		if h, p, err := net.SplitHostPort(addr); err == nil {
			host = h
			port = p
		} else {
			host = addr
		}
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	if port != "" {
		return "http://" + host + ":" + port
	}
	return "http://" + host
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
