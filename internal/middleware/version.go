package middleware

import (
	"context"
	"net/http"
	"strings"
)

type versionKey string

const VersionKey versionKey = "apiVersion"

func APIVersion(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := extractVersion(r)
		ctx := context.WithValue(r.Context(), VersionKey, version)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractVersion(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return "v1"
	}

	if strings.Contains(accept, "version=") {
		parts := strings.Split(accept, "version=")
		if len(parts) > 1 {
			version := strings.TrimSpace(parts[1])
			if strings.HasPrefix(version, "1") || version == "v1" {
				return "v1"
			}
			return version
		}
	}

	if strings.Contains(accept, "application/vnd.hourglass+json") {
		return "v1"
	}

	return "v1"
}

func GetAPIVersion(r *http.Request) string {
	if v := r.Context().Value(VersionKey); v != nil {
		return v.(string)
	}
	return "v1"
}
