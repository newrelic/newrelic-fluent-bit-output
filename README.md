# Fluent Bit Output for New Relic


The **newrelic-fluent-bit-output** plugin forwards output to New Relic.

It works on all versions of Fluent Bit greater than 0.12 but for the best experience we recommend using versions greater than 1.0.

## Getting started 
In order to insert records into New Relic, you can configure the plugin with a config file or configure it via command line flags.
* [Link to configuration](https://docs.fluentbit.io/manual/configuration)

You need to compile the plugin and store ```out_newrelic.so``` at a location that can be accessed by the fluent-bit daemon.

Note that for certain Linux Enterprise users, 
[including CentOS 7, Debian 8 and 9, Ubuntu, and Raspbian 8](https://fluentbit.io/documentation/0.13/installation/td-agent-bit.html), 
the name of Fluent Bit is td-agent-bit, instead of fluent-bit. So, anywhere in this doc where it says `fluent-bit`, 
just replace it with `td-agent-bit` (for example, you will need to edit `td-agent-bit.conf` instead of `fluent-bit.conf`).

## Compiling out_newrelic.so

This project is provided AS-IS WITHOUT WARRANTY OR SUPPORT, although you can report issues and contribute to the project here on GitHub.

Prerequisites:
* Fluent Bit
* a Go environment

To build the plugin:
1. Clone [https://github.com/newrelic/newrelic-fluent-bit-output](https://github.com/newrelic/newrelic-fluent-bit-output)
2. Build plugin: `cd newrelic-fluent-bit-output && make all`

## Configuring Fluent Bit

Fluent Bit needs to know the location of the New Relic plugin, and the license key for outputting to New Relic.

It is vitally important to pay attention to white space in your config files. Please use four spaces to indent, 
and one space between keys and values.

### plugins.conf
Find or create a `plugins.conf` file in your Fluent Bit directory and add a reference to out_newrelic.so,
adjacent to your `fluent-bit.conf` file.

in plugins.conf
```
[PLUGINS]
    Path /path/to/newrelic-fluent-bit-output/out_newrelic.so
```

### fluent-bit.conf
Modify fluent-bit.conf and add the following line under the `[SERVICE]` block:

in fluent-bit.conf
```
[SERVICE]
    # This is the main configuration block for fluent bit.
    # Ensure the follow line exists somewhere in the SERVICE block
    Plugins_File plugins.conf
```

And at the end of `fluent-bit.conf`, add the following to set up the input and output filter:
```
[INPUT]
    Name tail
    Path /path/to/your/log/file

[OUTPUT]
    Name newrelic
    Match *
    apiKey <NEW_RELIC_INSIGHTS_INSERT_KEY>
```

* Restart Fluent Bit: `fluent-bit -c /path/to/fluent-bit.conf`
* Append a test log message to your log file: `echo "test message" >> /path/to/your/log/file`
* Search New Relic Logs for `"test message"`

## Configuration Parameters

The plugin supports the following configuration parameters and include either an Insights or License Key:

| Key | Description | Default |  
|-----|-------------|---------|  
|endpoint      |  The endpoint you send data to |  `https://log-api.newrelic.com/log/v1` |
|apiKey        |  Your New Relic Insights Insert key | (none)   |  
|licenseKey    |  Your New relic License key         | (none)   |
|maxBufferSize |  The maximum size the payloads sent in bytes  | 256000 |  
|maxRecords    |  The maximum number of records to send at a time  | 1024 |   

For information on how to find your New Relic Insights Insert key, take a look at the 
documentation [here](https://docs.newrelic.com/docs/insights/insights-data-sources/custom-data/send-custom-events-event-api#register).

## Eu Configuration

Set `endpoint` to `https://log-api.eu.newrelic.com/log/v1`.

## Docker Container Configuration

This plugin comes with a Dockerfile and sample config that will let you get started with the plugin fairly easily.

### Environment Variables
| Key | Description | Required |
|-----|-------------| -----|
|API_KEY| Your New Relic Insights Insert Key | Yes |
|FILE_PATH   |  A path or glob to the file or files you wish to tail | Yes |
|BUFFER_SIZE |  The max size for json payload in bytes | No |
|MAX_RECORDS  |  The number of records to send in a payload | No |

### Docker Example
Within the root of the project run the following. You can supplement the image name and tag as you see fit.

```
docker build -t <YOUR-IMAGE-NAME>:<YOUR-TAG> .
docker run -e "FILE_PATH=/var/log/*" -e "API_KEY=<YOUR-API-KEY>" <YOUR-IMAGE-NAME>:<YOUR-TAG> 
```

