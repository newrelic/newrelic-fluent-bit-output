package config

import (
	"fmt"
	"github.com/fluent/fluent-bit-go/output"
	"strconv"
	"unsafe"
)

type PluginConfig struct {
	BufferManagerConfig BufferConfig
	NRClientConfig      NRClientConfig
	ProxyConfig         ProxyConfig
}

type BufferConfig struct {
	MaxBufferSize         int64
	MaxRecords            int64
	MaxTimeBetweenFlushes int64
}

type NRClientConfig struct {
	Endpoint   string
	ApiKey     string
	LicenseKey string
	UseApiKey  bool
}

type ProxyConfig struct {
	IgnoreSystemProxy bool
	Proxy             string
	CABundleFile      string
	CABundleDir       string
	ValidateCerts     bool
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

	cfg.ProxyConfig, err = parseProxyConfig(ctx)
	if err != nil {
		return
	}

	return
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

	return
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

func parseProxyConfig(ctx unsafe.Pointer) (cfg ProxyConfig, err error) {
	cfg.IgnoreSystemProxy, err = optBool(ctx, "ignoreSystemProxy", false)
	if err != nil {
		return
	}

	cfg.Proxy = output.FLBPluginConfigKey(ctx, "proxy")

	cfg.CABundleFile = output.FLBPluginConfigKey(ctx, "caBundleFile")

	cfg.CABundleDir = output.FLBPluginConfigKey(ctx, "caBundleDir")

	cfg.ValidateCerts, err = optBool(ctx, "validateProxyCerts", true)
	if err != nil {
		return
	}

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
