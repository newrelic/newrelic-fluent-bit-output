package main

import (
	"C"
	"log"
	"unsafe"
	"github.com/newrelic/newrelic-fluent-bit-output/record"
	"github.com/fluent/fluent-bit-go/output"
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-fluent-bit-output/nrclient"
)

var nrClientRepo = make(map[string]*nrclient.NRClient)
var statusAccepted = 202

const(
	nonRetriableConnectionError = -1
	retriableConnectionError = -2
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "newrelic", "New relic output plugin")
}

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	cfg, err := config.NewPluginConfig(ctx)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return output.FLB_ERROR
	}
	var nrClient *nrclient.NRClient
	nrClient, err = nrclient.NewNRClient(cfg.NRClientConfig, cfg.ProxyConfig)
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}

	id := cfg.NRClientConfig.GetNewRelicKey()
	nrClientRepo[id] = nrClient
	output.FLBPluginSetContext(ctx, id)

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))

	// Iterate, parse and accumulate records to be sent
	var buffer []record.LogRecord
	for {
		// Extract Record
		ret, ts, fbRecord := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		buffer = append(buffer, record.RemapRecord(fbRecord, ts, VERSION))
	}

	id := output.FLBPluginGetContext(ctx).(string)
	nrClient := nrClientRepo[id]
	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	code, err := nrClient.Send(buffer)
	
	if err == nil  && code == statusAccepted {
		log.Printf("[INFO] Request accepted.")
		return output.FLB_OK
	} 
	if (err == nil && isRetriableStatusCode(code)) || (code == retriableConnectionError){
		log.Printf("[DEBUG] Retriable error received. Retry:true")
		return output.FLB_RETRY
	}
	
	log.Printf("[DEBUG] Non-retriable error received. Retry:false")
	return output.FLB_ERROR
}

func isRetriableStatusCode (statusCode int) bool {
	retriableCodes := []int{408, 429, 500, 502, 503, 504, 599}

	for _, code := range retriableCodes {
		if code == statusCode {
			return true
		}
	}
	
	return false
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func main() {
}
