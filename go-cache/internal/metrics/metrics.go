package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/status-im/proxy-common/httpclient"
)

// AlchemyHTTPMetrics provides HTTP metrics for Alchemy API requests
type AlchemyHTTPMetrics struct {
	requests *prometheus.CounterVec
	retries  prometheus.Counter
}

var (
	alchemyHTTPRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nft_proxy_alchemy_requests_total",
			Help: "Total number of Alchemy API requests",
		},
		[]string{"status"}, // status: success, error, rate_limited
	)

	alchemyHTTPRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nft_proxy_alchemy_retries_total",
			Help: "Total number of Alchemy API request retries",
		},
	)
)

// NewAlchemyHTTPMetrics creates a new HTTP metrics recorder for Alchemy
func NewAlchemyHTTPMetrics() httpclient.IHttpStatusHandler {
	return &AlchemyHTTPMetrics{
		requests: alchemyHTTPRequests,
		retries:  alchemyHTTPRetries,
	}
}

// OnRequest records HTTP request status
func (m *AlchemyHTTPMetrics) OnRequest(status string) {
	m.requests.WithLabelValues(status).Inc()
}

// OnRetry records HTTP retry event
func (m *AlchemyHTTPMetrics) OnRetry() {
	m.retries.Inc()
}
