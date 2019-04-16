package main

import (
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
