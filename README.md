## Getting started
This is a Fluent Bit plugin that will forward your logs to New Relic. Prerequisites include knowledge of Fluent Bit, and a Go environment set up.

You need to initially compile the plugin and store ```out_newrelic.so``` at a location that can be accessed by the fluent-bit daemon.

## Building the plugin
```make all```

## Using the plugin
```fluent-bit -e /path/to/out_newrelic.so -c /etc/fluent-bit/fluent.conf```

## Configuration
We recommend putting the fluent bit configuration in /etc/fluent-bit/fluent.conf


|key           |default  |meaning                               |
|--------------|---------|--------------------------------------|
|apiKey        |  NONE   | insights insert api key              |
|maxBufferSize |  256000 | max size the payloads sent in bytes  |
|maxRecords    |  1024   | max number of records sent           |


Example:
Modify this to suit your needs.  This configuraiton will work for tailing and shipping all logs
in /var/log matching the pattern *.log.
```
[SERVICE]
	Flush          1
        Daemon         Off
        Log_Level      info
[INPUT]
        Name           tail
        Path           /var/log/*.log
        DB             /var/log/flb.db
        Mem_Buf_Limit  5MB
[OUTPUT]
    	Name  newrelic
    	Match *
	apiKey someKey
	maxBufferSize 256000
	maxRecords 1023
```

For more details on the configuration of fluent bit check out the official documentation [here] (https://fluentbit.io/documentation/0.12/configuration/)
