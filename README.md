[![Community Plus header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Plus.png)](https://opensource.newrelic.com/oss-category/#community-plus)

# Fluent Bit output plugin for New Relic

The **newrelic-fluent-bit-output** plugin forwards output to New Relic. It works on all versions of Fluent Bit greater than 0.12 but for the best experience we recommend using versions greater than 1.0. This project is provided AS-IS WITHOUT WARRANTY OR SUPPORT, although you can report issues and contribute to the project here on GitHub.

You can download the output plugin pre-compiled under our [releases](https://github.com/newrelic/newrelic-fluent-bit-output/releases/latest). Alternatively you can compile the plugin yourself and store `out_newrelic-linux-*.so` or `out_newrelic-windows-*.dll` at a location that can be accessed by the fluent-bit daemon. See to [this section](DEVELOPER.md#compiling-the-out_newrelic-plugin) in [DEVELOPER.md](DEVELOPER.md) for more details. The plugin, together with Fluent Bit, is also delivered as a standalone [Docker image](#docker-image).

Note that for certain Linux Enterprise users,
[including CentOS 7, Debian 8 and 9, Ubuntu, and Raspbian 8](https://fluentbit.io/documentation/0.13/installation/td-agent-bit.html),
the name of Fluent Bit is td-agent-bit, instead of fluent-bit. So, anywhere in this doc where it says `fluent-bit`,
just replace it with `td-agent-bit` (for example, you will need to edit `td-agent-bit.conf` instead of `fluent-bit.conf`).

In order to insert records into New Relic, you can configure the plugin with a config file or configure it via command line flags. You can find more details on how to configure Fluent Bit [here](https://docs.fluentbit.io/manual/configuration).

## Getting started with the Fluent Bit output plugin for New Relic

Fluent Bit needs to know the location of the New Relic output plugin, and the license/api key for outputting to New Relic. It is **vitally important** to pay attention to white space in your config files. Please use four spaces to indent,
and one space between keys and values.

1. Find or create a `plugins.conf` file in your Fluent Bit directory and add a reference to `out_newrelic-linux-*.so` or `out_newrelic-windows-*.dll`,
   adjacent to your `fluent-bit.conf` file:
    ```
    [PLUGINS]
        Path /path/to/newrelic-fluent-bit-output/out_newrelic-linux-*.so
    ```
2. Modify fluent-bit.conf` and add the following line under the `[SERVICE]` block:
    ```
    [SERVICE]
        # This is the main configuration block for fluent bit.
        # Ensure the follow line exists somewhere in the SERVICE block
        Plugins_File plugins.conf
    ```
3. And at the end of `fluent-bit.conf`, add the following to set up the input and output plugins and example filter to add additional attributes to your logs:
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
4. Restart Fluent Bit: `fluent-bit -c /path/to/fluent-bit.conf`
5. Append a test log message to your log file: `echo "test message" >> /path/to/your/log/file`
6. Search New Relic Logs for `"test message"`

## Configuration Parameters

The plugin supports the following configuration parameters (apart from the ones provided out-of-the-box by Fluent Bit for the output plugins, such as the [retry options](#retry-logic)). Note that it's **mandatory to supply either`apiKey` or `licenseKey`**.

| Key                | Description                                                                                                                                                                                                                                                                                                                                                                                                              | Default                               |
|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| endpoint           | The endpoint you send data to. By default, it sends it to the US (`endpoint=https://log-api.newrelic.com/log/v1`). Set it to `https://log-api.eu.newrelic.com/log/v1` to send it to the EU region.                                                                                                                                                                                                                       | `https://log-api.newrelic.com/log/v1` |
| apiKey             | Your New Relic Insights Insert key. For information on how to find your New Relic Insights Insert key, take a look at the documentation [here](https://docs.newrelic.com/docs/insights/insights-data-sources/custom-data/send-custom-events-event-api#register).                                                                                                                                                         | (none)                                |
| licenseKey         | Your New Relic License key                                                                                                                                                                                                                                                                                                                                                                                               | (none)                                |
| httpClientTimeout  | Http Client timeout for sending the logs (in seconds)                                                                                                                                                                                                                                                                                                                                                                    | 5                                     |
| maxBufferSize      | **[Deprecated since 1.3.0]** The maximum size the payloads sent in bytes                                                                                                                                                                                                                                                                                                                                                 | 256000                                |
| maxRecords         | **[Deprecated since 1.3.0]** The maximum number of records to send at a time                                                                                                                                                                                                                                                                                                                                             | 1024                                  |
| proxy              | Optional proxy to communicate with New Relic, overrides any environment-defined one. Must follow the format `https://user:password@hostname:port`. Can be HTTP or HTTPS.                                                                                                                                                                                                                                                 | (none)                                |
| ignoreSystemProxy  | Ignore any proxy defined via the `HTTP_PROXY` or `HTTPS_PROXY` environment variables. Note that if a proxy has been defined using the `proxy` parameter, this one has no effect.                                                                                                                                                                                                                                         | false                                 |
| caBundleFile       | **[LINUX HTTPS ONLY]** Specifies the Certificate Authority certificate to use for validating HTTPS connections against the proxy. Useful when the proxy uses a self-signed certificate. **The certificate file must be in the PEM format**. If not specified, then the operating system's CA list is used. Only used when `validateProxyCerts` is `true`.                                                                | (none)                                |
| caBundleDir        | **[LINUX HTTPS ONLY]** Specifies a folder containing one or more Certificate Authority certificates ot use for validating HTTPS connections against the proxy. Useful when the proxy uses a self-signed certificate. **Only certificate files in the PEM format and \*.pem extension will be considered**. If not specified, then the operating system's CA list is used. Only used when `validateProxyCerts` is `true`. | (none)                                |
| validateProxyCerts | **[HTTPS ONLY]** When using a HTTPS proxy, the proxy certificates are validated by default when establishing a HTTPS connection. To disable the proxy certificate validation, set `validateProxyCerts` to `false` (insecure)                                                                                                                                                                                             | true                                  |
| sendMetrics        | Set to true to send plugin troubleshoot metrics to the Metrics event type. Please see [this section](#troubleshooting-metrics) for more details                                                                                                                                                                                                                                                                          | false                                 |

#### Proxy support

The plugin automatically detects the `HTTP_PROXY` and `HTTPS_PROXY` environment variables, and automatically uses them to set up the proxy configuration.

If you want to bypass the system-wide defined proxy for some reason, you can use the `ignoreSystemProxy` configuration parameter.

You can also specify a custom proxy to send the logs to (different from the system-wide defined) by using the `proxy` configuration parameter.

HTTPS proxies (having an `https://...` URL) use a certificate to encrypt the connection between the plugin and the proxy. If you are using a self-signed certificate (not trusted by the Certification Authorities defined at your system level), you can:

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
#### Certificates
Certificates can be trick because depends on the proxy (`HTTP` vs `HTTPS`), Linux, Windows, Logging Endpoint,
Infra-Agent Endpoint, etc. We will talk about these configurations below.

##### Linux
For `HTTP` proxies we don't need to setup any certificate. Out of the box the plugin will load the system certificates
and we will be able to send logs into the logging endpoint.

For `HTTPS` proxies you can specify the proxy self-signed certificate (PEM file) using either the `caBundleFile` or `caBundleDir`
parameters.

##### Windows
Similar to Linux, for `HTTP` proxies we don't need to setup any certificate. Out of the box the plugin will load
the system certificates.

For `HTTPS`, **special attention is required**. There are two approaches to configure it: by importing them to the system certificate pool (recommended), or by using the `caBundleFile`/`caBundleDir` options.

###### Approach 1: import the proxy certificate to the system pool (**recommended**)
The recommended process to import the proxy self-signed certificate (PEM file) using the MMC tool. You can refer to
[this link](https://www.ssls.com/knowledgebase/how-to-import-intermediate-and-root-certificates-via-mmc/), but in Step 2
ensure to import it in your `Trusted Root Certification Authorities` instead of importing it in the
`Intermediate Certification Authorities`.

###### Approach 2: using `caBundleFile`/`caBundleDir`
On Windows (differently from Linux) we cannot load both the certificates from system certificate pool and the one(s) specified via `caBundleFile`/`caBundleDir`. So, if you are using `caBundleFile` or `caBundleDir`, you must **ensure that the following certificates are placed in the same PEM file (when using `caBundleFile`) or in the same directory (when using `caBundleDir`)**:
- The Proxy certificate (because it's a `HTTPS` proxy)
- The Logging Endpoint certificate (eg. `https://log-api.newrelic.com/log/v1`)

The Logging Endpoint certificate can be checked using the `openssl` command:

```shell
openssl s_client -connect log-api.newrelic.com:443 -servername log-api.newrelic.com
```

#### Retry logic

Fluent Bit provides an out-of-the-box retry logic, configurable via the `Retry_Limit` option. For recoverable errors, the New Relic output plugin requests Fluent Bit to retry flushing data again later. By default, the `Retry_Limit` is set to 1 attempt, but can be [overwritten manually](https://docs.fluentbit.io/manual/administration/scheduling-and-retries).

| Key         | Value | Description                                                                                                          |
| ----------- | ----- | -------------------------------------------------------------------------------------------------------------------- |
| Retry_Limit | N     | Integer value to set the maximum number of retries allowed. N must be >= 1 (default: 1)                              |
| Retry_Limit | False | When Retry_Limit is set to False, means that there is not limit for the number of retries that the Scheduler can do. |

#### Troubleshooting metrics
Set the `sendMetrics` option to `true` if you want to send troubleshooting metrics to your Metrics event type via the [Metrics API](https://docs.newrelic.com/docs/data-apis/ingest-apis/metric-api/introduction-metric-api/). Please note that **enabling this option will incur extra ingestion costs** due to the data size of the metrics stored in your New Relic account.

Please note that the **metrics reported by this plugin must not be considered as a stable API: they can change its naming or dimensions at any time in newer plugin versions**. That is, **no critical alerts or dashboard should be created out of them**. The purpose of these metrics is no other than to allow you to troubleshoot a malfunctioning Fluent Bit installation.

The following are the metrics currently reported by the plugin:

| Metric name               | Dimensions                        | Description                                                                                               | Units         |
|---------------------------|-----------------------------------|-----------------------------------------------------------------------------------------------------------|---------------|
| logs.fb.packaging.time    | hasError (bool)                   | Time used to package a Fluent Bit chunk into one or more <=1MB compressed New Relic payloads              | milliseconds  |
| logs.fb.payload.count     | -                                 | Amount of <=1MB compressed New Relic payloads that a single Fluent Bit chunk was divided into             | integer count |
| logs.fb.total.send.time   | -                                 | Time used to send a single Fluent Bit chunk consisting of one or more <=1MB compressed New Relic payloads | milliseconds  |
| logs.fb.payload.send.time | statusCode (int), hasError (bool) | Time used to send an individual <=1MB compressed New Relic payload                                        | milliseconds  |
| logs.fb.payload.size      | statusCode (int), hasError (bool) | Compressed size of an individual <=1MB compressed New Relic payload                                       | bytes         |

For convenience, we have included a Dashboard in JSON format (`troubleshooting-dashboard.json.template`) that you can import into your New Relic account.  **To use it, search for "YOUR_ACCOUNT_ID" and replace it by your New Relic Account ID before importing it as JSON.** The dashboard displays the above metrics in a convenient way and guidance to help you quickly detect problems in your installation. As mentioned above, this dashboard should be used when troubleshooting a malfunctioning installation, but should not be relied upon in the long term as any of the metrics it uses or their related dimensions could change at any time.

## Docker Image

This plugin also comes packaged in a Docker image, available [here](https://hub.docker.com/r/newrelic/newrelic-fluentbit-output). To use it, you just need to pull the image and run it with your desired configuration:

```
docker pull newrelic/newrelic-fluentbit-output
docker run -e "FILE_PATH=/var/log/*" -e "API_KEY=<YOUR-API-KEY>" newrelic/newrelic-fluentbit-output
```

The available Docker container configuration options are:

| Key         | Description                                          | Required                                        |
| ----------- | ---------------------------------------------------- | ----------------------------------------------- |
| API_KEY     | Your New Relic Insights Insert Key                   | Yes (either License Key or API Key is required) |
| FILE_PATH   | A path or glob to the file or files you wish to tail | Yes                                             |
| LICENSE_KEY | Your New Relic License key                           | Yes (either License Key or API Key is required) |

Alternatively, you can mount your own Fluent Bit configuration file as a volume at `/fluent-bit/etc/fluent-bit.conf` to use your custom configuration.

## Community

New Relic hosts and moderates an online forum where customers can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub: [Log forwarding](https://discuss.newrelic.com/tag/log-forwarding)

## A note about vulnerabilities

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, review [these guidelines](https://opensource.newrelic.com/code-of-conduct/).

## License

newrelic-fluent-bit-output is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
