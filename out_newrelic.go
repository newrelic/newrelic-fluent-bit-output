package main

import (
	"io"
	"io/ioutil"
	"net"
	"time"

	"os"
	"C"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log"
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
	endpoint                   string
	apiKey                     string
	maxBufferSize              int64
	maxRecords                 int64
	initialRetryDelayInSeconds int64
	maxRetryDelayInSeconds     int64
	maxRetries                 int64
	maxTimeBetweenFlushes      int64
}

type BufferManager struct {
	config PluginConfig
	buffer []map[string]interface{}
	client *http.Client
	lastFlushTime int64
}

type Retryer struct {
	retries                    int64
	maxRetries                 int64
	initialRetryDelayInSeconds int64
	maxRetryDelayInSeconds     int64
}

func newRetryer(config PluginConfig) Retryer {
	return Retryer {
		retries: 0,
		maxRetries: config.maxRetries,
		initialRetryDelayInSeconds: config.initialRetryDelayInSeconds,
		maxRetryDelayInSeconds: config.maxRetryDelayInSeconds,
	}
}

func min(first int64, last int64) int64 {
	if first < last {
		return first
	} else {
		return last
	}
}

func (retryer *Retryer) retry() bool {
	if retryer.retries + 1 >= retryer.maxRetries {
		return false
	} else {
		retryer.retries++
	}
	delay := retryer.retries * 2 + retryer.initialRetryDelayInSeconds
	duration := time.Duration(int(min(delay, retryer.maxRetryDelayInSeconds))) * time.Second
	time.Sleep(duration)
	return true;
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
	client := &http.Client{
		Transport: defaultTransport,
		Timeout: timeout,
	}

	return BufferManager{
		lastFlushTime: timeNowInMiliseconds(),
		config: config,
		client: client,
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
	return (int64(len(bufferManager.buffer)) >= bufferManager.config.maxRecords) ||
		(((timeNowInMiliseconds() - bufferManager.lastFlushTime)) > bufferManager.config.maxTimeBetweenFlushes)
}

func (bufferManager *BufferManager) sendRecords() (responseChan chan *http.Response) {
	newBuffer := make([]map[string]interface{}, len(bufferManager.buffer))
	copy(newBuffer, bufferManager.buffer)
	bufferManager.buffer = nil
	bufferManager.lastFlushTime = timeNowInMiliseconds()
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
		go func() {
			retryer := newRetryer(bufferManager.config)
			err := bufferManager.makeRequest(data, responseChan, &retryer)
			if err != nil {
				// TODO: what's the right thing to do here?
				log.Printf("[DEBUG] Error making HTTP request: %s", err)
			}
		}()
	}
}

func (bufferManager *BufferManager) makeRequest(buffer *bytes.Buffer, responseChan chan *http.Response, retryer *Retryer) error {
	req, err := http.NewRequest("POST", bufferManager.config.endpoint, buffer)
	if err != nil {
		return err
	}
	req.Header.Add("X-Insert-Key", bufferManager.config.apiKey)
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	resp, err := bufferManager.client.Do(req)
	if (err != nil || resp.StatusCode != 202) && retryer.retry() {
		return bufferManager.makeRequest(buffer, responseChan, retryer)
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

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	var config PluginConfig
	// Example to retrieve an optional configuration parameter
	config.endpoint = output.FLBPluginConfigKey(ctx, "endpoint")
	if len(config.endpoint) == 0 {
		config.endpoint = "https://log-api.newrelic.com/log/v1"
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

	possibleInitialRetryDelayInSeconds := output.FLBPluginConfigKey(ctx, "initialRetryDelayInSeconds")
	if len(possibleInitialRetryDelayInSeconds) == 0 {
		config.initialRetryDelayInSeconds = 5
	} else {
		config.initialRetryDelayInSeconds, _ = strconv.ParseInt(possibleInitialRetryDelayInSeconds, 10, 64)
	}

	possibleMaxRetryDelayInSeconds := output.FLBPluginConfigKey(ctx, "maxRetryDelayInSeconds")
	if len(possibleMaxRetryDelayInSeconds) == 0 {
		config.maxRetryDelayInSeconds = 30
	} else {
		config.maxRetryDelayInSeconds, _ = strconv.ParseInt(possibleMaxRetryDelayInSeconds, 10, 64)
	}

	possibleMaxRetries := output.FLBPluginConfigKey(ctx, "maxRetries")
	if len(possibleMaxRetries) == 0 {
		config.maxRetries = 5
	} else {
		config.maxRetries, _ = strconv.ParseInt(possibleMaxRetries, 10, 64)
	}

	possibleMaxTimeBetweenFlushes := output.FLBPluginConfigKey(ctx, "maxTimeBetweenFlushes")
	if len(possibleMaxTimeBetweenFlushes) == 0 {
		config.maxTimeBetweenFlushes = 5000
	} else {
		config.maxTimeBetweenFlushes, _ =  strconv.ParseInt(possibleMaxTimeBetweenFlushes, 10, 64)
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

func timeToMillis(time int64) int64 {
	// 18 Apr 2019 == 1555612951401 msecs
	const maxSeconds = 2000000000
	const maxMilliseconds = maxSeconds * 1000
	const maxMicroseconds = maxMilliseconds * 1000
	if time < maxSeconds {
		return time * 1000
	} else if time < maxMilliseconds {
		return time
	} else if time < maxMicroseconds {
		return time / 1000
	} else { // Assume nanoseconds
		return time / 1000000
	}
}

func prepareRecord(inputRecord map[interface{}]interface{}, inputTimestamp interface{}) (outputRecord map[string]interface{}) {
	outputRecord = make(map[string]interface{})
	outputRecord = remapRecord(inputRecord)

	switch inputTimestamp.(type) {
	case output.FLBTime:
		outputRecord["timestamp"] = timeToMillis(inputTimestamp.(output.FLBTime).UnixNano())
	case uint64:
		outputRecord["timestamp"] = timeToMillis(int64(inputTimestamp.(uint64)))
	default:
		// Unhandled timestamp type, just ignore (don't log, since I assume we'll fill up someone's disk)
	}

	if val, ok := outputRecord["log"]; ok {
		var nested map[string]interface{}
		if err := json.Unmarshal([]byte(val.(string)), &nested); err == nil {
			remapped := remapRecordString(nested)
			for k, v := range remapped {
				if _, ok := outputRecord[k]; !ok {
					outputRecord[k] = v
				}
			}
		} else {
			outputRecord["message"] = val
		}
		delete(outputRecord, "log")
	}
	source, ok := os.LookupEnv("SOURCE")
	if !ok {
		source = "BARE-METAL"
	}
	outputRecord["plugin"] = map[string]string {
		"type": "fluent-bit",
		"version": VERSION,
		"source": source,
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

//utility for time now in  miliseconds
func timeNowInMiliseconds() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}


func main() {
}
