// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

//go:build !headless

package main

// This file should be the only one to import ui-server packages.
// This is to avoid embedding the UI's static assets in the binary when the `headless` build tag is enabled.
import (
	"strings"

	provider "github.com/temporalio/ui-server/v2/plugins/fs_config_provider"
	uiserver "github.com/temporalio/ui-server/v2/server"
	uiconfig "github.com/temporalio/ui-server/v2/server/config"
	uiserveroptions "github.com/temporalio/ui-server/v2/server/server_options"

	"github.com/temporalio/temporalite"
)

func newUIOption(c *uiConfig, configDir string) (temporalite.ServerOption, error) {
	cfg, err := newUIConfig(
		c,
		configDir,
	)
	if err != nil {
		return nil, err
	}
	return temporalite.WithUI(uiserver.NewServer(uiserveroptions.WithConfigProvider(cfg))), nil
}

func newUIConfig(c *uiConfig, configDir string) (*uiconfig.Config, error) {
	cfg := &uiconfig.Config{
		Host:                c.Host,
		Port:                c.Port,
		TemporalGRPCAddress: c.TemporalGRPCAddress,
		EnableUI:            c.EnableUI,
		Codec: uiconfig.Codec{
			Endpoint: c.CodecEndpoint,
		},
	}

	if configDir != "" {
		if err := provider.Load(configDir, cfg, "temporalite-ui"); err != nil {
			if !strings.HasPrefix(err.Error(), "no config files found") {
				return nil, err
			}
		}
	}

	// See https://github.com/temporalio/temporalite/pull/174/files#r1035958308
	// for context about those overrides.
	cfg.TemporalGRPCAddress = c.TemporalGRPCAddress
	cfg.EnableUI = c.EnableUI

	return cfg, nil
}
