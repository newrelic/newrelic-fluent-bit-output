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
