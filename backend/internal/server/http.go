// Package server contains HTTP infrastructure shared by the API entrypoints.
package server

import "net/http"

// WithCORS restricts credentialed browser access to the configured frontend origin.
// The allow-list is intentionally limited to the methods and headers used by
// the API: DELETE, GET, POST, and PATCH for requests, OPTIONS for browser preflight,
// and Content-Type and Authorization for JSON, uploads, and bearer tokens.
// Credentials are enabled because the frontend sends the HttpOnly refresh
// cookie, so the exact origin must be echoed instead of using the wildcard.
func WithCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestOrigin := request.Header.Get("Origin")
		if requestOrigin == origin {
			writer.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
			writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			writer.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, PATCH, POST, OPTIONS")
			writer.Header().Set("Vary", "Origin")
		}
		if request.Method == http.MethodOptions {
			if requestOrigin != origin {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
