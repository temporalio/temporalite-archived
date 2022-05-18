// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporalite

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/rpc/encryption"
	"go.temporal.io/server/schema/sqlite"
	"go.temporal.io/server/temporal"

	"github.com/DataDog/temporalite/internal/liteconfig"
)

// Server wraps temporal.Server.
type Server struct {
	internal         temporal.Server
	ui               liteconfig.UIServer
	frontendHostPort string
	config           *liteconfig.Config
}

type ServerOption interface {
	apply(*liteconfig.Config)
}

// NewServer returns a new instance of Server.
func NewServer(opts ...ServerOption) (*Server, error) {
	c, err := liteconfig.NewDefaultConfig()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt.apply(c)
	}

	for pragma := range c.SQLitePragmas {
		if _, ok := liteconfig.SupportedPragmas[strings.ToLower(pragma)]; !ok {
			return nil, fmt.Errorf("ERROR: unsupported pragma %q, %v allowed", pragma, liteconfig.GetAllowedPragmas())
		}
	}

	cfg := liteconfig.Convert(c)
	sqlConfig := cfg.Persistence.DataStores[liteconfig.PersistenceStoreName].SQL

	if !c.Ephemeral {
		// Apply migrations if file does not already exist
		if _, err := os.Stat(c.DatabaseFilePath); os.IsNotExist(err) {
			if err := sqlite.SetupSchema(sqlConfig); err != nil {
				return nil, fmt.Errorf("error setting up schema: %w", err)
			}
		}
	}
	// Pre-create namespaces
	var namespaces []*sqlite.NamespaceConfig
	for _, ns := range c.Namespaces {
		namespaces = append(namespaces, sqlite.NewNamespaceConfig(cfg.ClusterMetadata.CurrentClusterName, ns, false))
	}
	if err := sqlite.CreateNamespaces(sqlConfig, namespaces...); err != nil {
		return nil, fmt.Errorf("error creating namespaces: %w", err)
	}

	authorizer, err := authorization.GetAuthorizerFromConfig(&cfg.Global.Authorization)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate authorizer: %w", err)
	}

	claimMapper, err := authorization.GetClaimMapperFromConfig(&cfg.Global.Authorization, c.Logger)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate claim mapper: %w", err)
	}

	serverOpts := []temporal.ServerOption{
		temporal.WithConfig(cfg),
		temporal.ForServices(temporal.Services),
		temporal.WithLogger(c.Logger),
		temporal.WithAuthorizer(authorizer),
		temporal.WithClaimMapper(func(cfg *config.Config) authorization.ClaimMapper {
			return claimMapper
		}),
		temporal.WithDynamicConfigClient(dynamicconfig.NewNoopClient()),
	}

	if len(c.UpstreamOptions) > 0 {
		serverOpts = append(serverOpts, c.UpstreamOptions...)
	}

	if cfg.Global.TLS.Frontend.Server.CertFile != "" {
		provider, err := encryption.NewLocalStoreTlsProvider(&cfg.Global.TLS, metrics.NoopClient.Scope(0), c.Logger, encryption.NewLocalStoreCertProvider)
		if err != nil {
			return nil, fmt.Errorf("unable to instiate tls provider: %w", err)
		}
		serverOpts = append(serverOpts, temporal.WithTLSConfigFactory(provider))
	}

	s := &Server{
		internal:         temporal.NewServer(serverOpts...),
		ui:               c.UIServer,
		frontendHostPort: cfg.PublicClient.HostPort,
		config:           c,
	}

	return s, nil
}

// Start temporal server.
func (s *Server) Start() error {
	go func() {
		if err := s.ui.Start(); err != nil {
			panic(err)
		}
	}()
	return s.internal.Start()
}

// Stop the server.
func (s *Server) Stop() {
	s.ui.Stop()
	s.internal.Stop()
}

// NewClient initializes a client ready to communicate with the Temporal
// server in the target namespace.
func (s *Server) NewClient(ctx context.Context, namespace string) (client.Client, error) {
	return s.NewClientWithOptions(ctx, client.Options{Namespace: namespace})
}

// NewClientWithOptions is the same as NewClient but allows further customization.
//
// To set the client's namespace, use the corresponding field in client.Options.
//
// Note that the HostPort and ConnectionOptions fields of client.Options will always be overridden.
func (s *Server) NewClientWithOptions(ctx context.Context, options client.Options) (client.Client, error) {
	options.HostPort = s.frontendHostPort
	return client.NewClient(options)
}

// FrontendHostPort returns the host:port for this server.
//
// When constructing a Temporalite client from within the same process,
// NewClient or NewClientWithOptions should be used instead.
func (s *Server) FrontendHostPort() string {
	return s.frontendHostPort
}

func timeoutFromContext(ctx context.Context, defaultTimeout time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline.Sub(time.Now())
	}
	return defaultTimeout
}
