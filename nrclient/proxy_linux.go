package nrclient

import (
	"crypto/x509"
	"log"
)

func systemCertPool() *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		log.Printf("[WARNING] Can't load load the system root certificates. If you have set up the" +
			" 'ca_bundle_file' or 'ca_bundle_dir' configuration options, you will need to manually download the New Relic" +
			" site certificate and store it into your CA bundle dir")
		pool = x509.NewCertPool()
	}
	return pool
}
