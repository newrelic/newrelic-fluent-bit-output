package metrics

import (
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type MetricsClient struct {
	metricHarvester *telemetry.Harvester
}

// Metrics sent from this plugin
const (
	PackagingTime        = "logs.fb.packaging.time"
	PayloadSendTime      = "logs.fb.payload.send.time"
	TotalSendTime        = "logs.fb.total.send.time"
	PayloadCountPerChunk = "logs.fb.payload.count"
	PayloadSize          = "logs.fb.payload.size"
)

// Metrics API URL to be used depending on the environment where logs are being sent
const (
	metricsUsProdUrl  = "https://metric-api.newrelic.com/metric/v1"
	metricsEuProdUrl  = "https://metric-api.eu.newrelic.com/metric/v1"
	metricsStagingUrl = "https://staging-metric-api.newrelic.com/metric/v1"
)

func NewMetricsHarvester(cfg config.PluginConfig) *MetricsClient {
	if cfg.NRClientConfig.SendMetrics {
		metricHarvester, err := telemetry.NewHarvester(
			telemetry.ConfigMetricsURLOverride(getCorrespondingMetricsUrl(cfg.NRClientConfig.Endpoint)),
			telemetry.ConfigAPIKey(cfg.NRClientConfig.GetNewRelicKey()))
		if err != nil {
			log.WithField("error", err).Error("Error creating metric harvester")
		}
		return &MetricsClient{
			metricHarvester: metricHarvester,
		}
	}
	return &MetricsClient{}
}

func getCorrespondingMetricsUrl(logsUrl string) string {
	if strings.Contains(logsUrl, "staging") {
		return metricsStagingUrl
	} else if strings.Contains(logsUrl, "eu") {
		return metricsEuProdUrl
	} else {
		return metricsUsProdUrl
	}
}

func (m *MetricsClient) SendSummaryDuration(metricName string, attributes map[string]interface{}, duration time.Duration) {
	if m.metricHarvester != nil {
		m.metricHarvester.MetricAggregator().Summary(metricName, attributes).RecordDuration(duration)
	}
}

func (m MetricsClient) SendSummaryValue(metricName string, attributes map[string]interface{}, value float64) {
	if m.metricHarvester != nil {
		m.metricHarvester.MetricAggregator().Summary(metricName, attributes).Record(value)
	}
}
