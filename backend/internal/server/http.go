// Package server contains shared HTTP server infrastructure.
package server

import "net/http"

// WithCORS adds credentialed CORS support for one configured browser origin.
func WithCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestOrigin := request.Header.Get("Origin")
		if requestOrigin == origin {
			writer.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
			writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, PATCH, POST, OPTIONS")
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
