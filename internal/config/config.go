package config

import (
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
	return &Config{
		PublicAddr:   getEnv("PUBLIC_ADDR", ":8080"),
		AdminAddr:    getEnv("ADMIN_ADDR", ":8081"),
		AdminUser:    getEnv("ADMIN_USER", "admin"),
		AdminPass:    getEnv("ADMIN_Pass", "admin"),
		SiteBaseURL:  strings.TrimRight(getEnv("SITE_BASE_URL", "http://localhost:8080"), "/"),
		AdminBaseURL: strings.TrimRight(getEnv("ADMIN_BASE_URL", "http://localhost:8081"), "/"),
		DataDir:      getEnv("DATA_DIR", "data"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
