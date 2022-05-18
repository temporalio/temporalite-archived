// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporalite

import (
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/temporal"

	"github.com/DataDog/temporalite/internal/liteconfig"
)

// WithLogger overrides the default logger.
func WithLogger(logger log.Logger) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Logger = logger
	})
}

// WithDatabaseFilePath persists state to the file at the specified path.
func WithDatabaseFilePath(filepath string) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Ephemeral = false
		cfg.DatabaseFilePath = filepath
	})
}

// WithPersistenceDisabled disables file persistence and uses the in-memory storage driver.
// State will be reset on each process restart.
func WithPersistenceDisabled() ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Ephemeral = true
	})
}

// WithUI enables the Temporal web interface.
//
// When unspecified, Temporal will run in headless mode.
//
// This option accepts a UIServer implementation in order to avoid bloating
// programs that do not need to embed the UI.
// See ./cmd/temporalite/main.go for an example of usage.
func WithUI(server liteconfig.UIServer) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.UIServer = server
	})
}

// WithFrontendPort sets the listening port for the temporal-frontend GRPC service.
//
// When unspecified, the default port number of 7233 is used.
func WithFrontendPort(port int) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.FrontendPort = port
	})
}

// WithFrontendIP binds the temporal-frontend GRPC service to a specific IP (eg. `0.0.0.0`)
// Check net.ParseIP for supported syntax; only IPv4 is supported.
//
// When unspecified, the frontend service will bind to localhost.
func WithFrontendIP(address string) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.FrontendIP = address
	})
}

// WithDynamicPorts starts Temporal on system-chosen ports.
func WithDynamicPorts() ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.DynamicPorts = true
	})
}

// WithNamespaces registers each namespace on Temporal start.
func WithNamespaces(namespaces ...string) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Namespaces = append(cfg.Namespaces, namespaces...)
	})
}

// WithSQLitePragmas applies pragma statements to SQLite on Temporal start.
func WithSQLitePragmas(pragmas map[string]string) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		if cfg.SQLitePragmas == nil {
			cfg.SQLitePragmas = make(map[string]string)
		}
		for k, v := range pragmas {
			cfg.SQLitePragmas[k] = v
		}
	})
}

// WithUpstreamOptions registers Temporal server options.
func WithUpstreamOptions(options ...temporal.ServerOption) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.UpstreamOptions = append(cfg.UpstreamOptions, options...)
	})
}

func WithTlsOptions(caCertificate, certificate, key string, useMtls bool) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		if caCertificate != "" {
			cfg.Tls.ClientCAFiles = append(cfg.Tls.ClientCAFiles, caCertificate)
		}
		cfg.Tls.CertFile = certificate
		cfg.Tls.KeyFile = key
		cfg.Tls.RequireClientAuth = useMtls
	})
}

type applyFuncContainer struct {
	applyInternal func(*liteconfig.Config)
}

func (fso *applyFuncContainer) apply(cfg *liteconfig.Config) {
	fso.applyInternal(cfg)
}

func newApplyFuncContainer(apply func(*liteconfig.Config)) *applyFuncContainer {
	return &applyFuncContainer{
		applyInternal: apply,
	}
}
