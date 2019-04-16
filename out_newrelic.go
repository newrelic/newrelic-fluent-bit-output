package main

import (
	"io"
	"io/ioutil"
	"net"
	"time"

	"C"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"strconv"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "newrelic", "New relic output plugin")
}

type PluginConfig struct {
	endpoint      string
	apiKey        string
	maxBufferSize int64
	maxRecords    int64
}

type BufferManager struct {
	config PluginConfig
	buffer []map[string]interface{}
	client *http.Client
}

var bufferManager BufferManager

func newBufferManager(config PluginConfig) BufferManager {
	keepAliveTimeout := 600 * time.Second
	timeout := 5 * time.Second
	defaultTransport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: keepAliveTimeout,
		}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	return BufferManager{
		config: config,
		client: &http.Client{
			Transport: defaultTransport,
			Timeout:   timeout,
		},
	}
}

func (bufferManager *BufferManager) addRecord(record map[string]interface{}) chan *http.Response {
	bufferManager.buffer = append(bufferManager.buffer, record)
	if bufferManager.shouldSend() {
		return bufferManager.sendRecords()
	}

	return nil
}

func (bufferManager *BufferManager) isEmpty() bool {
	return len(bufferManager.buffer) == 0
}

func (bufferManager *BufferManager) shouldSend() bool {
	return int64(len(bufferManager.buffer)) >= bufferManager.config.maxRecords
}

func (bufferManager *BufferManager) sendRecords() (responseChan chan *http.Response) {
	newBuffer := make([]map[string]interface{}, len(bufferManager.buffer))
	copy(newBuffer, bufferManager.buffer)
	bufferManager.buffer = nil

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
	if int64(data.Cap()) >= config.maxBufferSize {
		first := records[0 : len(records)/2]
		second := records[len(records)/2 : len(records)]
		bufferManager.prepare(first, responseChan)
		bufferManager.prepare(second, responseChan)
	} else {
		// TODO: error handling, retry, exponential backoff
		go bufferManager.makeRequest(data, responseChan)
	}
}

func (bufferManager *BufferManager) makeRequest(buffer *bytes.Buffer, responseChan chan *http.Response) {
	req, err := http.NewRequest("POST", bufferManager.config.endpoint, buffer)
	if err != nil {
		panic(err)
	}
	req.Header.Add("X-Insert-Key", bufferManager.config.apiKey)
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	resp, err := bufferManager.client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body) // WE READ THE BODY
	if err != nil {
		panic(err)
	}
	responseChan <- resp
}

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	var config PluginConfig
	// Example to retrieve an optional configuration parameter
	config.endpoint = output.FLBPluginConfigKey(ctx, "endpoint")
	if len(config.endpoint) == 0 {
		config.endpoint = "https://insights-collector.newrelic.com/logs/v1"
	}
	config.apiKey = output.FLBPluginConfigKey(ctx, "apiKey")
	if len(config.apiKey) == 0 {
		return output.FLB_ERROR
	}

	possibleeMaxBufferSize := output.FLBPluginConfigKey(ctx, "maxBufferSize")
	if len(possibleeMaxBufferSize) == 0 {
		config.maxBufferSize = 256000
	} else {
		config.maxBufferSize, _ = strconv.ParseInt(possibleeMaxBufferSize, 10, 64)
	}
	possibleMaxRecords := output.FLBPluginConfigKey(ctx, "maxRecords")
	if len(possibleMaxRecords) == 0 {
		config.maxRecords = 1024
	} else {
		config.maxRecords, _ = strconv.ParseInt(possibleMaxRecords, 10, 64)
	}
	bufferManager = newBufferManager(config)
	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}

	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))
	// Iterate Records
	for {
		// Extract Record
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}
		updatedRecord := prepareRecord(record, ts)
		bufferManager.addRecord(updatedRecord)
	}
	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

func remapRecord(inputRecord map[interface{}]interface{}) (outputRecord map[string]interface{}) {
	outputRecord = make(map[string]interface{})
	for k, v := range inputRecord {
		switch value := v.(type) {
		case []byte:
			outputRecord[k.(string)] = string(value)
			break
		case string:
			outputRecord[k.(string)] = value
			break
		case map[interface{}]interface{}:
			outputRecord[k.(string)] = remapRecord(value)
		default:
			outputRecord[k.(string)] = value
		}
	}
	return
}

func remapRecordString(inputRecord map[string]interface{}) (outputRecord map[string]interface{}) {
	outputRecord = make(map[string]interface{})
	for k, v := range inputRecord {
		switch value := v.(type) {
		case []byte:
			outputRecord[k] = string(value)
			break
		case string:
			outputRecord[k] = value
			break
		case map[interface{}]interface{}:
			outputRecord[k] = remapRecord(value)
		default:
			outputRecord[k] = value
		}
	}
	return
}

func prepareRecord(inputRecord map[interface{}]interface{}, inputTimestamp interface{}) (outputRecord map[string]interface{}) {
	outputRecord = make(map[string]interface{})
	timestamp := inputTimestamp.(output.FLBTime)
	outputRecord = remapRecord(inputRecord)
	outputRecord["timestamp"] = timestamp.UnixNano() / 1000000
	if val, ok := outputRecord["log"]; ok {
		var nested map[string]interface{}
		if err := json.Unmarshal([]byte(val.(string)), &nested); err == nil {
			remapped := remapRecordString(nested)
			for k, v := range remapped {
				switch k {
				case "timestamp":
					break
				default:
					outputRecord[k] = v
				}
			}
		} else {
			outputRecord["message"] = val
		}
		delete(outputRecord, "log")
	}
	return
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

//export FLBPluginExit
func FLBPluginExit() int {
	if !bufferManager.isEmpty() {
		bufferManager.sendRecords()
	}
	return output.FLB_OK
}

func main() {
}
