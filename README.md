# Fluent Bit Output for New Relic

The **newrelic-fluent-bit-output** plugin forwards output to New Relic.
It works on all versions greater than 0.12 but we recommend 1.X

## Getting started

In order to insert records into New Relic, you can run the plugin from the command line or through the configuration file.

You need to initially compile the plugin and store ```out_newrelic.so``` at a location that can be accessed by the fluent-bit daemon.

## Compiling out_newrelic.so

Prerequisites:
* Fluent Bit
* a Go environment

To build the plugin:
1. Clone [https://github.com/newrelic/newrelic-fluent-bit-output](https://github.com/newrelic/newrelic-fluent-bit-output)
2. Build plugin: `cd newrelic-fluent-bit-output && make all`

## Configuring Fluent Bit

Fluent Bit needs to know the location of the New Relic plugin, and the license key for outputting to New Relic.

It is vitally important to pay attention to white space in your config files. Please use four spaces to indent, and one space between keys and values.

### plugins.conf
Find or create a `plugins.conf` file and add a referencee to out_newrelic.so, adjacent to your `fluent-bit.conf` file.

in plugins.conf
```
[PLUGINS]
    Path /path/to/newrelic-fluent-bit-output/out_newrelic.so
```

### fluent-bit.conf
Modify flient-bit.conf and add the following line under the `[SERVICE]` block:

in fluent-bit.conf
```
[SERVICE]
    # This is the main configuration block for fluent bit.
    # Ensure the follow line exists somewhere in the SERVICE block
    Plugins_File plugins.conf

```

And at the end of `fluent-bit.conf`, add the following to set up input and output filter:
```
[INPUT]
    Name tail
    Path /path/to/your/log/file

[OUTPUT]
    Name newrelic
    Match *
    apiKey <NEW_RELIC_INSIGHTS_INSERT_KEY>

    # Optional
    maxBufferSize 256000
    maxRecords 1024
```

* Restart Fluent Bit: `fluent-bit -c /path/to/fluent-bit.conf`
* Append a test log message to your log file: `echo "test message" >> /path/to/your/log/file`
* Search New Relic Logs for `"test message"`

## Configuration Parameters

The plugin supports the following configuration parameters:


| Key | Description | Default |  
|-----|-------------|---------|  
|apiKey        |  Your New Relic Insights Insert key | NONE   |  
|maxBufferSize |  The maximum size the payloads sent in bytes  | 256000 |  
|maxRecords    |  The maximum number of records to send at a time  | 1024 |   

For information on how to find your New Relic Insights Insert key, take a look at the documentation [here](https://docs.newrelic.com/docs/insights/insights-data-sources/custom-data/send-custom-events-event-api#register).
