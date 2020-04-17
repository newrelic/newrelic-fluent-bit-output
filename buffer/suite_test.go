package buffer

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "testing"
)

func TestBuffer(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "New Relic Out Suite")
}
