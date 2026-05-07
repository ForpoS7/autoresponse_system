package middleware

import (
	"hh_autoapply_service/pkg/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Metrics is a gorilla/mux-compatible middleware that records Prometheus HTTP metrics.
// It uses the route template (e.g. /api/autoapply/{id}) as the path label to avoid
// high-cardinality from raw URL paths.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// gorilla/mux route template keeps label cardinality low
		path := r.URL.Path
		if route := mux.CurrentRoute(r); route != nil {
			if tmpl, err := route.GetPathTemplate(); err == nil {
				path = tmpl
			}
		}

		status := strconv.Itoa(rw.statusCode)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}
