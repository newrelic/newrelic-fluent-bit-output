package main

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Out New Relic", func() {

	// This lets the matching library (gomega) be able to notify the testing framework (ginkgo)
	gomega.RegisterFailHandler(ginkgo.Fail)

	Describe("Prepares payload", func() {
		AfterEach(func() {
			os.Unsetenv("SOURCE")
		})

		It("converts the map[interface{}] inteface{} to map[string] interface[], "+
			"updates the timestamp, and renames the log field to message",
			func() {
				inputMap := make(map[interface{}]interface{})
				var inputTimestamp interface{}
				inputTimestamp = output.FLBTime{
					time.Now(),
				}
				inputMap["log"] = "message"
				foundOutput := prepareRecord(inputMap, inputTimestamp)
				Expect(foundOutput["message"]).To(Equal("message"))
				Expect(foundOutput["log"]).To(BeNil())
				Expect(foundOutput["timestamp"]).To(Equal(inputTimestamp.(output.FLBTime).UnixNano() / 1000000))
				pluginMap := foundOutput["plugin"].(map[string]string)
				typeVal := pluginMap["type"]
				version := pluginMap["version"]
				source := pluginMap["source"]
				Expect(typeVal).To(Equal("fluent-bit"))
				Expect(version).To(Equal(VERSION))
				Expect(source).To(Equal("BARE-METAL"))
			},
		)
		It("sets the source if it is included as an environment variable",
			func() {
				inputMap := make(map[interface{}]interface{})
				var inputTimestamp interface{}
				inputTimestamp = output.FLBTime{
					time.Now(),
				}
				expectedSource := "docker"
				inputMap["log"] = "message"
				os.Setenv("SOURCE", expectedSource)
				foundOutput := prepareRecord(inputMap, inputTimestamp)
				pluginMap := foundOutput["plugin"].(map[string]string)
				Expect(pluginMap["source"]).To(Equal(expectedSource))
			},
		)

		It("Correctly massage nested map[interface]interface{} to map[string]interface{}",
			func() {
				inputMap := make(map[interface{}]interface{})
				nestedMap := make(map[interface{}]interface{})
				expectedOutput := make(map[string]interface{})
				expectedNestedOutput := make(map[string]interface{})
				expectedNestedOutput["foo"] = "bar"
				expectedOutput["nested"] = expectedNestedOutput
				nestedMap["foo"] = "bar"
				inputMap["nested"] = nestedMap
				foundOutput := remapRecord(inputMap)
				Expect(foundOutput).To(Equal(expectedOutput))

			},
		)
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
					inputMap := make(map[interface{}]interface{})

					foundOutput := prepareRecord(inputMap, input)

					Expect(foundOutput["timestamp"]).To(Equal(int64(expected)))
				},
			)
		}

		It("ignores timestamps of unhandled types",
			func() {
				inputMap := make(map[interface{}]interface{})

				// We don't handle string types
				foundOutput := prepareRecord(inputMap, "1234567890")

				Expect(foundOutput["timestamp"]).To(BeNil())
			},
		)
	})

	Describe("HTTP Request", func() {

		var server *ghttp.Server
		const insertKey = "some-insert-key"
		const licenseKey = "some-license-key"
		var endpoint string
		var insertKeyConfig PluginConfig
		var licenseKeyConfig PluginConfig
		var bufferManager BufferManager
		vortexSuccessCode := 202

		BeforeEach(func() {
			server = ghttp.NewServer()
			endpoint = server.URL() + "/v1/logs"

			insertKeyConfig = PluginConfig{
				apiKey:        insertKey,
				// Ideally we shouldn't have to set this separately from insertKey, but where this is set is
				// in the Fluent Bit code that we can't unit test
				useApiKey:     true,
				endpoint:      endpoint,
				maxBufferSize: 256000,
				maxRecords:    1,
				// Don't sleep in tests, to keep tests fast
				maxTimeBetweenFlushes: 5000,
			}

			licenseKeyConfig = PluginConfig{
				licenseKey:    licenseKey,
				// Ideally we shouldn't have to set this separately from licenseKey, but where this is set is
				// in the Fluent Bit code that we can't unit test
				useApiKey:     false,
				endpoint:      endpoint,
				maxBufferSize: 256000,
				maxRecords:    1,
				// Don't sleep in tests, to keep tests fast
				maxTimeBetweenFlushes: 5000,
			}
		})

		AfterEach(func() {
			server.Close()
		})

		It("Makes the expected HTTP call with api key", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""),
					ghttp.VerifyRequest("POST", "/v1/logs"),
					ghttp.VerifyHeader(http.Header{
						"X-Insert-Key":     []string{insertKey},
						"Content-Type":     []string{"application/json"},
						"Content-Encoding": []string{"gzip"},
					})))
			bufferManager = newBufferManager(insertKeyConfig)

			responseChan := bufferManager.addRecord(emptyMessage())

			// Wait for message to be sent
			Expect(responseChan).ToNot(BeNil())
			waitForChannel(responseChan)
		})

		It("Makes the expected HTTP call with License Key", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""),
					ghttp.VerifyRequest("POST", "/v1/logs"),
					ghttp.VerifyHeader(http.Header{
						"X-License-Key":    []string{licenseKey},
						"Content-Type":     []string{"application/json"},
						"Content-Encoding": []string{"gzip"},
					})))
			bufferManager = newBufferManager(licenseKeyConfig)

			responseChan := bufferManager.addRecord(emptyMessage())

			// Wait for message to be sent
			Expect(responseChan).ToNot(BeNil())
			waitForChannel(responseChan)
		})

		It("test buffering by time", func() {
			server.AppendHandlers(ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))

			insertKeyConfig.maxRecords = math.MaxInt64                                          // Do not flush by count (we are testing flushing by time)
			insertKeyConfig.maxTimeBetweenFlushes = int64((1 * time.Second) / time.Millisecond) // Flush after one second
			bufferManager = newBufferManager(insertKeyConfig)

			responseChan := bufferManager.addRecord(make(map[string]interface{}))
			Expect(responseChan).To(BeNil())

			// Wait twice as long as the max time between flushes
			time.Sleep(2 * time.Second)

			// This record doesn't fill the buffer, but we exceed the max time between flushes, so we flush
			responseChan = bufferManager.addRecord(make(map[string]interface{}))
			Expect(responseChan).ToNot(BeNil())

			<-responseChan
			Expect(bufferManager.shouldSend()).To(BeFalse())
		})

		It("only flushes when buffer is full, then resets buffer", func() {
			server.AppendHandlers(ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))

			// Don't send message until we've added two messages
			insertKeyConfig.maxRecords = 2
			bufferManager = newBufferManager(insertKeyConfig)

			// Add one message, should not send yet
			responseChan := bufferManager.addRecord(emptyMessage())
			Expect(responseChan).To(BeNil())

			// Add another message, should send
			responseChan = bufferManager.addRecord(emptyMessage())
			Expect(responseChan).ToNot(BeNil())

			// Check that buffer is cleared after sending
			waitForChannel(responseChan)
			Expect(bufferManager.shouldSend()).To(BeFalse())
		})
	})
})

func emptyMessage() map[string]interface{} {
	return make(map[string]interface{})
}

func waitForChannel(responseChan chan *http.Response) {
	maximumWaitInSeconds := 10
	maximumWait := time.Duration(maximumWaitInSeconds) * time.Second
	select {
	case <-responseChan:
	case <-time.After(maximumWait):
		Fail(fmt.Sprintf("Channel was not written to within %d seconds", maximumWaitInSeconds))
	}
}
