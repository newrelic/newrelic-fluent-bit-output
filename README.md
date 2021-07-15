[![Community Plus header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Plus.png)](https://opensource.newrelic.com/oss-category/#community-plus)

# Fluent Bit Output for New Relic

The **newrelic-fluent-bit-output** plugin forwards output to New Relic.

It works on all versions of Fluent Bit greater than 0.12 but for the best experience we recommend using versions greater than 1.0.

## Getting started

In order to insert records into New Relic, you can configure the plugin with a config file or configure it via command line flags.

- [Link to configuration](https://docs.fluentbit.io/manual/configuration)

You can download the output plugin pre-compiled under our [releases](https://github.com/newrelic/newrelic-fluent-bit-output/releases/latest).
Alternatively you can compile the plugin yourself and store `out_newrelic.so` or `out_newrelic_winXX.dll` at a location that can be accessed by the fluent-bit daemon.

Note that for certain Linux Enterprise users,
[including CentOS 7, Debian 8 and 9, Ubuntu, and Raspbian 8](https://fluentbit.io/documentation/0.13/installation/td-agent-bit.html),
the name of Fluent Bit is td-agent-bit, instead of fluent-bit. So, anywhere in this doc where it says `fluent-bit`,
just replace it with `td-agent-bit` (for example, you will need to edit `td-agent-bit.conf` instead of `fluent-bit.conf`).

## Compiling the out_newrelic plugin

This project is provided AS-IS WITHOUT WARRANTY OR SUPPORT, although you can report issues and contribute to the project here on GitHub.

Prerequisites:

- Fluent Bit
- a Go environment

To build the plugin:

1. Clone [https://github.com/newrelic/newrelic-fluent-bit-output](https://github.com/newrelic/newrelic-fluent-bit-output)
2. Build plugin: `cd newrelic-fluent-bit-output && go get github.com/fluent/fluent-bit-go/output && make {OS}`
   1. `make all` for linux/unix
   2. `make win32` for 32-bit windows
   3. `make win64` for 64-bit windows

## Configuring Fluent Bit

Fluent Bit needs to know the location of the New Relic plugin, and the license key for outputting to New Relic.

It is vitally important to pay attention to white space in your config files. Please use four spaces to indent,
and one space between keys and values.

### plugins.conf

Find or create a `plugins.conf` file in your Fluent Bit directory and add a reference to out_newrelic.so or out_newrelic_winXX.dll,
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
    licenseKey <NEW_RELIC_LICENSE_KEY>

[FILTER]
    Name modify
    Match *
    Add hostname <HOSTNAME>
    Add service_name <SERVICE_NAME>
```

- Restart Fluent Bit: `fluent-bit -c /path/to/fluent-bit.conf`
- Append a test log message to your log file: `echo "test message" >> /path/to/your/log/file`
- Search New Relic Logs for `"test message"`

## Proxy support

The plugin automatically detects the `HTTP_PROXY` and `HTTPS_PROXY` environment variables, and automatically uses them to set up the proxy configuration.

If you want to bypass the system-wide defined proxy for some reason, you can use the `ignoreSystemProxy` configuration parameter.

You can also specify a custom proxy to send the logs to (different from the system-wide defined) by using the `proxy` configuration parameter.

HTTPS proxies (having an `https://...` URL) use a certificate to encrypt the connection between the plugin and the proxy. If you are using a self-signed certificate (not trusted by the Certification Authorities defined at your system level), you can:

- Windows: import the self-signed certificate (PEM file) using the MMC tool. You can refer to [this link](https://www.ssls.com/knowledgebase/how-to-import-intermediate-and-root-certificates-via-mmc/), but in Step 2 ensure to import it in your "Trusted Root Certification Authorities" instead of importing it in the "Intermediate Certification Authorities".
- Linux: you can specify the self-signed certificate (PEM file) using either the `caBundleFile` or `caBundleDir` parameters (see next section).

Optionally, you can skip the self-signed certificate verification by setting `validateProxyCerts` to `false`, but please note that this option is not considered safe due to potential Man In The Middle Attacks.

A example setup, which defines an HTTPS proxy and its self-signed certificate, would result in the following configuration:

```
[OUTPUT]
    Name newrelic
    Match *
    licenseKey <NEW_RELIC_LICENSE_KEY>
    proxy https://https-proxy-hostname:3129
    # REMOVE following option when using it on Windows (see above)
    caBundleFile /path/to/proxy-certificate-bundle.pem
```

## Configuration Parameters

The plugin supports the following configuration parameters and include either an Insights or License Key:

| Key                | Description                                                                                                                                                                                                                                                                                                                                                                                                              | Default                               |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------- |
| endpoint           | The endpoint you send data to                                                                                                                                                                                                                                                                                                                                                                                            | `https://log-api.newrelic.com/log/v1` |
| apiKey             | Your New Relic Insights Insert key                                                                                                                                                                                                                                                                                                                                                                                       | (none)                                |
| licenseKey         | Your New Relic License key                                                                                                                                                                                                                                                                                                                                                                                               | (none)                                |
| maxBufferSize      | **[Deprecated since 1.3.0]** The maximum size the payloads sent in bytes                                                                                                                                                                                                                                                                                                                                                 | 256000                                |
| maxRecords         | **[Deprecated since 1.3.0]** The maximum number of records to send at a time                                                                                                                                                                                                                                                                                                                                             | 1024                                  |
| proxy              | Optional proxy to communicate with New Relic, overrides any environment-defined one. Must follow the format `https://user:password@hostname:port`. Can be HTTP or HTTPS.                                                                                                                                                                                                                                                 | (none)                                |
| ignoreSystemProxy  | Ignore any proxy defined via the `HTTP_PROXY` or `HTTPS_PROXY` environment variables. Note that if a proxy has been defined using the `proxy` parameter, this one has no effect.                                                                                                                                                                                                                                         | false                                 |
| caBundleFile       | **[LINUX HTTPS ONLY]** Specifies the Certificate Authority certificate to use for validating HTTPS connections against the proxy. Useful when the proxy uses a self-signed certificate. **The certificate file must be in the PEM format**. If not specified, then the operating system's CA list is used. Only used when `validateProxyCerts` is `true`.                                                                | (none)                                |
| caBundleDir        | **[LINUX HTTPS ONLY]** Specifies a folder containing one or more Certificate Authority certificates ot use for validating HTTPS connections against the proxy. Useful when the proxy uses a self-signed certificate. **Only certificate files in the PEM format and \*.pem extension will be considered**. If not specified, then the operating system's CA list is used. Only used when `validateProxyCerts` is `true`. | (none)                                |
| validateProxyCerts | **[HTTPS ONLY]** When using a HTTPS proxy, the proxy certificates are validated by default when establishing a HTTPS connection. To disable the proxy certificate validation, set `validateProxyCerts` to `false` (insecure)                                                                                                                                                                                             | true                                  |

For information on how to find your New Relic Insights Insert key, take a look at the
documentation [here](https://docs.newrelic.com/docs/insights/insights-data-sources/custom-data/send-custom-events-event-api#register).

## Eu Configuration

Set `endpoint` to `https://log-api.eu.newrelic.com/log/v1`.

## Docker Container Configuration

This plugin comes with a Dockerfile and sample config that will let you get started with the plugin fairly easily.

### Environment Variables

| Key         | Description                                          | Required                                        |
| ----------- | ---------------------------------------------------- | ----------------------------------------------- |
| API_KEY     | Your New Relic Insights Insert Key                   | Yes (either License Key or API Key is required) |
| FILE_PATH   | A path or glob to the file or files you wish to tail | Yes                                             |
| LICENSE_KEY | Your New Relic License key                           | Yes (either License Key or API Key is required) |

### Docker Example

Within the root of the project run the following. You can supplement the image name and tag as you see fit.

```
docker build -t <YOUR-IMAGE-NAME>:<YOUR-TAG> .
docker run -e "FILE_PATH=/var/log/*" -e "API_KEY=<YOUR-API-KEY>" <YOUR-IMAGE-NAME>:<YOUR-TAG>
```

### Retry logic

For recoverables error, the plugin is set to send a Retry order to Fluent Bit to flush data again. By default the `Retry_Limit` is set to 1 attempt. But can be [overwritten manually](https://docs.fluentbit.io/manual/administration/scheduling-and-retries).

| Key         | Value | Description                                                                                                          |
| ----------- | ----- | -------------------------------------------------------------------------------------------------------------------- |
| Retry_Limit | N     | Integer value to set the maximum number of retries allowed. N must be >= 1 (default: 1)                              |
| Retry_Limit | False | When Retry_Limit is set to False, means that there is not limit for the number of retries that the Scheduler can do. |

## Testing docker images

For build and test docker image just run `bash test.sh`. Bear in mind that docker-compose is required to run the tests.

Testing steps are:

1. Create test/testdata folder for store temporal configurations and the log file
2. Build the image (default dockerfile is ./Dockerfile but you can set the one you want with DOCKERFILE env)
3. Run the docker-compose (./test/docker-compose.yml) with the following instances:
   - A mockserver with expectations from ./test/expectations.json
   - The built docker image with fluent bit configuration from ./test/fluent-bit.conf
4. Send some logs
5. Verify that logs are reaching the mockserver
   - Mockserver requests are verified using ./test/verification.json
6. Cleanup

## Community

New Relic hosts and moderates an online forum where customers can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub: [Log forwarding](https://discuss.newrelic.com/tag/log-forwarding)

## A note about vulnerabilities

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, review [these guidelines](https://opensource.newrelic.com/code-of-conduct/).

## License

newrelic-fluent-bit-output is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
