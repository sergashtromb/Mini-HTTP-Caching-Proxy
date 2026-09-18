package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metricts struct {
	// req mini_http_proxy_server_request_total
	httpReqTotal 	*prometheus.CounterVec
	// req mini_http_proxy_server_current_request_total
	httpReqCurrent 	prometheus.Gauge
}

func NewMetric(reg prometheus.Registerer) *Metricts {

	metrics := Metricts{}
	
	metrics.httpReqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mini_http_proxy_server",
			Subsystem: "http",
			Name: "request_total",
			Help: "All request in server",
		}, 
		[]string {"method", "path", "status"})

	metrics.httpReqCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mini_http_proxy_server",
		Subsystem: "http",
		Name: "current_request_total",
		Help: "All curr request",
	})

	reg.Register(metrics.httpReqTotal)
	reg.Register(metrics.httpReqCurrent)

	return &metrics
}