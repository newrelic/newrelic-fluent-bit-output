package nrclient

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"github.com/newrelic/newrelic-fluent-bit-output/record"
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
	httpSuccessCode := 202
	httpRetryableErrorCode := 500
	httpNonRetryableErrorCode := 345
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
			UseApiKey:      true,
			TimeoutSeconds: 2,
		}

		licenseKeyConfig = config.NRClientConfig{
			Endpoint:   endpoint,
			LicenseKey: licenseKey,
			// Ideally we shouldn't have to set this separately from licenseKey, but where this is set is
			// in the Fluent Bit code that we can't unit test
			UseApiKey:      false,
			TimeoutSeconds: 2,
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
		shouldRetry, err := nrClient.Send(nil)

		// Then
		Expect(shouldRetry).To(BeFalse())
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

		shouldRetry, err := nrClient.Send([]record.LogRecord{})

		// Then
		Expect(shouldRetry).To(BeFalse())
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(0))
	})

	It("Makes the expected HTTP call with api key", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&httpSuccessCode, ""),
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
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeFalse())
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Makes the expected HTTP call with License Key", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&httpSuccessCode, ""),
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
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeFalse())
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Returns retry=true when status code is included in the retryable list", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&httpRetryableErrorCode, ""),
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
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeTrue())
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Returns retry=false when status code is NOT included in the retryable list", func() {
		// Given
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&httpNonRetryableErrorCode, ""),
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
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeFalse())
		Expect(err).To(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Returns retry=true and the original error when a timeout happens", func() {
		// Given
		server.RouteToHandler("POST", "/v1/logs", func(http.ResponseWriter, *http.Request) {
			// Timeout is set to 2 seconds, so this will cause a timeout.
			time.Sleep(4 * time.Second)
		})

		nrClient, err := NewNRClient(licenseKeyConfig, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeTrue())
		Expect(err).NotTo(BeNil())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
	})

	It("Returns retry=true and the original error when a non-resolvable host name is provided", func() {
		// Given
		configWithWrongEndpoint := config.NRClientConfig{
			Endpoint:   "https://unresolvable.host.name/v1/logs",
			LicenseKey: licenseKey,
			// Ideally we shouldn't have to set this separately from licenseKey, but where this is set is
			// in the Fluent Bit code that we can't unit test
			UseApiKey:      false,
		}

		nrClient, err := NewNRClient(configWithWrongEndpoint, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeTrue())
		Expect(err).NotTo(BeNil())
		// The server is never called in this test, since the host name is unresolvable
		Expect(server.ReceivedRequests()).To(HaveLen(0))
	})

	It("Returns retry=true and the original error when an existing server is hit using the wrong port", func() {
		// Given
		serverUrl, err := url.Parse(server.URL())
		if err != nil {
			Fail("Could not parse server URL")
		}
		host, _, _ := net.SplitHostPort(serverUrl.Host)
		configWithWrongEndpoint := config.NRClientConfig{
			Endpoint:   serverUrl.Scheme + "://" + host + ":666/v1/logs",
			LicenseKey: licenseKey,
			// Ideally we shouldn't have to set this separately from licenseKey, but where this is set is
			// in the Fluent Bit code that we can't unit test
			UseApiKey:      false,
		}

		nrClient, err := NewNRClient(configWithWrongEndpoint, noProxy)
		if err != nil {
			Fail("Could not initialize the NRClient")
		}

		// When
		shouldRetry, err := nrClient.Send(logRecords)

		// Then
		Expect(shouldRetry).To(BeTrue())
		Expect(err).NotTo(BeNil())
		// The server is never called in this test, since the host name is unresolvable
		Expect(server.ReceivedRequests()).To(HaveLen(0))
	})
})
