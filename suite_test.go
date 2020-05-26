package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestOutputPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "New Relic Output Plugin Suite")
}
