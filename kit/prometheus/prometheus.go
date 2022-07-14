package prometheus

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})
	statusCodes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "status_code",
		Help: "Counter for each status code",
	}, []string{"status_code"})
)

type promResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newPrometheusResponseWriter(w http.ResponseWriter) *promResponseWriter {
	return &promResponseWriter{w, http.StatusOK}
}

func (lrw *promResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Middleware implements mux.MiddlewareFunc.
// It will add duration to the prometheus metrics.
// This way we know the latency that each request takes.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		lrw := newPrometheusResponseWriter(w)
		next.ServeHTTP(lrw, r)
		timer.ObserveDuration()
		statusCodes.WithLabelValues(strconv.Itoa(lrw.statusCode)).Inc()
	})
}
