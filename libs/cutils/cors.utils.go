package cutils

import (
	"os"
	"strings"
)

// CORSAllowedOrigins reads the comma-separated CORS_ALLOWED_ORIGINS env var.
// Falls back to "*" so local development keeps working; production must set it.
// Services pair this with AllowCredentials=false, so a wildcard never reflects
// arbitrary origins with credentials.
func CORSAllowedOrigins() []string {
	origins := []string{}
	for _, o := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}
