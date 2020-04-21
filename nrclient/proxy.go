package nrclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/newrelic/newrelic-fluent-bit-output/config"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

func buildHttpTransport(cfg config.ProxyConfig, nrUrl string) (*http.Transport, error) {
	proxyResolver, err := getProxyResolver(cfg.IgnoreSystemProxy, cfg.Proxy)
	if err != nil {
		return nil, err
	}

	caCertPool, err := getCertPool(cfg.CABundleFile, cfg.CABundleDir)
	if err != nil {
		return nil, err
	}

	var tlsConfig *tls.Config
	if cfg.CABundleFile != "" || cfg.CABundleDir != "" {
		tlsConfig = &tls.Config{RootCAs: caCertPool}
	}

	transport := &http.Transport{
		Proxy:                 proxyResolver,
		DialContext:           (&net.Dialer{KeepAlive: 600 * time.Second}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   600 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	proxyURL, err := resolveProxyURL(proxyResolver, nrUrl)
	if err != nil {
		return nil, fmt.Errorf("can't determine the proxy URL to be used to contact NR: %v", err)
	}

	if proxyURL != nil && proxyURL.Scheme == "https" && !cfg.ValidateCerts {
		transport.DialTLS = fallbackDialer(transport)
	}

	return transport, nil
}

func getCertPool(certFile string, certDirectory string) (*x509.CertPool, error) {
	caCertPool := systemCertPool()

	if certFile != "" {
		caCert, err := ioutil.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("can't read certificate file %v: %v", certFile, err)
		}

		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			log.Printf("[WARN] certificates from %v could not be appended", certFile)
		}
	}
	if certDirectory != "" {
		files, err := ioutil.ReadDir(certDirectory)
		if err != nil {
			return nil, fmt.Errorf("can't read certificate directory %v: %v", certDirectory, err)
		}

		for _, f := range files {
			if strings.Contains(f.Name(), ".pem") {
				caCertFilePath := filepath.Join(certDirectory + "/" + f.Name())
				caCert, err := ioutil.ReadFile(caCertFilePath)
				if err != nil {
					return nil, fmt.Errorf("can't read certificate file %v: %v", caCertFilePath, err)
				}
				ok := caCertPool.AppendCertsFromPEM(caCert)
				if !ok {
					log.Printf("[WARN] certificate %v could not be appended", caCertFilePath)
				}
			}
		}
	}
	return caCertPool, nil
}

func getProxyResolver(ignoreSystemProxy bool, proxy string) (func(*http.Request) (*url.URL, error), error) {
	if proxy != "" {
		// User-defined proxy
		prUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}

		return http.ProxyURL(prUrl), nil
	} else if !ignoreSystemProxy {
		// Proxy defined via the HTTPS_PROXY (takes precedence) or HTTP_PROXY environment variables
		return http.ProxyFromEnvironment, nil
	} else {
		// No proxy
		return http.ProxyURL(nil), nil
	}
}

func resolveProxyURL (proxyResolver func(*http.Request) (*url.URL, error), nrEndpoint string) (*url.URL, error){
	nrEndpointURL, err := url.Parse(nrEndpoint)
	if err != nil {
		return nil, fmt.Errorf("can't parse the NR logging endpoint, please contact New Relic: %v", err)
	}
	nrUrlRequest := http.Request{
		URL: nrEndpointURL,
	}

	return proxyResolver(&nrUrlRequest)
}

// fallbackDialer implements the transport.Dialer interface to provide backwards compatibility with Go 1.9 proxy
// implementation.
//
// **WARNING** : Don't use this method unless you really have to, as it might lead to an insecure communication between
//               the NewRelic FuentBit output plugin and your HTTPS proxy. Use it at your own risk. In this plugin, it is
//               only used if the user explicitly sets `validateProxyCerts` to `false`.
//
// In versions of go up to Go 1.9, the TLS handshake was not performed, by default, when establishing a secure
// connection. This allowed establishing the HTTPS connection even if the proxy certificates were signed by an unknown
// CA. However, starting from Go 1.10, this verification is performed by default. In order to provide backwards-
// compatibility in the "infra-agent" to those customers that were not performing the TLS handshake and that were using
// self-signed certificates, the fallbackDialer() method was introduced. This method basically uses the "legacy proxy
// implementation" if the 'validateProxyCerts' configuration option is set to false.
//
// fallbackDialer attempts to select the most secure TLS dialer with the following process:
//
// 1. Tries to normally connect to an HTTPS proxy
// 2. If succeeds, uses the normal `tls.Dial` function in further connections (secure option)
// 3. If an Unknown Authority Error is returned, InsecureSkipVerify is set to true and we continue using
//    `tls.Dial` for the following connections.
// 4. If the secure connection is not accepted, we use an unsecured "Go1.9-like" dialer that does not
//    performs the TLS handshake.
func fallbackDialer(transport *http.Transport) func(network string, addr string) (net.Conn, error) {
	return func(network string, addr string) (conn net.Conn, e error) {
		// test the tlsDialer with normal configuration
		log.Printf("[DEBUG] dialing with usual, secured configuration")
		dialer := tlsDialer(transport)
		conn, err := dialer(network, addr)
		if err == nil {
			log.Printf("[DEBUG] usual, secured configuration worked as expected. Defaulting to it")
			// if worked, we will use tlsDialer directly from now on
			transport.DialTLS = dialer
			return conn, err
		}
		switch err.(type) {
		case x509.UnknownAuthorityError:
			log.Printf("[DEBUG] usual, secured configuration did not work as expected (%v). Retrying with verification skip", err)
			// if in the previous request we received an authority error, we skip verification and
			// continue using tlsDialer directly from now on
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{}
			}
			transport.TLSClientConfig.InsecureSkipVerify = true

			// we will use tlsDialer directly from now on, with the insecure skip configuration
			transport.DialTLS = tlsDialer(transport)
			return transport.DialTLS(network, addr)
		case tls.RecordHeaderError:
			log.Printf("[DEBUG] usual, secured configuration did not work as expected (%v). Retrying with HTTP dialing", err)
			// if the problem was due to a non-https connection, we use a non-tls dialer directly
			// from now on
			transport.DialTLS = nonTLSDialer
			return transport.DialTLS(network, addr)
		default:
			return conn, err
		}
	}
}

// tlsDialer wraps the standard library tls.Dial function
func tlsDialer(transport *http.Transport) func(network string, addr string) (net.Conn, error) {
	return func(network string, addr string) (conn net.Conn, e error) {
		return tls.Dial(network, addr, transport.TLSClientConfig)
	}
}

// nonTlsDial mimics the tls.DialWithDialer function, but without performing TLS handshakes
func nonTLSDialer(network, addr string) (net.Conn, error) {
	dialer := new(net.Dialer)
	// We want the Timeout and Deadline values from dialer to cover the
	// whole process: TCP connection and TLS handshake. This means that we
	// also need to start our own timers now.
	timeout := dialer.Timeout

	if !dialer.Deadline.IsZero() {
		deadlineTimeout := time.Until(dialer.Deadline)
		if timeout == 0 || deadlineTimeout < timeout {
			timeout = deadlineTimeout
		}
	}

	var errChannel chan error

	if timeout != 0 {
		errChannel = make(chan error, 2)
		time.AfterFunc(timeout, func() {
			errChannel <- timeoutError{}
		})
	}

	return dialer.Dial(network, addr)
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "tls: DialWithDialer timed out" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
