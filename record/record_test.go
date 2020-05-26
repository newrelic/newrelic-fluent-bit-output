package record

import (
	"fmt"
	"os"
	"time"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
)

var _ = Describe("Out New Relic", func() {
	const pluginVersion = "0.0.0"

	// This lets the matching library (gomega) be able to notify the testing framework (ginkgo)
	gomega.RegisterFailHandler(ginkgo.Fail)

	Describe("Prepares payload", func() {
		AfterEach(func() {
			os.Unsetenv("SOURCE")
		})

		It("converts the map[interface{}] inteface{} to map[string] interface[], "+
			"updates the timestamp, and renames the log field to message",
			func() {
				inputMap := make(FluentBitRecord)
				var inputTimestamp interface{}
				inputTimestamp = output.FLBTime{
					time.Now(),
				}
				inputMap["log"] = "message"
				foundOutput := RemapRecord(inputMap, inputTimestamp, pluginVersion)
				Expect(foundOutput["message"]).To(Equal("message"))
				Expect(foundOutput["log"]).To(BeNil())
				Expect(foundOutput["timestamp"]).To(Equal(inputTimestamp.(output.FLBTime).UnixNano() / 1000000))
				pluginMap := foundOutput["plugin"].(map[string]string)
				typeVal := pluginMap["type"]
				version := pluginMap["version"]
				source := pluginMap["source"]
				Expect(typeVal).To(Equal("fluent-bit"))
				Expect(version).To(Equal(pluginVersion))
				Expect(source).To(Equal("BARE-METAL"))
			},
		)
		It("sets the source if it is included as an environment variable",
			func() {
				inputMap := make(FluentBitRecord)
				var inputTimestamp interface{}
				inputTimestamp = output.FLBTime{
					time.Now(),
				}
				expectedSource := "docker"
				inputMap["log"] = "message"
				os.Setenv("SOURCE", expectedSource)
				foundOutput := RemapRecord(inputMap, inputTimestamp, pluginVersion)
				pluginMap := foundOutput["plugin"].(map[string]string)
				Expect(pluginMap["source"]).To(Equal(expectedSource))
			},
		)

		It("Correctly massage nested map[interface]interface{} to map[string]interface{}",
			func() {
				inputMap := make(FluentBitRecord)
				nestedMap := make(map[interface{}]interface{})
				expectedOutput := make(LogRecord)
				expectedNestedOutput := make(LogRecord)
				expectedNestedOutput["foo"] = "bar"
				expectedOutput["nested"] = expectedNestedOutput
				nestedMap["foo"] = "bar"
				inputMap["nested"] = nestedMap
				foundOutput := parseRecord(inputMap)
				Expect(foundOutput).To(Equal(expectedOutput))

			},
		)
	})

	Describe("Timestamp handling", func() {

		inputTimestampToExpectedOutput := map[interface{}]int64{
			// Modern Fluent Bit does uses FLBTime
			output.FLBTime{time.Unix(1234567890, 123456789)}: 1234567890123,

			// We've seen older of Fluent Bit versions use uint64
			// (generally being sent in seconds, but we handle other granularities out of paranoia)
			uint64(1234567890):          1234567890000,
			uint64(1234567890123):       1234567890123,
			uint64(1234567890123456):    1234567890123,
			uint64(1234567890123456789): 1234567890123,
		}

		for inputTimestamp, expectedOutputTime := range inputTimestampToExpectedOutput {
			// Lock in current values (otherwise all tests will run with the last values in the map)
			input := inputTimestamp
			expected := expectedOutputTime

			It("handles timestamps of various types and granularites : "+fmt.Sprintf("%v", input),
				func() {
					inputMap := make(FluentBitRecord)

					foundOutput := RemapRecord(inputMap, input, pluginVersion)

					Expect(foundOutput["timestamp"]).To(Equal(expected))
				},
			)
		}

		It("ignores timestamps of unhandled types",
			func() {
				inputMap := make(FluentBitRecord)

				// We don't handle string types
				foundOutput := RemapRecord(inputMap, "1234567890", pluginVersion)

				Expect(foundOutput["timestamp"]).To(BeNil())
			},
		)
	})
})
