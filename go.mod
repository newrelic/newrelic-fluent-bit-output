module github.com/newrelic/newrelic-fluent-bit-output

// We can't go past 1.20.X until this issue is solved: https://github.com/golang/go/issues/62130#issuecomment-1687335898
go 1.20

require (
	github.com/fluent/fluent-bit-go v0.0.0-20200729034236-b9c0d6a20853
	github.com/newrelic/newrelic-telemetry-sdk-go v0.8.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	golang.org/x/net v0.0.0-20180906233101-161cd47e91fd // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.3.8 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
