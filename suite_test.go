package main

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    "testing"
)

func TestOut(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "New Relic Out Suite")
}
