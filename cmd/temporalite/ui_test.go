//go:build !headless

package main

import (
	"runtime/debug"
	"testing"
)

// This test ensures that ui-server is a dependency of Temporalite built in non-headless mode.
func TestHasUIServerDependency(t *testing.T) {
	info, _ := debug.ReadBuildInfo()
	for _, dep := range info.Deps {
		if dep.Path == uiServerModule {
			return
		}
	}
	t.Errorf("%s should be a dependency when headless tag is not enabled", uiServerModule)
	// If the ui-server module name is ever changed, this test should fail and indicate that the
	// module name should be updated for this and the equivalent test case in ui_disabled_test.go
	// to continue working.
	t.Logf("Temporalite's %s dependency is missing. Was this module renamed recently?", uiServerModule)
}

func TestNewUIConfig(t *testing.T) {
	cfg := newUIConfig("localhost:7233", "localhost", 8233)
	if err := cfg.Validate(); err != nil {
		t.Errorf("config not valid: %s", err)
	}
}
