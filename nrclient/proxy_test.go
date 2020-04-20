package nrclient

import (
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"net/http"
	"net/url"
	"os"

	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP Proxy", func() {

	// This lets the matching library (gomega) be able to notify the testing framework (ginkgo)
	gomega.RegisterFailHandler(ginkgo.Fail)

	const configuredProxy = "https://user:password@hostname:8888"
	configuredProxyURL := url.URL{
		Scheme: "https",
		User:   url.UserPassword("user", "password"),
		Host:   "hostname:8888",
	}

	const httpEnvironmentProxy = "http://envuser:envpassword@envhostname:8888"
	httpEnvironmentProxyURL := url.URL{
		Scheme: "http",
		User:   url.UserPassword("envuser", "envpassword"),
		Host:   "envhostname:8888",
	}

	const httpsEnvironmentProxy = "https://envssluser:envsslpassword@envsslhostname:9999"
	httpsEnvironmentProxyURL := url.URL{
		Scheme: "https",
		User:   url.UserPassword("envssluser", "envsslpassword"),
		Host:   "envsslhostname:9999",
	}

	dummyHTTPRequest := http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "someserver:1234",
		},
	}
	dummyHTTPSRequest := http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   "someserver:1234",
		},
	}

	var originalHTTPProxy string
	var originalHTTPSProxy string

	BeforeSuite(func() {
		originalHTTPProxy = os.Getenv("HTTP_PROXY")
		originalHTTPSProxy = os.Getenv("HTTPS_PROXY")
		os.Setenv("HTTP_PROXY", httpEnvironmentProxy)
		os.Setenv("HTTPS_PROXY", httpsEnvironmentProxy)
	})

	AfterSuite(func() {
		os.Setenv("HTTP_PROXY", originalHTTPProxy)
		os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
	})

	It("uses the environment HTTP proxy for HTTP requests", func() {
		const ignoreSystemProxy = false

		proxyProvider, err := getProxyResolver(ignoreSystemProxy, "")
		Expect(err).To(BeNil())
		proxyURL, err := proxyProvider(&dummyHTTPRequest)
		Expect(err).To(BeNil())

		Expect(proxyURL).To(Not(BeNil()))
		Expect(*proxyURL).To(Equal(httpEnvironmentProxyURL))
	})

	It("uses the environment HTTPS proxy for HTTPS requests (takes precedence)", func() {
		const ignoreSystemProxy = false

		proxyProvider, err := getProxyResolver(ignoreSystemProxy, "")
		Expect(err).To(BeNil())
		proxyURL, err := proxyProvider(&dummyHTTPSRequest)
		Expect(err).To(BeNil())

		Expect(proxyURL).To(Not(BeNil()))
		Expect(*proxyURL).To(Equal(httpsEnvironmentProxyURL))
	})

	It("ignores the environment HTTP and HTTPS proxies when the user uses ignoreSystemProxy (no proxy if none defined by the user)", func() {
		const ignoreSystemProxy = true

		proxyProvider, err := getProxyResolver(ignoreSystemProxy, "")
		Expect(err).To(BeNil())
		proxyURL, err := proxyProvider(&dummyHTTPRequest)
		Expect(err).To(BeNil())

		Expect(proxyURL).To(BeNil())
	})

	It("uses the user-provided proxy, which takes precedence over the ones defined via environment variables", func() {
		const ignoreSystemProxy = false

		proxyProvider, err := getProxyResolver(ignoreSystemProxy, configuredProxy)
		Expect(err).To(BeNil())
		proxyURL, err := proxyProvider(&dummyHTTPRequest)
		Expect(err).To(BeNil())

		Expect(proxyURL).To(Not(BeNil()))
		Expect(*proxyURL).To(Equal(configuredProxyURL))
	})
})
