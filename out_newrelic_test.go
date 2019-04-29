package main

import (
	"fmt"
	"math"
	"net/http"
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
				Expect(typeVal).To(Equal("fluent-bit"))
				Expect(version).To(Equal(VERSION))
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
		It("Correctly parses json sent as a message and doesn't overwrite reserved keys",
			func() {
				inputMap := make(map[interface{}]interface{})
				var inputTimestamp interface{}
				inputTimestamp = output.FLBTime{
					time.Now(),
				}
				inputMap["log"] = string(`{"message": "foo", "timestamp": 9001, "hostname": "bar"}`)
				foundOutput := prepareRecord(inputMap, inputTimestamp)
				Expect(foundOutput["timestamp"]).To(Equal(inputTimestamp.(output.FLBTime).UnixNano() / 1000000))
				Expect(foundOutput["message"]).To(Equal("foo"))
				Expect(foundOutput["hostname"]).To(Equal("bar"))
			},
		)
		It("Correctly parses nested json sent as a message",
			func() {
				inputMap := make(map[interface{}]interface{})
				var inputTimestamp interface{}
				inputTimestamp = output.FLBTime{
					time.Now(),
				}
				inputMap["log"] = string(`{"coolStories": {"foo": "bar", "hostname": "bar"}}`)
				foundOutput := prepareRecord(inputMap, inputTimestamp)
				Expect(foundOutput["timestamp"]).To(Equal(inputTimestamp.(output.FLBTime).UnixNano() / 1000000))
				Expect(foundOutput["coolStories"]).To(Equal(map[string]interface{}{"foo": "bar", "hostname": "bar"}))
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
		const insertKey = "sweetKey"
		var endpoint string
		var testConfig PluginConfig
		var bufferManager BufferManager
		vortexSuccessCode := 202
		vortexFailureCode := 500
		const maxRetries int64 = 7

		BeforeEach(func() {
			server = ghttp.NewServer()
			endpoint = server.URL() + "/v1/logs"

			testConfig = PluginConfig{
				apiKey:        insertKey,
				endpoint:      endpoint,
				maxBufferSize: 256000,
				maxRecords:    1,
				maxRetries:    maxRetries,
				// Don't sleep in tests, to keep tests fast
				initialRetryDelayInSeconds: 0,
				maxRetryDelayInSeconds:     0,
				maxTimeBetweenFlushes: 			5000,
			}
		})

		AfterEach(func() {
			server.Close()
		})

		It("Makes the expected HTTP call", func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v1/logs"),
					ghttp.VerifyHeader(http.Header{
						"X-Insert-Key":     []string{insertKey},
						"Content-Type":     []string{"application/json"},
						"Content-Encoding": []string{"gzip"},
					})))
			bufferManager = newBufferManager(testConfig)

			responseChan := bufferManager.addRecord(emptyMessage())

			// Wait for message to be sent
			Expect(responseChan).ToNot(BeNil())
			waitForChannel(responseChan)
		})

		It("test buffering by time", func() {
			server.AppendHandlers(ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))

			testConfig.maxRecords = math.MaxInt64 // Do not flush by count (we are testing flushing by time) 
			testConfig.maxTimeBetweenFlushes = int64((1 * time.Second) / time.Millisecond) // Flush after one second
			bufferManager = newBufferManager(testConfig)

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
			testConfig.maxRecords = 2
			bufferManager = newBufferManager(testConfig)

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

		It("retries if first call fails", func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexFailureCode, ""),
				ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))
			bufferManager = newBufferManager(testConfig)

			responseChan := bufferManager.addRecord(emptyMessage())

			Expect(responseChan).ToNot(BeNil())
			waitForChannel(responseChan)
		})

		It("retries if first two calls fail", func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexFailureCode, ""),
				ghttp.RespondWithJSONEncodedPtr(&vortexFailureCode, ""),
				ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))
			bufferManager = newBufferManager(testConfig)

			responseChan := bufferManager.addRecord(emptyMessage())

			Expect(responseChan).ToNot(BeNil())
			waitForChannel(responseChan)
		})

		It("quits retrying if all calls fail", func() {
			server.SetAllowUnhandledRequests(true)
			server.AppendHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexFailureCode, ""))
			bufferManager = newBufferManager(testConfig)

			bufferManager.addRecord(emptyMessage())

			// If the production code doesn't obey the max retries configuration, then this test
			// may not fail if the Eventually checks exactly when there have been the right amount of retries.
			// However, with zero sleep time, on a laptop the POST was done 0 times on the first check, and
			// ~50 times on the second check, so if you have a maxRetries in between, this should be relatively safe.
			Eventually(func() int { return len(server.ReceivedRequests()) }).Should(Equal(int(maxRetries) + 1))
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
