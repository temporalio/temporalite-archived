//go:build !headless

package main

import (
	// This file should be the only one to import ui-server packages.
	// This is to avoid embedding the UI's static assets in the binary when the `headless` build tag is enabled.
	uiserver "github.com/temporalio/ui-server/server"
	uiconfig "github.com/temporalio/ui-server/server/config"
	uiserveroptions "github.com/temporalio/ui-server/server/server_options"

	"github.com/DataDog/temporalite"
)

func newUIOption(frontendAddr string, uiIP string, uiPort int) temporalite.ServerOption {
	return temporalite.WithUI(uiserver.NewServer(uiserveroptions.WithConfigProvider(newUIConfig(
		frontendAddr,
		uiIP,
		uiPort,
	))))
}

func newUIConfig(frontendAddr string, uiIP string, uiPort int) *uiconfig.Config {
	return &uiconfig.Config{
		TemporalGRPCAddress: frontendAddr,
		Host:                uiIP,
		Port:                uiPort,
		EnableUI:            true,
	}
}
