package main

import (
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/fluent/fluent-bit-go/output"
)
import (
	"C"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"strconv"
	"unsafe"
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "newrelic", "New relic output plugin")
}

var client *http.Client

type PluginConfig struct {
	endpoint      string
	apiKey        string
	maxBufferSize int64
	maxRecords    int64
}

var config PluginConfig

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
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
	keepAliveTimeout := 600 * time.Second
	timeout := 5 * time.Second
	defaultTransport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: keepAliveTimeout,
		}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	client = &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}

	return output.FLB_OK
}

func prepareRecord(inputRecord map[interface{}]interface{}, inputTimestamp interface{}) (outputRecord map[string]interface{}) {
	outputRecord = make(map[string]interface{})
	timestamp := inputTimestamp.(output.FLBTime)
	inputRecord["timestamp"] = timestamp.UnixNano() / 1000000
	for k, v := range inputRecord {
		// TODO:  We may have to do flattening
		switch value := v.(type) {
		case []byte:
			outputRecord[k.(string)] = string(value)
			break
		case string:
			outputRecord[k.(string)] = value
			break
		default:
			outputRecord[k.(string)] = value
		}
	}
	if val, ok := outputRecord["log"]; ok {
		outputRecord["message"] = val
		delete(outputRecord, "log")
	}
	return
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var count int64
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}
	var buffer []map[string]interface{}

	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))

	// Iterate Records
	count = 0
	for {
		// Extract Record
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}

		updatedRecord := prepareRecord(record, ts)
		buffer = append(buffer, updatedRecord)
		count++
		if config.maxRecords >= count {
			newBuffer := make([]map[string]interface{}, len(buffer))
			copy(newBuffer, buffer)
			prepare(buffer, &config)
			count = 0
			buffer = nil
		}
	}
	if len(buffer) > 0 {
		prepare(buffer, &config)
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

func prepare(records []map[string]interface{}, config *PluginConfig) (responseChan chan *http.Response) {
	responseChan = make(chan *http.Response)
	data, err := packagePayload(records)
	if err != nil {
		panic(err)
	}
	if int64(data.Cap()) >= config.maxBufferSize {
		first := records[0 : len(records)/2]
		second := records[len(records)/2 : len(records)]
		prepare(first, config)
		prepare(second, config)
	} else {
		// TODO: error handling, retry, exponential backoff
		go makeRequest(data, config, responseChan)
		return responseChan
	}
	return nil
}

func makeRequest(buffer *bytes.Buffer, config *PluginConfig, responseChan chan *http.Response) {
	req, err := http.NewRequest("POST", config.endpoint, buffer)
	if err != nil {
		panic(err)
	}
	req.Header.Add("X-Insert-Key", config.apiKey)
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
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
	return output.FLB_OK
}

func main() {
}
