// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

//go:build headless

package main

import "github.com/temporalio/temporalite"

func newUIOption(c *uiConfig, configDir string) (temporalite.ServerOption, error) {
	return nil, nil
}
