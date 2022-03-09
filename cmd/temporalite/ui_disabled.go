//go:build headless

package main

import "github.com/DataDog/temporalite"

func newUIOption(frontendAddr string, uiIP string, uiPort int) temporalite.ServerOption {
	return nil
}
