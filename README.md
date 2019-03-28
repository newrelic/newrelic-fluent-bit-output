The **newrelic-fluent-bit-output** plugin forwards output to New Relic.
It works on all versions greater than 0.12 but we recommend 1.X

## Getting started

In order to insert records into New Relic, you can run the plugin from the command line or through the configuration file.

You need to initially compile the plugin and store ```out_newrelic.so``` at a location that can be accessed by the fluent-bit daemon.

## Building the plugin

Prerequisites:
* Fluent Bit
* a Go environment

To build the plugin:
* Install dependencies: `go get github.com/fluent/fluent-bit-go/output`
* Build plugin: `make all`

## Configuration Parameters

The plugin supports the following configuration parameters:

|Key           |Description |Default                               |
|--------------|---------|--------------------------------------|
|apiKey        |  Your New Relic API Insert key |NONE   | 
|maxBufferSize |  The maximum size the payloads sent in bytes  |256000 | 
|maxRecords    |  The maximum number of records to send at a time  |1024   | 

Example:
```
[OUTPUT]
        Name            newrelic
        Match           *
        apiKey          <API_INSERT_KEY>
```

## Testing

* Add the following block to your Fluent Bit configuration file (with your specific API Insert key)

```
        [INPUT]
                Name           tail
                Path           /path/to/your/log/file
        [OUTPUT]
                Name            newrelic
                Match           *
                apiKey          <API_INSERT_KEY>
```

* Restart Fluent Bit: `fluent-bit -e /path/to/out_newrelic.so -c /path/to/fluent.conf`
* Append a test log message to your log file: `echo "test message" >> /path/to/your/log/file`
* Search New Relic Logs for `"test message"`
