package buffer

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/newrelic-fluent-bit-output/config"
	"github.com/newrelic-fluent-bit-output/nrclient"
	"github.com/newrelic-fluent-bit-output/utils"
	"log"
	"net/http"
)

type BufferManager struct {
	buffer        []map[string]interface{}
	config        config.BufferConfig
	lastFlushTime int64
	nrClient      nrclient.NRClient
}

func NewBufferManager(cfg config.BufferConfig, nrClient nrclient.NRClient) BufferManager {
	return BufferManager{
		config:        cfg,
		lastFlushTime: utils.TimeNowInMiliseconds(),
		nrClient:      nrClient,
	}
}

func (bufferManager *BufferManager) IsEmpty() bool {
	return len(bufferManager.buffer) == 0
}

func (bufferManager *BufferManager) AddRecord(record map[string]interface{}) chan *http.Response {
	bufferManager.buffer = append(bufferManager.buffer, record)
	if bufferManager.shouldSend() {
		return bufferManager.SendRecords()
	}

	return nil
}

func (bufferManager *BufferManager) shouldSend() bool {
	return (int64(len(bufferManager.buffer)) >= bufferManager.config.MaxRecords) ||
		((utils.TimeNowInMiliseconds() - bufferManager.lastFlushTime) > bufferManager.config.MaxTimeBetweenFlushes)
}

func (bufferManager *BufferManager) SendRecords() (responseChan chan *http.Response) {
	newBuffer := make([]map[string]interface{}, len(bufferManager.buffer))
	copy(newBuffer, bufferManager.buffer)
	bufferManager.buffer = nil
	bufferManager.lastFlushTime = utils.TimeNowInMiliseconds()
	responseChan = make(chan *http.Response, 1)
	bufferManager.prepare(newBuffer, responseChan)
	return responseChan
}

func (bufferManager *BufferManager) prepare(records []map[string]interface{}, responseChan chan *http.Response) {
	config := &bufferManager.config
	data, err := packagePayload(records)
	if err != nil {
		panic(err)
	}
	if int64(data.Cap()) >= config.MaxBufferSize {
		first := records[0 : len(records)/2]
		second := records[len(records)/2 : len(records)]
		bufferManager.prepare(first, responseChan)
		bufferManager.prepare(second, responseChan)
	} else {
		go func() {
			err := bufferManager.nrClient.Send(data, responseChan)
			if err != nil {
				log.Printf("[DEBUG] Error making HTTP request: %s", err)
			}
		}()
	}
}

func packagePayload(records []map[string]interface{}) (*bytes.Buffer, error) {
	var buffer bytes.Buffer
	data, err := json.Marshal(records)
	if err != nil {
		panic(err)
	}
	g := gzip.NewWriter(&buffer)
	if _, err = g.Write(data); err != nil {
		panic(err)
	}
	if err := g.Flush(); err != nil {
		panic(err)
	}
	if err = g.Close(); err != nil {
		panic(err)
	}
	return &buffer, nil
}
