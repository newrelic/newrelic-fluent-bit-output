package nrclient

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/newrelic-fluent-bit-output/config"
)

type NRClient struct {
	client *http.Client
	config config.NRClientConfig
}

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

func (nrClient *NRClient) Send(buffer *bytes.Buffer, responseChan chan *http.Response) error {
	req, err := http.NewRequest("POST", nrClient.config.Endpoint, buffer)
	if err != nil {
		return err
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
		log.Printf("[ERROR] Error making HTTP request: %s", err)
		return err
	} else if resp.StatusCode != 202 {
		log.Printf("[ERROR] Error making HTTP request.  Got status code: %v", resp.StatusCode)
		return nil
	}
	defer resp.Body.Close()
	defer func() {
		_, err = io.Copy(ioutil.Discard, resp.Body) // WE READ THE BODY
	}()
	if err != nil {
		return err
	}

	responseChan <- resp
	return nil
}
