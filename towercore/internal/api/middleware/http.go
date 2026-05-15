package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"towercore/internal/infrastructure/logger"
)

const requestIDHeader = "X-Request-Id"

func Chain(h http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get(requestIDHeader)
			if reqID == "" {
				reqID = generateRequestID()
			}
			w.Header().Set(requestIDHeader, reqID)
			next.ServeHTTP(w, r)
		})
	}
}

func AccessLog(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			log.Infof(
				"http request method=%s path=%s status=%d duration_ms=%d",
				r.Method,
				r.URL.Path,
				rw.status,
				time.Since(start).Milliseconds(),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "0000000000000000"
	}
	dst := make([]byte, 16)
	hex.Encode(dst, b)
	return string(dst)
}
