package metrics

import (
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("metrics", func() {
	It("Creates a no-op client if sendMetrics is false", func() {
		nrClientConfig := config.NRClientConfig{
			SendMetrics: false,
		}

		metricsClient, err := NewClient(nrClientConfig)

		Expect(err).To(BeNil())
		Expect(metricsClient).To(BeAssignableToTypeOf(&noopMetricAggregator{}))
	})

	It("Returns a noop metricsClient and an error if an invalid Logs URL is provided and sendMetrics is enabled", func() {
		nrClientConfig := config.NRClientConfig{
			Endpoint:    "invalidOnPurpose",
			SendMetrics: true,
		}

		metricsClient, err := NewClient(nrClientConfig)

		Expect(err).NotTo(BeNil())
		Expect(metricsClient).To(BeAssignableToTypeOf(&noopMetricAggregator{}))
	})

	It("Returns a no-op metricsClient and a nil error if an invalid Logs URL is provided but sendMetrics is disabled", func() {
		nrClientConfig := config.NRClientConfig{
			Endpoint:    "willBeIgnored",
			SendMetrics: false,
		}

		metricsClient, err := NewClient(nrClientConfig)

		Expect(err).To(BeNil())
		Expect(metricsClient).To(BeAssignableToTypeOf(&noopMetricAggregator{}))
	})

	It("Returns a real metricsClient and a nil error if a valid Logs URL is provided and sendMetrics is enabled", func() {
		nrClientConfig := config.NRClientConfig{
			Endpoint:    "https://log-api.newrelic.com/log/v1",
			LicenseKey:  "dummy",
			SendMetrics: true,
		}

		metricsClient, err := NewClient(nrClientConfig)

		Expect(err).To(BeNil())
		Expect(metricsClient).To(BeAssignableToTypeOf(&wrappedMetricAggregator{}))
	})
})
