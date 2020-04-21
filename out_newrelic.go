package main

import (
	"C"
	"log"
	"os"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"

	"github.com/newrelic/newrelic-fluent-bit-output/buffer"
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-fluent-bit-output/nrclient"
	"github.com/newrelic/newrelic-fluent-bit-output/utils"
)

var bufferManager buffer.BufferManager

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

	nrClient, err := nrclient.NewNRClient(cfg.NRClientConfig, cfg.ProxyConfig)
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}

	bufferManager = buffer.NewBufferManager(cfg.BufferManagerConfig, *nrClient)
	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))
	// Iterate Records
	for {
		// Extract Record
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}
		updatedRecord := prepareRecord(record, ts)
		bufferManager.AddRecord(updatedRecord)
	}
	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

func prepareRecord(inputRecord map[interface{}]interface{}, inputTimestamp interface{}) (outputRecord map[string]interface{}) {
	outputRecord = make(map[string]interface{})
	outputRecord = remapRecord(inputRecord)

	switch inputTimestamp.(type) {
	case output.FLBTime:
		outputRecord["timestamp"] = utils.TimeToMillis(inputTimestamp.(output.FLBTime).UnixNano())
	case uint64:
		outputRecord["timestamp"] = utils.TimeToMillis(int64(inputTimestamp.(uint64)))
	default:
		// Unhandled timestamp type, just ignore (don't log, since I assume we'll fill up someone's disk)
	}

	if val, ok := outputRecord["log"]; ok {
		outputRecord["message"] = val
		delete(outputRecord, "log")
	}
	source, ok := os.LookupEnv("SOURCE")
	if !ok {
		source = "BARE-METAL"
	}
	outputRecord["plugin"] = map[string]string{
		"type":    "fluent-bit",
		"version": VERSION,
		"source":  source,
	}
	return
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

//export FLBPluginExit
func FLBPluginExit() int {
	if !bufferManager.IsEmpty() {
		bufferManager.SendRecords()
	}
	return output.FLB_OK
}

func main() {
}
