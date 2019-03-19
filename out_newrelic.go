package main

import (
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

var client = http.Client{}
var endpoint, apiKey string
var maxBufferSize, maxRecords int64

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	// Example to retrieve an optional configuration parameter
	possibleOverride := output.FLBPluginConfigKey(ctx, "endpoint")
	if len(possibleOverride) > 0 {
		endpoint = possibleOverride
	} else {
		endpoint = "https://insights-collector.newrelic.com/logs/v1"
	}
	apiKey = output.FLBPluginConfigKey(ctx, "apiKey")
	maxBufferSize, _ = strconv.ParseInt(output.FLBPluginConfigKey(ctx, "maxBufferSize"), 10, 64)
	maxRecords, _ = strconv.ParseInt(output.FLBPluginConfigKey(ctx, "maxRecords"), 10, 64)
	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var count int64
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}
	var updatedRecord = make(map[string]interface{})
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

		// Print record keys and values
		timestamp := ts.(output.FLBTime)
		record["timestamp"] = timestamp.UnixNano() / 1000000
		for k, v := range record {
			// TODO:  We may have to do flattening
			switch value := v.(type) {
			case []byte:
				updatedRecord[k.(string)] = string(value)
				break
			case string:
				updatedRecord[k.(string)] = value
				break
			default:
				updatedRecord[k.(string)] = value
			}
		}
		buffer = append(buffer, updatedRecord)
		count++
		if maxRecords >= count {
			newBuffer := make([]map[string]interface{}, len(buffer))
			copy(newBuffer, buffer)
			prepare(buffer)
			count = 0
			buffer = nil
		}
	}
	if len(buffer) > 0 {
		prepare(buffer)
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

func prepare(records []map[string]interface{}) {
	data, err := packagePayload(records)
	if err != nil {
		panic(err)
	}
	if int64(data.Cap()) >= maxBufferSize {
		first := records[0 : len(records)/2]
		second := records[len(records)/2 : len(records)]
		prepare(first)
		prepare(second)
	} else {
		// TODO: error handling, retry, exponential backoff
		go makeRequest(data)
	}
}

func makeRequest(buffer *bytes.Buffer) {
	req, err := http.NewRequest("POST", endpoint, buffer)
	if err != nil {
		panic(err)
	}
	req.Header.Add("X-Insert-Key", apiKey)
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
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
