package metrics

import (
	"fmt"
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"time"
)

type Client interface {
	SendSummaryDuration(metricName string, attributes map[string]interface{}, duration time.Duration)
	SendSummaryValue(metricName string, attributes map[string]interface{}, value float64)
}

type wrappedMetricAggregator struct {
	metricAggregator *telemetry.MetricAggregator
}

func (m *wrappedMetricAggregator) SendSummaryDuration(metricName string, attributes map[string]interface{}, duration time.Duration) {
	m.metricAggregator.Summary(metricName, attributes).RecordDuration(duration)
}

func (m *wrappedMetricAggregator) SendSummaryValue(metricName string, attributes map[string]interface{}, value float64) {
	m.metricAggregator.Summary(metricName, attributes).Record(value)
}

type noopMetricAggregator struct{}

func (*noopMetricAggregator) SendSummaryDuration(metricName string, attributes map[string]interface{}, duration time.Duration) {
}

func (*noopMetricAggregator) SendSummaryValue(metricName string, attributes map[string]interface{}, value float64) {
}

// Return a new metrics client. If sendMetrics is true and a valid Logs API URL is supplied, a real client is returned.
// If sendMetrics is true or no Metrics API URL mapping exists for the supplied Logs API URL, a noop client is returned.
func NewClient(nrClientConfig config.NRClientConfig) (Client, error) {
	metricReportingEnabled := nrClientConfig.SendMetrics
	logsApiUrl := nrClientConfig.Endpoint
	metricsApiUrl, ok := logsToMetricsUrlMapping[logsApiUrl]
	if metricReportingEnabled && !ok {
		return newNoopMetricAggregator(), fmt.Errorf("no Metrics API URL can be inferred out ot the Logs API URL %s", logsApiUrl)
	}

	if metricReportingEnabled {
		licenseKey := nrClientConfig.GetNewRelicKey()
		return newWrappedMetricAggregator(metricsApiUrl, licenseKey)
	}
	return newNoopMetricAggregator(), nil
}

func newWrappedMetricAggregator(metricsApiUrl string, licenseKey string) (*wrappedMetricAggregator, error) {
	metricHarvester, err := telemetry.NewHarvester(
		telemetry.ConfigMetricsURLOverride(metricsApiUrl),
		telemetry.ConfigAPIKey(licenseKey))
	if err != nil {
		return nil, fmt.Errorf("can't create metrics harvester: %v", err)
	}

	return &wrappedMetricAggregator{
		metricAggregator: metricHarvester.MetricAggregator(),
	}, nil
}

func newNoopMetricAggregator() *noopMetricAggregator {
	return &noopMetricAggregator{}
}
