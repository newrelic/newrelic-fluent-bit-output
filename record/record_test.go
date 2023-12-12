package record

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/newrelic/newrelic-fluent-bit-output/config"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
)

var _ = Describe("Out New Relic", func() {
	const pluginVersion = "0.0.0"

	// This lets the matching library (gomega) be able to notify the testing framework (ginkgo)
	gomega.RegisterFailHandler(ginkgo.Fail)

	Describe("Prepares payload", func() {
		AfterEach(func() {
			os.Unsetenv("SOURCE")
		})

		It("converts the map[interface{}] inteface{} to map[string] interface[], "+
			"updates the timestamp, and renames the log field to message", func() {
			inputMap := make(FluentBitRecord)
			var inputTimestamp interface{}
			inputTimestamp = output.FLBTime{
				time.Now(),
			}
			inputMap["log"] = "message"
			foundOutput := RemapRecord(inputMap, inputTimestamp, pluginVersion, config.DataFormatConfig{false})
			Expect(foundOutput["message"]).To(Equal("message"))
			Expect(foundOutput["log"]).To(BeNil())
			Expect(foundOutput["timestamp"]).To(Equal(inputTimestamp.(output.FLBTime).UnixNano() / 1000000))
			pluginMap := foundOutput["plugin"].(map[string]string)
			typeVal := pluginMap["type"]
			version := pluginMap["version"]
			source := pluginMap["source"]
			Expect(typeVal).To(Equal("fluent-bit"))
			Expect(version).To(Equal(pluginVersion))
			Expect(source).To(Equal("BARE-METAL"))
		})

		It("Doesn't rewrite plugin.type if it exits", func() {
			inputMap := make(FluentBitRecord)
			var inputTimestamp interface{}
			inputTimestamp = output.FLBTime{
				time.Now(),
			}
			expectedType := "something"
			inputMap["plugin"] = map[string]string{
				"type": expectedType,
			}
			foundOutput := RemapRecord(inputMap, inputTimestamp, pluginVersion, config.DataFormatConfig{false})
			pluginMap := foundOutput["plugin"].(map[string]string)
			Expect(pluginMap["type"]).To(Equal(expectedType))
		})

		It("Doesn't add plugin.type and plugin.version and uses compacted plugin.source format when in low data mode", func() {
			inputMap := make(FluentBitRecord)
			var inputTimestamp interface{}
			inputTimestamp = output.FLBTime{
				time.Now(),
			}
			foundOutput := RemapRecord(inputMap, inputTimestamp, pluginVersion, config.DataFormatConfig{true})
			Expect(foundOutput["plugin.source"]).To(Equal("BARE-METAL-fb-" + pluginVersion))
		})

		It("sets the source if it is included as an environment variable", func() {
			inputMap := make(FluentBitRecord)
			var inputTimestamp interface{}
			inputTimestamp = output.FLBTime{
				time.Now(),
			}
			expectedSource := "docker"
			inputMap["log"] = "message"
			os.Setenv("SOURCE", expectedSource)
			foundOutput := RemapRecord(inputMap, inputTimestamp, pluginVersion, config.DataFormatConfig{false})
			pluginMap := foundOutput["plugin"].(map[string]string)
			Expect(pluginMap["source"]).To(Equal(expectedSource))
		})

		It("Correctly massage nested map[interface]interface{} to map[string]interface{}", func() {
			// Given
			inputMap := map[interface{}]interface{}{
				"nestedMap": map[interface{}]interface{}{
					"foo":     "bar",
					"numeric": 2,
				},
			}

			// When
			foundOutput := parseRecord(inputMap)

			// Then
			expectedOutput := map[string]interface{}{
				"nestedMap": map[string]interface{}{
					"foo":     "bar",
					"numeric": 2,
				},
			}
			Expect(foundOutput).To(Equal(expectedOutput))
		})

		It("Correctly handles a JSON array in a []interface{}", func() {
			// Given
			inputMap := map[interface{}]interface{}{
				"nestedArray": []interface{}{
					map[interface{}]interface{}{
						"stringField":  "value1",
						"numericField": 1,
					},
					map[interface{}]interface{}{
						"stringField":  "value2",
						"numericField": 2,
					},
				},
			}

			// When
			foundOutput := parseRecord(inputMap)

			// Then
			expectedOutput := map[string]interface{}{
				"nestedArray": []interface{}{
					map[string]interface{}{
						"stringField":  "value1",
						"numericField": 1,
					},
					map[string]interface{}{
						"stringField":  "value2",
						"numericField": 2,
					},
				},
			}
			Expect(foundOutput).To(Equal(expectedOutput))
		})
	})

	Describe("Timestamp handling", func() {

		inputTimestampToExpectedOutput := map[interface{}]int64{
			// Modern Fluent Bit does uses FLBTime
			output.FLBTime{time.Unix(1234567890, 123456789)}: 1234567890123,

			// We've seen older of Fluent Bit versions use uint64
			// (generally being sent in seconds, but we handle other granularities out of paranoia)
			uint64(1234567890):          1234567890000,
			uint64(1234567890123):       1234567890123,
			uint64(1234567890123456):    1234567890123,
			uint64(1234567890123456789): 1234567890123,
		}

		for inputTimestamp, expectedOutputTime := range inputTimestampToExpectedOutput {
			// Lock in current values (otherwise all tests will run with the last values in the map)
			input := inputTimestamp
			expected := expectedOutputTime

			It("handles timestamps of various types and granularites : "+fmt.Sprintf("%v", input),
				func() {
					inputMap := make(FluentBitRecord)

					foundOutput := RemapRecord(inputMap, input, pluginVersion, config.DataFormatConfig{false})

					Expect(foundOutput["timestamp"]).To(Equal(expected))
				},
			)
		}

		It("extracts FLBTime timestamp from Fluent Bit Event nested array", func() {
			inputMap := make(FluentBitRecord)

			timestamp := []interface{}{
				output.FLBTime{time.Unix(1234567890, 123456789)},
				"Other metadata",
			}

			foundOutput := RemapRecord(inputMap, timestamp, pluginVersion, config.DataFormatConfig{false})

			Expect(foundOutput["timestamp"]).To(Equal(int64(1234567890123)))
		})

		It("extracts UInt64 timestamp from Fluent Bit Event nested array", func() {
			inputMap := make(FluentBitRecord)

			timestamp := []interface{}{
				uint64(1234567890),
				"Other metadata",
			}

			foundOutput := RemapRecord(inputMap, timestamp, pluginVersion, config.DataFormatConfig{false})

			Expect(foundOutput["timestamp"]).To(Equal(int64(1234567890000)))
		})

		It("ignores timestamps of unhandled types", func() {
			inputMap := make(FluentBitRecord)

			// We don't handle string types
			foundOutput := RemapRecord(inputMap, "1234567890", pluginVersion, config.DataFormatConfig{false})

			Expect(foundOutput["timestamp"]).To(BeNil())
		})

		// If the record has a timestamp attribute, we use it as-is
		// Otherwise we use the timestamp provided by fluentbit
		It("Record timestamp has precedence over fluentbit's", func() {
			inputMap := FluentBitRecord{"timestamp": 654321}

			foundOutput := RemapRecord(inputMap, uint64(1234567890), pluginVersion, config.DataFormatConfig{false})

			Expect(foundOutput["timestamp"]).To(Equal(654321))
		})
	})

	Describe("Record packaging", func() {

		It("returns an empty array of packages if the provided slice is nil", func() {
			// When
			packagedRecords, err := PackageRecords(nil)

			// Then
			Expect(err).To(BeNil())
			Expect(packagedRecords).To(Not(BeNil()))
			Expect(packagedRecords).To(HaveLen(0))
		})

		It("returns an empty array of packages if the provided slice is empty", func() {
			// When
			packagedRecords, err := PackageRecords([]LogRecord{})

			// Then
			Expect(err).To(BeNil())
			Expect(packagedRecords).To(Not(BeNil()))
			Expect(packagedRecords).To(HaveLen(0))
		})

		It("returns a compressed JSON array when an array of records is provided", func() {
			// Given
			logRecords := []LogRecord{
				{
					"timestamp": 1,
					"message":   "Some message 1",
				},
				{
					"timestamp": 2,
					"message":   "Some message 2",
				},
			}
			expectedJson := `[{"message":"Some message 1","timestamp":1},{"message":"Some message 2","timestamp":2}]`

			// When
			packagedRecords, err := PackageRecords(logRecords)

			// Then
			Expect(err).To(BeNil())
			Expect(packagedRecords).To(Not(BeNil()))
			// The two records are compressed into a single byte buffer, as they don't exceed 1MB
			Expect(packagedRecords).To(HaveLen(1))

			// Decompress and validate record
			uncompressedJson, err := uncompressRecord(packagedRecords[0])
			if err != nil {
				Fail(fmt.Sprintf("Could not uncompress record 0: %v", err))
			}
			Expect(uncompressedJson).To(Equal(expectedJson))
		})

		It("discards a log record if its compressed size exceeds 1MB in size", func() {
			// Given
			// Always use the same seed to get deterministic results
			rand.Seed(1)
			logRecords := []LogRecord{
				{
					"timestamp": 1,
					"message":   "Short message",
				},
				{
					"timestamp": 2,
					"message":   longRandomMessage(5),
				},
				{
					"timestamp": 3,
					"message":   "Short message 3",
				},
				{
					"timestamp": 4,
					"message":   "Short message 4",
				},
			}
			expectedJson0 := `[{"message":"Short message","timestamp":1}]`
			expectedJson1 := `[{"message":"Short message 3","timestamp":3},{"message":"Short message 4","timestamp":4}]`

			// When
			packagedRecords, err := PackageRecords(logRecords)

			// Then
			Expect(err).To(BeNil())
			Expect(packagedRecords).To(Not(BeNil()))
			// The two records are compressed into a single byte buffer, as they don't exceed 1MB
			Expect(packagedRecords).To(HaveLen(2))

			// Decompress and validate records
			uncompressedJson0, err := uncompressRecord(packagedRecords[0])
			if err != nil {
				Fail(fmt.Sprintf("Could not uncompress record 0: %v", err))
			}
			Expect(uncompressedJson0).To(Equal(expectedJson0))

			uncompressedJson1, err := uncompressRecord(packagedRecords[1])
			if err != nil {
				Fail(fmt.Sprintf("Could not uncompress record 1: %v", err))
			}
			Expect(uncompressedJson1).To(Equal(expectedJson1))
		})

		It("compacts multiple messages exceeding 1MB overall in multiple PackagedRecords", func() {
			// Given
			// Always use the same seed to get deterministic results
			rand.Seed(1)
			const numRecords = 20
			// 20 log records of about 256KB (given that they are random, we can assume that once
			// compressed their size won't be reduced much by gzipping them)
			var logRecords []LogRecord
			for i := 0; i < numRecords; i++ {
				randomMessage := longRandomMessage(1)
				logRecord := LogRecord{
					"recordId": i,
					"message":  randomMessage[:len(randomMessage)/4], // ~256KB approx
				}
				logRecords = append(logRecords, logRecord)
			}

			// When
			packagedRecords, err := PackageRecords(logRecords)

			// Then
			Expect(err).To(BeNil())
			Expect(packagedRecords).To(Not(BeNil()))
			// The 20 records are compressed into 12 byte buffers, as overall they exceed 1MB. Note that this is
			// a deterministic result (unless the gzip implementation changes), since we're always using the same
			// seed when generating the random messages, which will lead to the same messages always being sent.
			Expect(packagedRecords).To(HaveLen(12))

			// Count that we end up having 20 uncompressed records, with all the recordIds
			type Record struct {
				RecordId int    `json:"recordId"`
				Message  string `json:"message"`
			}
			receivedRecords := make(map[int]struct{}) // set of ints in Go
			for i, packagedRecord := range packagedRecords {
				uncompressedJson, err := uncompressRecord(packagedRecord)
				if err != nil {
					Fail(fmt.Sprintf("Could not uncompress record %d: %v. Record: %v", i, err, uncompressedJson))
				}

				var records []Record
				if err := json.Unmarshal([]byte(uncompressedJson), &records); err != nil {
					Fail(fmt.Sprintf("Could not unmarshall record %d: %v. Record: %s...%s", i, err, uncompressedJson[:50], uncompressedJson[len(uncompressedJson)-50:]))
				}

				// Record which recordIds we have received
				for _, record := range records {
					receivedRecords[record.RecordId] = struct{}{}
				}
			}

			Expect(receivedRecords).To(HaveLen(numRecords))
			for i := 0; i < numRecords; i++ {
				if _, ok := receivedRecords[i]; !ok {
					Fail(fmt.Sprintf("Record %d was not found in the uncompressed JSONs", i))
				}
			}
		})
	})
})

func uncompressRecord(packagedRecords PackagedRecords) (res string, err error) {
	gzipRecord := bytes.Buffer(*packagedRecords)
	reader, err := gzip.NewReader(&gzipRecord)
	if err != nil {
		return "", err
	}
	buff := make([]byte, 1024)
	for {
		n, err := reader.Read(buff)
		if err == io.EOF {
			// Finished reading
			break
		}
		if err != nil {
			return "", err
		}
		res += string(buff[:n])
	}
	if err != nil {
		return "", err
	}
	return res, nil
}

func longRandomMessage(sizeInMb int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Create a 5 MegaByte random message
	msgBytes := make([]byte, sizeInMb<<20)
	for i := range msgBytes {
		msgBytes[i] = charset[rand.Intn(len(charset))]
	}
	return string(msgBytes)
}
