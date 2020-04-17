package nrclient

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/newrelic-fluent-bit-output/config"
)

type NRClient struct {
	client *http.Client
	config config.NRClientConfig
}

func NewNRClient(cfg config.NRClientConfig) NRClient {
	keepAliveTimeout := 600 * time.Second
	timeout := 5 * time.Second
	defaultTransport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: keepAliveTimeout,
		}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}

	return NRClient{
		client: &http.Client{
			Transport: defaultTransport,
			Timeout:   timeout,
		},
		config: cfg,
	}
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
		log.Printf("[DEBUG] Error making HTTP request: %s", err)
		return err
	} else if resp.StatusCode != 202 {
		log.Printf("[DEBUG] Error making HTTP request.  Got status code: %v", resp.StatusCode)
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
