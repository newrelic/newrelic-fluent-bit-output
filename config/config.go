package config

import (
	"fmt"
	"github.com/fluent/fluent-bit-go/output"
	"net/http"
	"net/url"
	"strconv"
	"unsafe"
)

type PluginConfig struct {
	BufferManagerConfig	  BufferConfig
	NRClientConfig		  NRClientConfig
	ProxyResolver         func(*http.Request) (*url.URL, error)
}

type BufferConfig struct {
	MaxBufferSize         int64
	MaxRecords            int64
	MaxTimeBetweenFlushes int64
}

type NRClientConfig struct {
	Endpoint              string
	ApiKey                string
	LicenseKey            string
	UseApiKey             bool
}


func NewPluginConfig(ctx unsafe.Pointer) (cfg PluginConfig, err error) {
	cfg.BufferManagerConfig, err = parseBufferConfig(ctx)
	if err != nil {
		return
	}

	cfg.NRClientConfig, err = parseNRClientConfig(ctx)
	if err != nil {
		return
	}

	ignoreSystemProxy, err := optBool(ctx, "ignoreSystemProxy", false)
	if err != nil {
		return
	}
	proxy := output.FLBPluginConfigKey(ctx, "nrclient")

	proxyResolver, err := getProxyResolver(ignoreSystemProxy, proxy)
	if err != nil {
		err = fmt.Errorf("invalid nrclient configuration: %v", err)
		return
	}
	cfg.ProxyResolver = proxyResolver

	return cfg, nil
}

func parseBufferConfig(ctx unsafe.Pointer) (cfg BufferConfig, err error) {
	maxBufferSize, err := optInt64(ctx, "maxBufferSize", 256000)
	if err != nil {
		return
	}
	cfg.MaxBufferSize = maxBufferSize

	maxRecords, err := optInt64(ctx, "maxRecords", 1024)
	if err != nil {
		return
	}
	cfg.MaxRecords = maxRecords

	maxTimeBetweenFlushes, err := optInt64(ctx, "maxTimeBetweenFlushes", 5000)
	if err != nil {
		return
	}
	cfg.MaxTimeBetweenFlushes = maxTimeBetweenFlushes

	return cfg, err
}

func parseNRClientConfig(ctx unsafe.Pointer) (cfg NRClientConfig, err error) {
	cfg.Endpoint = output.FLBPluginConfigKey(ctx, "endpoint")
	if len(cfg.Endpoint) == 0 {
		cfg.Endpoint = "https://log-api.newrelic.com/log/v1"
	}
	cfg.LicenseKey = output.FLBPluginConfigKey(ctx, "licenseKey")
	cfg.ApiKey = output.FLBPluginConfigKey(ctx, "apiKey")

	if len(cfg.ApiKey) == 0 && len(cfg.LicenseKey) == 0 {
		err = fmt.Errorf("either apiKey or licenseKey must be specified")
		return
	}

	if len(cfg.ApiKey) > 0 && len(cfg.LicenseKey) > 0 {
		err = fmt.Errorf("only one of apiKey or licenseKey can be specified")
		return
	}

	cfg.UseApiKey = len(cfg.ApiKey) > 0

	return
}

func optInt64(ctx unsafe.Pointer, keyName string, defaultValue int64) (int64, error) {
	rawVal := output.FLBPluginConfigKey(ctx, keyName)
	if len(rawVal) == 0 {
		return defaultValue, nil
	} else {
		value, err := strconv.ParseInt(rawVal, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid value for %s: %s. Must be an integer.", keyName, rawVal)
		}
		return value, nil
	}
}

func optBool(ctx unsafe.Pointer, keyName string, defaultValue bool) (bool, error) {
	rawVal := output.FLBPluginConfigKey(ctx, keyName)
	if len(rawVal) == 0 {
		return defaultValue, nil
	} else {
		value, err := strconv.ParseBool(rawVal)
		if err != nil {
			return false, fmt.Errorf("invalid value for %s: %s. Valid values: true, false.", keyName, rawVal)
		}
		return value, nil
	}
}

func getProxyResolver(ignoreSystemProxy bool, proxy string) (func(*http.Request) (*url.URL, error), error) {
	if len(proxy) > 0 {
		// User-defined nrclient
		prUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}

		return http.ProxyURL(prUrl), nil
	} else if !ignoreSystemProxy {
		// Proxy defined via the HTTPS_PROXY (takes precedence) or HTTP_PROXY environment variables
		return http.ProxyFromEnvironment, nil
	} else {
		// No nrclient
		return http.ProxyURL(nil), nil
	}
}