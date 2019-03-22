package main

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Out New Relic", func() {

  Describe("Validation of config", func() {
    It("should fail to load when required parameters are not provided", func() {

    })
  })

  Describe("HTTP Request headers", func() {
    It("Insert key should be set", func() {
      // ‘X-Insert-Key’	Is set to Insert API Key
    })

    It("Event source should be correct", func(){
      // ‘X-Event-Source’	Is set to ‘logs’
    })

    It("Content encodint should be correct", func(){
      // ‘Content-Encoding’	Is set to ‘gzip’
    })
  })

  Describe("HTTP Request body", func(){
    It("should have a gzipped json array of size 1 with timestamp field added, when there's a single event with just a message field", func(){

    })

    It("should have a gzipped json array size 1 with all fields when theres a single event with multiple fields including message and timestamp", func(){

    })

    It("should have a gzipped json array with all events with all fields when theres a single event with multiple fields including message and timestamp", func(){

    })

  })

  Describe("Retry logic", func(){
    It("should send a POST to the Insights collector if initial call results in 200", func(){

    })

    It("should retry if initial call results in 500", func(){

    })

    It("should retry but give up eventually if all calls result in 500", func(){

    })
  })

  Describe("Retry delay calculation", func(){
    It("should retry double each time up to a maximum if there's a non-zero delay", func() {
      
    })
  })

})



Retry delay calculation
 	Non-zero delay	Retries double each time up to maximum
