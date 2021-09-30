package nrclient

import (
	"crypto/x509"
	log "github.com/sirupsen/logrus"
)

// since Go 1.8, Go can't properly load the System root certificates on windows.
// For more info, search Golang issues 16736 and 18609
func systemCertPool() *x509.CertPool {
	log.Warn("Can't load load the system root certificates. If you have set up the" +
		" 'caBundleFile' or 'caBundleDir' configuration options, you will need to manually download the New Relic" +
		" site certificate and store it into your CA bundle dir")
	return x509.NewCertPool()
}
