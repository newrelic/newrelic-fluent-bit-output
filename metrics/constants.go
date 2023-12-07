package metrics

// Metrics sent from this plugin
const (
	PackagingTime        = "logs.fb.packaging.time"
	PayloadSendTime      = "logs.fb.payload.send.time"
	TotalSendTime        = "logs.fb.total.send.time"
	PayloadCountPerChunk = "logs.fb.payload.count"
	PayloadSize          = "logs.fb.payload.size"
)

// API URLs
const (
	metricsUsProdUrl  = "https://metric-api.newrelic.com/metric/v1"
	metricsEuProdUrl  = "https://metric-api.eu.newrelic.com/metric/v1"
	metricsStagingUrl = "https://staging-metric-api.newrelic.com/metric/v1"
	logsUsProdUrl     = "https://log-api.newrelic.com/log/v1"
	logsEuProdUrl     = "https://log-api.eu.newrelic.com/log/v1"
	logsStagingUrl    = "https://staging-log-api.newrelic.com/log/v1"
)

// Maps the Metrics API URL that corresponds to the same environment as the provided
// Logs API URL. It returns nil if an incorrect Logs API URL was provided.
var logsToMetricsUrlMapping = map[string]string{
	logsUsProdUrl:  metricsUsProdUrl,
	logsEuProdUrl:  metricsEuProdUrl,
	logsStagingUrl: metricsStagingUrl,
}
