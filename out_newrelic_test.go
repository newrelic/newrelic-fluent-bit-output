package main

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Out New Relic", func() {

	Describe("HTTP Request body", func() {
		var server *ghttp.Server
		const expectedInsertKey = "sweetKey"
		var expectedEndpoint string
		BeforeEach(func() {
			server = ghttp.NewServer()
			expectedEndpoint = server.URL() + "/v1/logs"
			server.AppendHandlers(
				ghttp.CombineHandlers(ghttp.VerifyRequest("POST", "/v1/logs"),
					ghttp.VerifyHeader(http.Header{
						"X-Insert-Key":     []string{expectedEndpoint},
						"Content-Type":     []string{"application/json"},
						"Content-Encoding": []string{"gzip"},
					}),
				))
		})

		AfterEach(func() {
			server.Close()
		})
		It("correctly packages and posts json", func() {

			var testConfig = PluginConfig{
				apiKey:        expectedInsertKey,
				endpoint:      expectedEndpoint,
				maxBufferSize: 256000,
				maxRecords:    2,
			}

			var testRecords []map[string]interface{}
			var testRecord map[string]interface{}

			testRecord = make(map[string]interface{})
			testRecord["timestamp"] = time.Now().UnixNano() / int64(time.Millisecond)
			testRecord["message"] = "cool story"
			testRecords = append(testRecords, testRecord)
			prepare(testRecords, &testConfig)
		})
	})
})
