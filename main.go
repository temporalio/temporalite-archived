package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/DataDog/temporalite/internal/common/persistence/sql/sqlplugin/sqlite" // needed to load sqlite plugin
	"github.com/DataDog/temporalite/internal/liteconfig"
	"github.com/DataDog/temporalite/server"
	"github.com/urfave/cli/v2"
	"go.temporal.io/server/common/headers"
	tlog "go.temporal.io/server/common/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultCfg *liteconfig.Config
)

const (
	ephemeralFlag = "ephemeral"
	dbPathFlag    = "filename"
	portFlag      = "port"
	logFormatFlag = "log-format"
)

func init() {
	defaultCfg, _ = liteconfig.NewDefaultConfig()
}

func main() {
	if err := buildCLI().Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func buildCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "temporal"
	app.Usage = "Temporal server"
	app.Version = headers.ServerVersion

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
				&cli.IntFlag{
					Name:    portFlag,
					Aliases: []string{"p"},
					Usage:   "port for the temporal-frontend GRPC service",
					Value:   defaultCfg.FrontendPort,
				},
				&cli.StringFlag{
					Name:    logFormatFlag,
					Usage:   `customize the log formatting (allowed: "json", "pretty")`,
					EnvVars: nil,
					Value:   "json",
				},
			},
			Before: func(c *cli.Context) error {
				if c.Args().Len() > 0 {
					return cli.Exit("ERROR: start command doesn't support arguments.", 1)
				}
				if c.IsSet(ephemeralFlag) && c.IsSet(dbPathFlag) {
					return cli.Exit(fmt.Sprintf("ERROR: only one of %q or %q flags may be passed at a time", ephemeralFlag, dbPathFlag), 1)
				}
				switch c.String(logFormatFlag) {
				case "json", "pretty":
				default:
					return cli.Exit(fmt.Sprintf("bad value %q passed for flag %q", c.String(logFormatFlag), logFormatFlag), 1)
				}
				return nil
			},
			Action: func(c *cli.Context) error {
				opts := []server.Option{
					server.WithFrontendPort(c.Int(portFlag)),
					server.WithDatabaseFilePath(c.String(dbPathFlag)),
				}
				if c.Bool(ephemeralFlag) {
					opts = append(opts, server.WithPersistenceDisabled())
				}
				if c.String(logFormatFlag) == "pretty" {
					lcfg := zap.NewDevelopmentConfig()
					lcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
					l, _ := lcfg.Build(
						zap.WithCaller(false),
						zap.AddStacktrace(zapcore.ErrorLevel),
					)
					logger := tlog.NewZapLogger(l)
					opts = append(opts, server.WithLogger(logger))
				}

				s, err := server.New(opts...)
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
