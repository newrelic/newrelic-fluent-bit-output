package nrclient

import (
	"bytes"
	"fmt"
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("NR Client", func() {

	// This lets the matching library (gomega) be able to notify the testing framework (ginkgo)
	gomega.RegisterFailHandler(ginkgo.Fail)

	var server *ghttp.Server
	const insertKey = "some-insert-key"
	const licenseKey = "some-license-key"
	var endpoint string
	var insertKeyConfig config.NRClientConfig
	var licenseKeyConfig config.NRClientConfig
	var noProxy config.ProxyConfig
	var nrClient *NRClient
	vortexSuccessCode := 202
	var emptyMessage bytes.Buffer

	BeforeEach(func() {
		server = ghttp.NewServer()
		endpoint = server.URL() + "/v1/logs"

		insertKeyConfig = config.NRClientConfig{
			Endpoint: endpoint,
			ApiKey:   insertKey,
			// Ideally we shouldn't have to set this separately from insertKey, but where this is set is
			// in the Fluent Bit code that we can't unit test
			UseApiKey: true,
		}

		licenseKeyConfig = config.NRClientConfig{
			Endpoint:   endpoint,
			LicenseKey: licenseKey,
			// Ideally we shouldn't have to set this separately from licenseKey, but where this is set is
			// in the Fluent Bit code that we can't unit test
			UseApiKey: false,
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

		var err error
		nrClient, err = NewNRClient(insertKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}
		responseChan := make(chan *http.Response, 1)
		nrClient.Send(&emptyMessage, responseChan)

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

		var err error
		nrClient, err = NewNRClient(licenseKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}
		responseChan := make(chan *http.Response, 1)
		nrClient.Send(&emptyMessage, responseChan)

		// Wait for message to be sent
		Expect(responseChan).ToNot(BeNil())
		waitForChannel(responseChan)
	})
})

func waitForChannel(responseChan chan *http.Response) {
	maximumWaitInSeconds := 10
	maximumWait := time.Duration(maximumWaitInSeconds) * time.Second
	select {
	case <-responseChan:
	case <-time.After(maximumWait):
		Fail(fmt.Sprintf("Channel was not written to within %d seconds", maximumWaitInSeconds))
	}
}
