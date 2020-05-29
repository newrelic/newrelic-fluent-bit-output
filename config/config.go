package config

import (
	"fmt"
	"github.com/fluent/fluent-bit-go/output"
	"log"
	"strconv"
	"unsafe"
)

type PluginConfig struct {
	NRClientConfig NRClientConfig
	ProxyConfig    ProxyConfig
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

func (cfg NRClientConfig) GetNewRelicKey() string {
	var id string
	if cfg.UseApiKey {
		id = cfg.ApiKey
	} else {
		id = cfg.LicenseKey
	}
	return id
}

func NewPluginConfig(ctx unsafe.Pointer) (cfg PluginConfig, err error) {
	cfg.NRClientConfig, err = parseNRClientConfig(ctx)
	if err != nil {
		return
	}

	cfg.ProxyConfig, err = parseProxyConfig(ctx)
	if err != nil {
		return
	}

	checkDeprecatedConfigFields(ctx)

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

func checkDeprecatedConfigFields(ctx unsafe.Pointer) {
	checkDeprecatedConfigField(ctx, "maxBufferSize")
	checkDeprecatedConfigField(ctx, "maxRecords")
	checkDeprecatedConfigField(ctx, "maxTimeBetweenFlushes")
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

func checkDeprecatedConfigField(ctx unsafe.Pointer, keyName string) {
	if rawVal := output.FLBPluginConfigKey(ctx, keyName); len(rawVal) > 0 {
		log.Printf("Configuration field %s is deprecated and will be ignored\n", keyName)
	}
}
