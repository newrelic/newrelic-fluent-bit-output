package nrclient

import (
	"crypto/x509"
	log "github.com/sirupsen/logrus"
)

// since Go 1.8, Go can't properly load the System root certificates on windows.
// For more info, search Golang issues 16736 and 18609
func systemCertPool() *x509.CertPool {
	log.Info("Please check our documentation if you need to set up a proxy with self-signed certificates: https://github.com/newrelic/newrelic-fluent-bit-output#windows")
	return x509.NewCertPool()
}
