package liteconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Ephemeral                       bool
	DatabaseFilePath                string
	FrontendPort                    int
	DynamicPorts                    bool
	Namespaces                      []string
	DefaultNamespaceRetentionPeriod time.Duration
	Logger                          log.Logger
	portProvider                    *portProvider
}

func NewDefaultConfig() (*Config, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine user config directory: %w", err)
	}

	return &Config{
		Ephemeral:                       false,
		DatabaseFilePath:                filepath.Join(userConfigDir, "temporalite/db/default.db"),
		FrontendPort:                    7233,
		DynamicPorts:                    false,
		Namespaces:                      nil,
		DefaultNamespaceRetentionPeriod: 24 * time.Hour,
		Logger: log.NewZapLogger(log.BuildZapLogger(log.Config{
			Stdout:     true,
			Level:      "debug",
			OutputFile: "",
		})),
		portProvider: &portProvider{},
	}, nil
}

func Convert(cfg *Config) *config.Config {
	defer func() {
		if err := cfg.portProvider.close(); err != nil {
			panic(err)
		}
		// time.Sleep(5 * time.Second)
	}()

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

	if len(cfg.Namespaces) > 0 {
		sqliteConfig.ConnectAttributes["preCreateNamespaces"] = strings.Join(cfg.Namespaces, ",")
	}

	var (
		metricsPort = cfg.FrontendPort + 200
		pprofPort   = cfg.FrontendPort + 201
	)
	if cfg.DynamicPorts {
		if cfg.FrontendPort == 0 {
			cfg.FrontendPort = cfg.portProvider.mustGetFreePort()
		}
		metricsPort = cfg.portProvider.mustGetFreePort()
		pprofPort = cfg.portProvider.mustGetFreePort()
	}

	return &config.Config{
		Global: config.Global{
			Membership: config.Membership{
				MaxJoinDuration:  30 * time.Second,
				BroadcastAddress: broadcastAddress,
			},
			Metrics: &metrics.Config{
				Prometheus: &metrics.PrometheusConfig{
					ListenAddress: fmt.Sprintf("%s:%d", broadcastAddress, metricsPort),
					HandlerPath:   "/metrics",
				},
			},
			PProf: config.PProf{Port: pprofPort},
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
			"frontend": cfg.mustGetService(0),
			"history":  cfg.mustGetService(1),
			"matching": cfg.mustGetService(2),
			"worker":   cfg.mustGetService(3),
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

func (o *Config) mustGetService(frontendPortOffset int) config.Service {
	var (
		grpcPort       = o.FrontendPort + frontendPortOffset
		membershipPort = o.FrontendPort + 100 + frontendPortOffset
	)
	if o.DynamicPorts {
		if frontendPortOffset != 0 {
			grpcPort = o.portProvider.mustGetFreePort()
		}
		membershipPort = o.portProvider.mustGetFreePort()
	}
	return config.Service{
		RPC: config.RPC{
			GRPCPort:        grpcPort,
			MembershipPort:  membershipPort,
			BindOnLocalHost: true,
			BindOnIP:        "",
		},
	}
}
