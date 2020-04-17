package buffer

import (
	"fmt"
	"github.com/newrelic-fluent-bit-output/config"
	"github.com/newrelic-fluent-bit-output/nrclient"
	"math"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Buffer Manager", func() {

	// This lets the matching library (gomega) be able to notify the testing framework (ginkgo)
	gomega.RegisterFailHandler(ginkgo.Fail)

	var server *ghttp.Server
	var endpoint string
	var bufferManager BufferManager
	var nrClient *nrclient.NRClient
	vortexSuccessCode := 202

	BeforeEach(func() {
		server = ghttp.NewServer()
		endpoint = server.URL() + "/v1/logs"

		nrClientConfig := config.NRClientConfig{
			Endpoint: endpoint,
			ApiKey:   "irrelevant",
			// Ideally we shouldn't have to set this separately from insertKey, but where this is set is
			// in the Fluent Bit code that we can't unit test
			UseApiKey: true,
		}
		noProxy := config.ProxyConfig{}

		var err error
		nrClient, _ = nrclient.NewNRClient(nrClientConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}
	})

	AfterEach(func() {
		server.Close()
	})

	It("test buffering by time", func() {
		server.AppendHandlers(ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))

		bufferConfig := config.BufferConfig{
			MaxBufferSize:         256000,
			MaxRecords:            math.MaxInt64,                               // Do not flush by count (we are testing flushing by time)
			MaxTimeBetweenFlushes: int64((1 * time.Second) / time.Millisecond), // Flush after one second
		}
		bufferManager = NewBufferManager(bufferConfig, *nrClient)

		responseChan := bufferManager.AddRecord(make(map[string]interface{}))
		Expect(responseChan).To(BeNil())

		// Wait twice as long as the max time between flushes
		time.Sleep(2 * time.Second)

		// This record doesn't fill the buffer, but we exceed the max time between flushes, so we flush
		responseChan = bufferManager.AddRecord(make(map[string]interface{}))
		Expect(responseChan).ToNot(BeNil())

		<-responseChan
		Expect(bufferManager.shouldSend()).To(BeFalse())
	})

	It("only flushes when buffer is full, then resets buffer", func() {
		server.AppendHandlers(ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""))

		bufferConfig := config.BufferConfig{
			MaxBufferSize:         256000,
			MaxRecords:            2, // Don't send message until we've added two messages
			MaxTimeBetweenFlushes: 5000,
		}
		bufferManager = NewBufferManager(bufferConfig, *nrClient)

		// Add one message, should not send yet
		responseChan := bufferManager.AddRecord(emptyMessage())
		Expect(responseChan).To(BeNil())

		// Add another message, should send
		responseChan = bufferManager.AddRecord(emptyMessage())
		Expect(responseChan).ToNot(BeNil())

		// Check that buffer is cleared after sending
		waitForChannel(responseChan)
		Expect(bufferManager.shouldSend()).To(BeFalse())
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
