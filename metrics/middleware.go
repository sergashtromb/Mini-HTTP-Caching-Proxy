package metrics

import (
	"bufio"
	"net"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type statusRecorder struct {
	status int
	wroteHeader bool
	http.ResponseWriter
}

func (sc *statusRecorder) WriteHeader(code int) {

	if !sc.wroteHeader {
		sc.status = code
		sc.wroteHeader = true
	}
	sc.ResponseWriter.WriteHeader(code)
}

func (sc *statusRecorder) Write(b []byte) (int, error) {
	if !sc.wroteHeader {
		sc.wroteHeader = true
	}
	return sc.ResponseWriter.Write(b)
}

func (sc *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := sc.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	
	return hijack.Hijack()
}

func (sc *statusRecorder) Flush() {
	if f, ok := sc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (sc *statusRecorder) Unwrap() http.ResponseWriter {
	return sc.ResponseWriter
}

type Middleware struct {
	metric *Metricts
}

func NewMiddleware(m *Metricts) *Middleware {
	return &Middleware{
		metric: m,
	}
}

func (mi *Middleware) MetricMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		mi.metric.httpReqCurrent.Inc()
		defer mi.metric.httpReqCurrent.Dec()

		record := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(record, r)

		labels := prometheus.Labels{
			"method": r.Method,
			"path": r.Pattern,
			"status": strconv.Itoa(record.status),
		}

		mi.metric.httpReqTotal.With(labels).Inc()

	})
}