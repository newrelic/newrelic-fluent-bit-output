package main

import (
	"C"
	"github.com/fluent/fluent-bit-go/output"
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-fluent-bit-output/nrclient"
	"github.com/newrelic/newrelic-fluent-bit-output/record"
	log "github.com/sirupsen/logrus"
	"os"
	"unsafe"
)

var (
	nrClientRepo         = make(map[string]*nrclient.NRClient)
	dataFormatConfigRepo = make(map[string]config.DataFormatConfig)
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "newrelic", "New relic output plugin")
}

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	cfg, err := config.NewPluginConfig(ctx)
	if err != nil {
		log.WithField("error", err).Error("Error creating NewPluginConfig")
		return output.FLB_ERROR
	}
	var nrClient *nrclient.NRClient
	nrClient, err = nrclient.NewNRClient(cfg.NRClientConfig, cfg.ProxyConfig)
	if err != nil {
		log.WithField("error", err).Error("Error creating NewNRClient")
	}

	id := cfg.NRClientConfig.GetNewRelicKey()
	nrClientRepo[id] = nrClient
	dataFormatConfigRepo[id] = cfg.DataFormatConfig
	output.FLBPluginSetContext(ctx, id)

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))

	// Get New Relic Client
	id := output.FLBPluginGetContext(ctx).(string)
	nrClient := nrClientRepo[id]
	dataFormatConfig := dataFormatConfigRepo[id]

	// Iterate, parse and accumulate records to be sent
	var buffer []record.LogRecord
	for {
		// Extract Record
		ret, ts, fbRecord := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		buffer = append(buffer, record.RemapRecord(fbRecord, ts, VERSION, dataFormatConfig))
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	retry, err := nrClient.Send(buffer)
	if retry {
		log.Debug("Retryable error received.")
		return output.FLB_RETRY
	}
	if err != nil {
		log.WithField("error", err).Error("Unexpected non-retryable error received. Logs were discarded.")
		return output.FLB_ERROR
	}
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func main() {
	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}

	log.SetLevel(logLevel)
}
