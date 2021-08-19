package nrclient

import (
	"bytes"
	"fmt"
	"github.com/newrelic/newrelic-fluent-bit-output/record"
	"io"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"

	"github.com/newrelic/newrelic-fluent-bit-output/config"
)

type NRClient struct {
	client *http.Client
	config config.NRClientConfig
}

const(
	nonRetriableConnectionError = -1
	retriableConnectionError = -2
)

func NewNRClient(cfg config.NRClientConfig, proxyCfg config.ProxyConfig) (*NRClient, error) {
	httpTransport, err := buildHttpTransport(proxyCfg, cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("building HTTP transport: %v", err)
	}

	nrClient := &NRClient{
		client: &http.Client{
			Transport: httpTransport,
			Timeout:   5 * time.Second,
		},
		config: cfg,
	}

	return nrClient, nil
}

func (nrClient *NRClient) Send(logRecords []record.LogRecord) (int, error) {
	payloads, err := record.PackageRecords(logRecords)
	if err != nil {
		return nonRetriableConnectionError, err
	}

	for _, payload := range payloads {
		statusCode, err := nrClient.sendPacket(payload)
		if err != nil || !isSuccesful(statusCode) {
			return statusCode, err
		}
	}

	return http.StatusAccepted, nil
}

func (nrClient *NRClient) sendPacket(buffer *bytes.Buffer) (status int, err error) {
	req, err := http.NewRequest("POST", nrClient.config.Endpoint, buffer)
	if err != nil {
		return nonRetriableConnectionError, err
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
		log.WithField("error", err).Error("Error making HTTP request")
		return retriableConnectionError, err
	} else if !isSuccesful(resp.StatusCode) {
		log.WithField("status_code", resp.StatusCode).Error("HTTP request made but got error reponse")
		return resp.StatusCode, nil
	}
	defer resp.Body.Close()
	defer func() {
		// WE READ THE BODY, err will be returned if there's a problem reading it
		_, err = io.Copy(ioutil.Discard, resp.Body)
	}()

	return http.StatusAccepted, nil
}

func isSuccesful(statusCode int) bool {
	return statusCode/100 == 2
}
