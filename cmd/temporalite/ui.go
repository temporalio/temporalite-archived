// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

//go:build !headless

package main

import (
	// This file should be the only one to import ui-server packages.
	// This is to avoid embedding the UI's static assets in the binary when the `headless` build tag is enabled.
	uiserver "github.com/temporalio/ui-server/v2/server"
	uiconfig "github.com/temporalio/ui-server/v2/server/config"
	uiserveroptions "github.com/temporalio/ui-server/v2/server/server_options"

	"github.com/temporalio/temporalite"
	"go.temporal.io/server/common/config"
)

func newUIOption(frontendAddr string, uiIP string, uiPort int, configDir string) (temporalite.ServerOption, error) {
	cfg, err := newUIConfig(
		frontendAddr,
		uiIP,
		uiPort,
		configDir,
	)
	if err != nil {
		return nil, err
	}
	return temporalite.WithUI(uiserver.NewServer(uiserveroptions.WithConfigProvider(cfg))), nil
}

func newUIConfig(frontendAddr string, uiIP string, uiPort int, configDir string) (*uiconfig.Config, error) {
	cfg := &uiconfig.Config{}
	if configDir != "" {
		if err := config.Load("temporalite-ui", configDir, "", cfg); err != nil {
			return nil, err
		}
	}
	cfg.TemporalGRPCAddress = frontendAddr
	cfg.Host = uiIP
	cfg.Port = uiPort
	cfg.EnableUI = true
	return cfg, nil
}
