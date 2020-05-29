package main

import (
	"C"
	"github.com/newrelic/newrelic-fluent-bit-output/record"
	"log"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"

	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-fluent-bit-output/nrclient"
)

var nrClientRepo = make(map[string]*nrclient.NRClient)

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
	if err := nrClient.Send(buffer); err != nil {
		return output.FLB_ERROR
	} else {
		return output.FLB_OK
	}
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func main() {
}
