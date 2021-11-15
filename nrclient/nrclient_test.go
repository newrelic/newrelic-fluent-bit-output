package nrclient

import (
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-fluent-bit-output/record"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"net/http"
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
	vortexSuccessCode := 202
	vortexServerErrorCode := 500
	logRecords := []record.LogRecord{
		{
			"timestamp": 1,
			"message":   "Some message 1",
		},
		{
			"timestamp": 2,
			"message":   "Some message 2",
		},
	}

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

	It("Makes no HTTP call when a nil slice is provided", func() {
		// Given
		nrClient, err := NewNRClient(licenseKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		statusCode, err := nrClient.Send(nil)

		// Then
		Expect(statusCode).To(Equal(http.StatusAccepted))
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(0))
	})

	It("Makes no HTTP call when no records are provided", func() {
		// Given
		nrClient, err := NewNRClient(licenseKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When

		statusCode, err := nrClient.Send([]record.LogRecord{})

		// Then
		Expect(statusCode).To(Equal(http.StatusAccepted))
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(0))
	})

	It("Makes the expected HTTP call with api key", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""),
				ghttp.VerifyRequest("POST", "/v1/logs"),
				ghttp.VerifyHeader(http.Header{
					"X-Insert-Key":     []string{insertKey},
					"Content-Type":     []string{"application/json"},
					"Content-Encoding": []string{"gzip"},
				})))

		nrClient, err := NewNRClient(insertKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		statusCode, err := nrClient.Send(logRecords)

		// Then
		Expect(statusCode).To(Equal(http.StatusAccepted))
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Makes the expected HTTP call with License Key", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexSuccessCode, ""),
				ghttp.VerifyRequest("POST", "/v1/logs"),
				ghttp.VerifyHeader(http.Header{
					"X-License-Key":    []string{licenseKey},
					"Content-Type":     []string{"application/json"},
					"Content-Encoding": []string{"gzip"},
				})))

		nrClient, err := NewNRClient(licenseKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		statusCode, err := nrClient.Send(logRecords)

		// Then
		Expect(statusCode).To(Equal(http.StatusAccepted))
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Returns status code when request fails", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&vortexServerErrorCode, ""),
				ghttp.VerifyRequest("POST", "/v1/logs"),
				ghttp.VerifyHeader(http.Header{
					"X-License-Key":    []string{licenseKey},
					"Content-Type":     []string{"application/json"},
					"Content-Encoding": []string{"gzip"},
				})))

		nrClient, err := NewNRClient(licenseKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		statusCode, err := nrClient.Send(logRecords)

		// Then
		Expect(statusCode).To(Equal(vortexServerErrorCode))
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})
})
