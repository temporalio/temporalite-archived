package liteconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DataDog/temporalite/internal/common/persistence/sql/sqlplugin/sqlite"

	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
)

const (
	broadcastAddress     = "127.0.0.1"
	persistenceStoreName = "sqlite-default"
)

type Config struct {
	Ephemeral        bool
	DatabaseFilePath string
	FrontendPort     int
	Logger           log.Logger
}

func NewDefaultConfig() (*Config, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine user config directory: %w", err)
	}

	return &Config{
		Ephemeral:        false,
		DatabaseFilePath: filepath.Join(userConfigDir, "temporalite/db/default.db"),
		FrontendPort:     7233,
		Logger: log.NewZapLogger(log.BuildZapLogger(log.Config{
			Stdout:     true,
			Level:      "debug",
			OutputFile: "",
		})),
	}, nil
}

func Convert(cfg *Config) *config.Config {
	sqliteConfig := config.SQL{
		PluginName:        sqlite.PluginName,
		ConnectAttributes: make(map[string]string),
	}
	if cfg.Ephemeral {
		sqliteConfig.ConnectAttributes["mode"] = "memory"
		sqliteConfig.DatabaseName = "temporal.db"
	} else {
		sqliteConfig.ConnectAttributes["mode"] = "rwc"
		sqliteConfig.DatabaseName = cfg.DatabaseFilePath
	}

	return &config.Config{
		Global: config.Global{
			Membership: config.Membership{
				MaxJoinDuration:  30 * time.Second,
				BroadcastAddress: broadcastAddress,
			},
			Metrics: &metrics.Config{
				Prometheus: &metrics.PrometheusConfig{
					ListenAddress: fmt.Sprintf("%s:%d", broadcastAddress, cfg.FrontendPort+200),
					HandlerPath:   "/metrics",
				},
			},
			PProf: config.PProf{Port: cfg.FrontendPort + 201},
		},
		Persistence: config.Persistence{
			DefaultStore:     persistenceStoreName,
			VisibilityStore:  persistenceStoreName,
			NumHistoryShards: 1,
			DataStores: map[string]config.DataStore{
				persistenceStoreName: {SQL: &sqliteConfig},
			},
		},
		ClusterMetadata: &config.ClusterMetadata{
			EnableGlobalNamespace:    false,
			FailoverVersionIncrement: 10,
			MasterClusterName:        "active",
			CurrentClusterName:       "active",
			ClusterInformation: map[string]config.ClusterInformation{
				"active": {
					Enabled:                true,
					InitialFailoverVersion: 1,
					RPCAddress:             fmt.Sprintf("%s:%d", broadcastAddress, cfg.FrontendPort),
				},
			},
		},
		DCRedirectionPolicy: config.DCRedirectionPolicy{
			Policy: "noop",
		},
		Services: map[string]config.Service{
			"frontend": cfg.getService(0),
			"history":  cfg.getService(1),
			"matching": cfg.getService(2),
			"worker":   cfg.getService(3),
		},
		Archival: config.Archival{
			History: config.HistoryArchival{
				State:      "disabled",
				EnableRead: false,
				Provider:   nil,
			},
			Visibility: config.VisibilityArchival{
				State:      "disabled",
				EnableRead: false,
				Provider:   nil,
			},
		},
		PublicClient: config.PublicClient{
			HostPort: fmt.Sprintf("%s:%d", broadcastAddress, cfg.FrontendPort),
		},
		NamespaceDefaults: config.NamespaceDefaults{
			Archival: config.ArchivalNamespaceDefaults{
				History: config.HistoryArchivalNamespaceDefaults{
					State: "disabled",
				},
				Visibility: config.VisibilityArchivalNamespaceDefaults{
					State: "disabled",
				},
			},
		},
	}
}

func (o *Config) getService(frontendPortOffset int) config.Service {
	return config.Service{
		RPC: config.RPC{
			GRPCPort:        o.FrontendPort + frontendPortOffset,
			MembershipPort:  o.FrontendPort + 100 + frontendPortOffset,
			BindOnLocalHost: true,
			BindOnIP:        "",
		},
	}
}
