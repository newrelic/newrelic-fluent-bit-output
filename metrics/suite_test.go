package metrics

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestNRClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics")
}
