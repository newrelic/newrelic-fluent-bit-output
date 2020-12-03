package record

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log"
	"os"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/newrelic/newrelic-fluent-bit-output/utils"
)

const maxPacketSize = 1000000 // bytes

type FluentBitRecord map[interface{}]interface{}

type LogRecord map[string]interface{}

type PackagedRecords *bytes.Buffer

// RemapRecord takes a log record emitted by FluentBit, parses it into a NewRelic LogRecord
// domain type and performs several key name re-mappings.
func RemapRecord(inputRecord FluentBitRecord, inputTimestamp interface{}, pluginVersion string) (outputRecord LogRecord) {
	outputRecord = make(map[string]interface{})
	outputRecord = parseRecord(inputRecord)

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
	if _, ok = outputRecord["plugin"]; !ok {
		outputRecord["plugin"] = map[string]string{
			"type":    "fluent-bit",
			"version": pluginVersion,
			"source":  source,
		}
	}
	return
}

// parseRecord transforms a log record emitted by FluentBit into a LogRecord
// domain type: a map of string keys and arbitrary (int, string, etc.) values.
// No value modification is performed by this method (except casting).
func parseRecord(inputRecord map[interface{}]interface{}) map[string]interface{} {
	return parseValue(inputRecord).(map[string]interface{})
}

func parseValue(value interface{}) interface{} {
	switch value := value.(type) {
	case []byte:
		return string(value)
	case map[interface{}]interface{}:
		remapped := make(map[string]interface{})
		for k, v := range value {
			remapped[k.(string)] = parseValue(v)
		}
		return remapped
	case []interface{}:
		remapped := make([]interface{}, len(value))
		for i, v := range value {
			remapped[i] = parseValue(v)
		}
		return remapped
	default:
		return value
	}
}

// PackageRecords gets an array of LogRecords and returns them as an array of PackagedRecords
// (byte buffers), ready to be sent to NewRelic.
//
// If any record exceeds 1MB after being compressed, then it does not get included in the final result
// and the resulting compressed array is split at the  point where that long record was present.
//
// For example:
//     INPUT: [shortRecord, longRecord, shortRecord2, shortRecord3]
//     OUTPUT: [GZIP(JSON(shortRecord)), GZIP(JSON(shortRecord2, shortRecord3))]
func PackageRecords(records []LogRecord) ([]PackagedRecords, error) {
	if len(records) == 0 {
		return []PackagedRecords{}, nil
	}

	compressedData, err := asGzippedJson(records)
	if err != nil {
		return nil, err
	}
	// TODO Check Ian/Brian: I do believe that this should be compresssedData.Len(), let's confirm it before changing.
	compressedSize := int64(compressedData.Cap())
	if compressedSize >= maxPacketSize && len(records) == 1 {
		log.Printf("[ERROR] Can't compress record below required maximum packet size and it will be discarded.")
		return []PackagedRecords{}, nil
	} else if compressedSize >= maxPacketSize && len(records) > 1 {
		log.Printf("[DEBUG] Records were too big, splitting in half and retrying compression again.")
		firstHalf, err := PackageRecords(records[:len(records)/2])
		if err != nil {
			return nil, err
		}
		secondHalf, err := PackageRecords(records[len(records)/2:])
		if err != nil {
			return nil, err
		}

		return append(firstHalf, secondHalf...), nil
	} else {
		return []PackagedRecords{compressedData}, nil
	}
}

// asGzippedJson takes an array of LogRecords, encodes them as a JSON array and
// compresses them into a byte buffer using the GZip compression algorithm.
func asGzippedJson(records []LogRecord) (*bytes.Buffer, error) {
	buff := new(bytes.Buffer)
	data, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}
	g := gzip.NewWriter(buff)
	if _, err := g.Write(data); err != nil {
		return nil, err
	}
	if err = g.Flush(); err != nil {
		return nil, err
	}
	if err = g.Close(); err != nil {
		return nil, err
	}
	return buff, nil
}
