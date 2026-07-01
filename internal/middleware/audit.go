package middleware

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/JustVugg/gonk/internal/auth"
)

func Audit(routeName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &auditResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		authCtx := auth.GetAuthContext(r)
		clientID := authCtx.ClientID
		if clientID == "" {
			clientID = authCtx.UserID
		}
		if clientID == "" {
			clientID = "anonymous"
		}

		log.Printf(
			"audit route=%s method=%s path=%s status=%d duration_ms=%d client_ip=%s identity_type=%s identity=%s roles=%v scopes=%v",
			routeName,
			r.Method,
			r.URL.RequestURI(),
			wrapped.statusCode,
			time.Since(start).Milliseconds(),
			clientIP(r),
			authCtx.IdentityType,
			clientID,
			authCtx.Roles,
			authCtx.Scopes,
		)
	})
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *auditResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *auditResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *auditResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
