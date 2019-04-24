package main

import (
	"fmt"
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

	Describe("Prepares payload", func() {
		gomega.RegisterFailHandler(ginkgo.Fail)
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
				Expect(version).To(Equal("0.0.26"))
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

	Describe("HTTP Request body", func() {
		gomega.RegisterFailHandler(ginkgo.Fail)
		var server *ghttp.Server
		const expectedInsertKey = "sweetKey"
		var expectedEndpoint string
		var testConfig PluginConfig

		BeforeEach(func() {
			server = ghttp.NewServer()
			expectedEndpoint = server.URL() + "/v1/logs"
			server.AppendHandlers(
				ghttp.CombineHandlers(ghttp.VerifyRequest("POST", "/v1/logs"),
					ghttp.VerifyHeader(http.Header{
						"X-Insert-Key":     []string{expectedInsertKey},
						"Content-Type":     []string{"application/json"},
						"Content-Encoding": []string{"gzip"},
					}),
				))

			testConfig = PluginConfig{
				apiKey:        expectedInsertKey,
				endpoint:      expectedEndpoint,
				maxBufferSize: 256000,
				maxRecords:    2,
			}
		})

		AfterEach(func() {
			server.Close()
		})

		It("flushes when buffer is full, resets buffer", func() {
			bufferManager := newBufferManager(testConfig)
			testRecord := make(map[string]interface{})
			responseChan := bufferManager.addRecord(testRecord)

			Expect(responseChan).To(BeNil())

			testRecord2 := make(map[string]interface{})
			responseChan = bufferManager.addRecord(testRecord2)

			Expect(responseChan).ToNot(BeNil())
			<-responseChan

			Expect(bufferManager.shouldSend()).To(BeFalse())
		})
	})
})
