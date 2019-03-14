package main

import (
	"github.com/fluent/fluent-bit-go/output"
)
import (
	"C"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
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
	endpoint = output.FLBPluginConfigKey(ctx, "endpoint")
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
				fmt.Println("we are default")
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

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

func prepare(records []map[string]interface{}) {
	data, _ := packagePayload(records)
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
	req, _ := http.NewRequest("POST", endpoint, buffer)
	req.Header.Add("X-Insert-Key", apiKey)
	req.Header.Add("Content-Encoding", "gzip")
	client.Do(req)
}

func packagePayload(records []map[string]interface{}) (buffer *bytes.Buffer, err error) {
	data, err := json.Marshal(records)
	g := gzip.NewWriter(buffer)
	if _, err = g.Write(data); err != nil {
		return
	}
	if err = g.Close(); err != nil {
		return
	}
	return
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func main() {
}
