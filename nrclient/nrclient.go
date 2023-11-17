package nrclient

import (
	"bytes"
	"fmt"
	"github.com/newrelic/newrelic-fluent-bit-output/metrics"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/newrelic/newrelic-fluent-bit-output/record"
	log "github.com/sirupsen/logrus"

	"github.com/newrelic/newrelic-fluent-bit-output/config"
)

var retryableCodesSet = map[int]struct{}{
	408: {},
	429: {},
	500: {},
	502: {},
	503: {},
	504: {},
	599: {},
}

type NRClient struct {
	client        *http.Client
	config        config.NRClientConfig
	metricsClient *metrics.MetricsClient
}

func NewNRClient(cfg config.NRClientConfig, proxyCfg config.ProxyConfig, metricHarvester *metrics.MetricsClient) (*NRClient, error) {
	httpTransport, err := buildHttpTransport(proxyCfg, cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("building HTTP transport: %v", err)
	}

	nrClient := &NRClient{
		client: &http.Client{
			Transport: httpTransport,
			Timeout:   time.Second * time.Duration(cfg.TimeoutSeconds),
		},
		config:        cfg,
		metricsClient: metricHarvester,
	}

	return nrClient, nil
}

func (nrClient *NRClient) Send(logRecords []record.LogRecord) (retry bool, err error) {
	packaging_start := time.Now()
	payloads, err := record.PackageRecords(logRecords)
	packaging_time := time.Since(packaging_start)
	nrClient.metricsClient.SendSummaryDuration(metrics.PackagingTime, nil, packaging_time)
	if err != nil {
		log.WithField("error", err).Error("Error packaging request")
		return false, err
	}

	payload_send_start := time.Now()
	for _, payload := range payloads {
		send_start := time.Now()
		statusCode, err := nrClient.sendPacket(payload)
		send_time := time.Since(send_start)

		// If we receive any error, we'll always retry sending the logs...
		if err != nil {
			return true, err
		}

		nrClient.metricsClient.SendSummaryDuration(
			metrics.PayloadSendTime,
			map[string]interface{}{
				"statusCode": statusCode,
			},
			send_time)

		// ...unless we receive an explicit non-2XX HTTP status code from the server that tells us otherwise
		if statusCode/100 != 2 {
			return isStatusCodeRetryable(statusCode), fmt.Errorf("received non-2XX HTTP status code: %d", statusCode)
		}
	}
	payload_send_time := time.Since(payload_send_start)
	nrClient.metricsClient.SendSummaryDuration(metrics.TotalSendTime, nil, payload_send_time)
	nrClient.metricsClient.SendSummaryValue(metrics.PayloadCountPerChunk, nil, float64(len(payloads)))

	return false, nil
}

func (nrClient *NRClient) sendPacket(buffer *bytes.Buffer) (status int, err error) {
	req, err := http.NewRequest("POST", nrClient.config.Endpoint, buffer)
	if err != nil {
		return 0, err
	}
	if nrClient.config.UseApiKey {
		req.Header.Add("X-Insert-Key", nrClient.config.ApiKey)
	} else {
		req.Header.Add("X-License-Key", nrClient.config.LicenseKey)
	}
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	resp, err := nrClient.client.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	return resp.StatusCode, nil
}

func isStatusCodeRetryable(statusCode int) bool {
	_, ok := retryableCodesSet[statusCode]
	return ok
}
