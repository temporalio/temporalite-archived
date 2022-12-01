// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package main

import (
	"encoding/json"
	"fmt"
	goLog "log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/headers"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/temporal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Load sqlite storage driver
	_ "go.temporal.io/server/common/persistence/sql/sqlplugin/sqlite"

	"github.com/temporalio/temporalite"
	"github.com/temporalio/temporalite/internal/liteconfig"
)

// Name of the ui-server module, used in tests to verify that it is included/excluded
// as a dependency when building with the `headless` tag enabled.
const uiServerModule = "github.com/temporalio/ui-server/v2"

const (
	ephemeralFlag          = "ephemeral"
	dbPathFlag             = "filename"
	portFlag               = "port"
	metricsPortFlag        = "metrics-port"
	uiPortFlag             = "ui-port"
	headlessFlag           = "headless"
	ipFlag                 = "ip"
	uiIPFlag               = "ui-ip"
	uiCodecEndpointFlag    = "ui-codec-endpoint"
	logFormatFlag          = "log-format"
	logLevelFlag           = "log-level"
	namespaceFlag          = "namespace"
	pragmaFlag             = "sqlite-pragma"
	configFlag             = "config"
	dynamicConfigValueFlag = "dynamic-config-value"
)

type uiConfig struct {
	Host                string
	Port                int
	TemporalGRPCAddress string
	EnableUI            bool
	CodecEndpoint       string
}

func main() {
	if err := buildCLI().Run(os.Args); err != nil {
		goLog.Fatal(err)
	}
}

// These variables are set by GoReleaser using ldflags
var version string

func buildCLI() *cli.App {
	defaultCfg, _ := liteconfig.NewDefaultConfig()

	if version == "" {
		version = "(devel)"
	}
	app := cli.NewApp()
	app.Name = "temporalite"
	app.Usage = "An experimental distribution of Temporal that runs as a single process\n\nFind more information at: https://github.com/temporalio/temporalite"
	app.Version = fmt.Sprintf("%s (server %s)", version, headers.ServerVersion)
	app.Commands = []*cli.Command{
		{
			Name:      "start",
			Usage:     "Start Temporal server",
			ArgsUsage: " ",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  ephemeralFlag,
					Value: defaultCfg.Ephemeral,
					Usage: "enable the in-memory storage driver **data will be lost on restart**",
				},
				&cli.StringFlag{
					Name:    dbPathFlag,
					Aliases: []string{"f"},
					Value:   defaultCfg.DatabaseFilePath,
					Usage:   "file in which to persist Temporal state",
				},
				&cli.StringSliceFlag{
					Name:    namespaceFlag,
					Aliases: []string{"n"},
					Usage:   `specify namespaces that should be pre-created`,
					EnvVars: nil,
					Value:   nil,
				},
				&cli.IntFlag{
					Name:    portFlag,
					Aliases: []string{"p"},
					Usage:   "port for the temporal-frontend GRPC service",
					Value:   liteconfig.DefaultFrontendPort,
				},
				&cli.IntFlag{
					Name:  metricsPortFlag,
					Usage: "port for the metrics listener",
					Value: liteconfig.DefaultMetricsPort,
				},
				&cli.IntFlag{
					Name:        uiPortFlag,
					Usage:       "port for the temporal web UI",
					DefaultText: fmt.Sprintf("--port + 1000, eg. %d", liteconfig.DefaultFrontendPort+1000),
				},
				&cli.BoolFlag{
					Name:  headlessFlag,
					Usage: "disable the temporal web UI",
				},
				&cli.StringFlag{
					Name:    ipFlag,
					Usage:   `IPv4 address to bind the frontend service to instead of localhost`,
					EnvVars: nil,
					Value:   "127.0.0.1",
				},
				&cli.StringFlag{
					Name:        uiIPFlag,
					Usage:       `IPv4 address to bind the web UI to instead of localhost`,
					DefaultText: "same as --ip (eg. 127.0.0.1)",
				},
				&cli.StringFlag{
					Name:    uiCodecEndpointFlag,
					Usage:   `UI Remote data converter HTTP endpoint`,
					EnvVars: nil,
				},
				&cli.StringFlag{
					Name:    logFormatFlag,
					Usage:   `customize the log formatting (allowed: ["json" "pretty"])`,
					EnvVars: nil,
					Value:   "json",
				},
				&cli.StringFlag{
					Name:    logLevelFlag,
					Usage:   `customize the log level (allowed: ["debug" "info" "warn" "error" "fatal"])`,
					EnvVars: nil,
					Value:   "info",
				},
				&cli.StringSliceFlag{
					Name:    pragmaFlag,
					Aliases: []string{"sp"},
					Usage:   fmt.Sprintf("specify sqlite pragma statements in pragma=value format (allowed: %q)", liteconfig.GetAllowedPragmas()),
					EnvVars: nil,
					Value:   nil,
				},
				&cli.StringFlag{
					Name:    configFlag,
					Aliases: []string{"c"},
					Usage:   `config dir path`,
					EnvVars: []string{config.EnvKeyConfigDir},
					Value:   "",
				},
				&cli.StringSliceFlag{
					Name:  dynamicConfigValueFlag,
					Usage: `dynamic config value, as KEY=JSON_VALUE (meaning strings need quotes)`,
				},
			},
			Before: func(c *cli.Context) error {
				if c.Args().Len() > 0 {
					return cli.Exit("ERROR: start command doesn't support arguments.", 1)
				}
				if c.IsSet(ephemeralFlag) && c.IsSet(dbPathFlag) {
					return cli.Exit(fmt.Sprintf("ERROR: only one of %q or %q flags may be passed at a time", ephemeralFlag, dbPathFlag), 1)
				}

				// Make sure the default db path exists (user does not specify path explicitly)
				if !c.IsSet(dbPathFlag) {
					if err := os.MkdirAll(filepath.Dir(c.String(dbPathFlag)), os.ModePerm); err != nil {
						return cli.Exit(err.Error(), 1)
					}
				}

				switch c.String(logFormatFlag) {
				case "json", "pretty", "noop":
				default:
					return cli.Exit(fmt.Sprintf("bad value %q passed for flag %q", c.String(logFormatFlag), logFormatFlag), 1)
				}

				switch c.String(logLevelFlag) {
				case "debug", "info", "warn", "error", "fatal":
				default:
					return cli.Exit(fmt.Sprintf("bad value %q passed for flag %q", c.String(logLevelFlag), logLevelFlag), 1)
				}

				// Check that ip address is valid
				if c.IsSet(ipFlag) && net.ParseIP(c.String(ipFlag)) == nil {
					return cli.Exit(fmt.Sprintf("bad value %q passed for flag %q", c.String(ipFlag), ipFlag), 1)
				}

				if c.IsSet(configFlag) {
					cfgPath := c.String(configFlag)
					if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
						return cli.Exit(fmt.Sprintf("bad value %q passed for flag %q: file not found", c.String(configFlag), configFlag), 1)
					}
				}

				return nil
			},
			Action: func(c *cli.Context) error {
				var (
					ip              = c.String(ipFlag)
					serverPort      = c.Int(portFlag)
					metricsPort     = c.Int(metricsPortFlag)
					uiPort          = serverPort + 1000
					uiIP            = ip
					uiCodecEndpoint = ""
				)

				if c.IsSet(uiPortFlag) {
					uiPort = c.Int(uiPortFlag)
				}

				if c.IsSet(uiIPFlag) {
					uiIP = c.String(uiIPFlag)
				}

				if c.IsSet(uiCodecEndpointFlag) {
					uiCodecEndpoint = c.String(uiCodecEndpointFlag)
				}

				pragmas, err := getPragmaMap(c.StringSlice(pragmaFlag))
				if err != nil {
					return err
				}

				baseConfig := &config.Config{}
				if c.IsSet(configFlag) {
					// Temporal server requires a couple of persistence config values to
					// be explicitly set or the config loading fails. While these are the
					// same values used internally, they are overridden later anyways,
					// they are just here to pass validation.
					baseConfig.Persistence.DefaultStore = liteconfig.PersistenceStoreName
					baseConfig.Persistence.NumHistoryShards = 1
					if err := config.Load("temporalite", c.String(configFlag), "", &baseConfig); err != nil {
						return err
					}
				}

				interruptChan := make(chan interface{}, 1)
				go func() {
					if doneChan := c.Done(); doneChan != nil {
						s := <-doneChan
						interruptChan <- s
					} else {
						s := <-temporal.InterruptCh()
						interruptChan <- s
					}
				}()

				opts := []temporalite.ServerOption{
					temporalite.WithDynamicPorts(),
					temporalite.WithFrontendPort(serverPort),
					temporalite.WithMetricsPort(metricsPort),
					temporalite.WithFrontendIP(ip),
					temporalite.WithDatabaseFilePath(c.String(dbPathFlag)),
					temporalite.WithNamespaces(c.StringSlice(namespaceFlag)...),
					temporalite.WithSQLitePragmas(pragmas),
					temporalite.WithUpstreamOptions(
						temporal.InterruptOn(interruptChan),
					),
					temporalite.WithBaseConfig(baseConfig),
				}
				if !c.Bool(headlessFlag) {
					frontendAddr := fmt.Sprintf("%s:%d", ip, serverPort)
					cfg := &uiConfig{
						Host:                uiIP,
						Port:                uiPort,
						TemporalGRPCAddress: frontendAddr,
						EnableUI:            true,
						CodecEndpoint:       uiCodecEndpoint,
					}

					opt, err := newUIOption(cfg, c.String(configFlag))
					if err != nil {
						return err
					}
					if opt != nil {
						opts = append(opts, opt)
					}
				}
				if c.Bool(ephemeralFlag) {
					opts = append(opts, temporalite.WithPersistenceDisabled())
				}

				var logger log.Logger
				switch c.String(logFormatFlag) {
				case "pretty":
					lcfg := zap.NewDevelopmentConfig()
					switch c.String(logLevelFlag) {
					case "debug":
						lcfg.Level.SetLevel(zap.DebugLevel)
					case "info":
						lcfg.Level.SetLevel(zap.InfoLevel)
					case "warn":
						lcfg.Level.SetLevel(zap.WarnLevel)
					case "error":
						lcfg.Level.SetLevel(zap.ErrorLevel)
					case "fatal":
						lcfg.Level.SetLevel(zap.FatalLevel)
					}
					lcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
					l, err := lcfg.Build(
						zap.WithCaller(false),
						zap.AddStacktrace(zapcore.ErrorLevel),
					)
					if err != nil {
						return err
					}
					logger = log.NewZapLogger(l)
				case "noop":
					logger = log.NewNoopLogger()
				default:
					logger = log.NewZapLogger(log.BuildZapLogger(log.Config{
						Stdout:     true,
						Level:      c.String(logLevelFlag),
						OutputFile: "",
					}))
				}
				opts = append(opts, temporalite.WithLogger(logger))

				configVals, err := getDynamicConfigValues(c.StringSlice(dynamicConfigValueFlag))
				if err != nil {
					return err
				}
				for k, v := range configVals {
					opts = append(opts, temporalite.WithDynamicConfigValue(k, v))
				}

				s, err := temporalite.NewServer(opts...)
				if err != nil {
					return err
				}

				if err := s.Start(); err != nil {
					return cli.Exit(fmt.Sprintf("Unable to start server. Error: %v", err), 1)
				}
				return cli.Exit("All services are stopped.", 0)
			},
		},
	}

	return app
}

func getPragmaMap(input []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, pragma := range input {
		vals := strings.Split(pragma, "=")
		if len(vals) != 2 {
			return nil, fmt.Errorf("ERROR: pragma statements must be in KEY=VALUE format, got %q", pragma)
		}
		result[vals[0]] = vals[1]
	}
	return result, nil
}

func getDynamicConfigValues(input []string) (map[dynamicconfig.Key][]dynamicconfig.ConstrainedValue, error) {
	ret := make(map[dynamicconfig.Key][]dynamicconfig.ConstrainedValue, len(input))
	for _, keyValStr := range input {
		keyVal := strings.SplitN(keyValStr, "=", 2)
		if len(keyVal) != 2 {
			return nil, fmt.Errorf("dynamic config value not in KEY=JSON_VAL format")
		}
		key := dynamicconfig.Key(keyVal[0])
		// We don't support constraints currently
		var val dynamicconfig.ConstrainedValue
		if err := json.Unmarshal([]byte(keyVal[1]), &val.Value); err != nil {
			return nil, fmt.Errorf("invalid JSON value for key %q: %w", key, err)
		}
		ret[key] = append(ret[key], val)
	}
	return ret, nil
}
