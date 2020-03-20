# Developing the plugin

## Getting started

* Install go: `brew install go`

## Developing

* Write tests and production code
* Update the version in version.go
* Run tests: `go test`
* Build the plugin: `make all`
* Run Fluent Bit with the plugin using the template config file: `FILE_PATH=/usr/local/var/log/test.log API_KEY=(your-api-key) BUFFER_SIZE= MAX_RECORDS= fluent-bit -c ./fluent-bit.conf -e ./out_newrelic.so`
* Cause a change that you've configured Fluent Bit to pick up: (`echo "FluentBitTest" >> /usr/local/var/log/test.log`)
* Look in `https://one.newrelic.com/launcher/logger.log-launcher` for your log message ("FluentBitTest")


## Update the kubernetes image version
* Update the image version number in `new-relic-fluent-plugin.yml` in the
 [kubernetes logging repo](https://github.com/newrelic/kubernetes-logging/blob/master/DEVELOPER.md#making-changes)

## Cross compiling in the pipeline

* The pipeline uses a Linux machine to compile the plugins
* Go has a built in way to do [cross compiling](https://github.com/golang/go/wiki/WindowsCrossCompiling)
* To cross compile our plugin we will need the set the `CGO_ENABLED` variable to `1` as we are building a C-shared library. When we set this, Go will also allow us to provide our own C compiler as `CC` and a C++ compiler as `CXX`. 
* In the pipeline script we use [Mingw-w64](http://mingw-w64.org/doku.php/start) as our compiler because it supports both x86 and x64 Windows architectures.