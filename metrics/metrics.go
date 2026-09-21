package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	Namespace = "mini_http_proxy_server"
	Subsystem = "http"
)

type Metricts struct {
	// req mini_http_proxy_server_request_total
	httpReqTotal 	*prometheus.CounterVec
	// req mini_http_proxy_server_current_request_total
	httpReqCurrent 	prometheus.Gauge
	// req mini_http_proxy_server_time_deal
	httpReqTimeDeal prometheus.Histogram
}

func NewMetric(reg prometheus.Registerer) *Metricts {

	metrics := Metricts{}
	
	metrics.httpReqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name: "request_total",
			Help: "All request in server",
		}, 
		[]string {"method", "path", "status"})

	metrics.httpReqCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name: "current_request_total",
		Help: "All curr request",
	})

	metrics.httpReqTimeDeal = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name: "time_deal",
		Help: "Request execution time",
	})

	reg.Register(metrics.httpReqTotal)
	reg.Register(metrics.httpReqCurrent)
	reg.Register(metrics.httpReqTimeDeal)

	return &metrics
}