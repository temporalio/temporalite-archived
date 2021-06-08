package server

import (
	"fmt"

	"github.com/DataDog/temporalite/internal/liteconfig"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/temporal"
)

type Server struct {
	internal         *temporal.Server
	frontendHostPort string
}

type Option interface {
	apply(*liteconfig.Config)
}

func New(opts ...Option) (*Server, error) {
	c, err := liteconfig.NewDefaultConfig()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	cfg := liteconfig.Convert(c)

	authorizer, err := authorization.GetAuthorizerFromConfig(&cfg.Global.Authorization)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate authorizer: %w", err)
	}

	claimMapper, err := authorization.GetClaimMapperFromConfig(&cfg.Global.Authorization, c.Logger)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate claim mapper: %w", err)
	}

	return &Server{
		internal: temporal.NewServer(
			temporal.WithConfig(cfg),
			temporal.ForServices(temporal.Services),
			temporal.WithLogger(c.Logger),
			temporal.InterruptOn(temporal.InterruptCh()),
			temporal.WithAuthorizer(authorizer),
			temporal.WithClaimMapper(func(cfg *config.Config) authorization.ClaimMapper {
				return claimMapper
			}),
			temporal.WithDynamicConfigClient(dynamicconfig.NewNoopClient()),
		),
		frontendHostPort: cfg.PublicClient.HostPort,
	}, nil
}

func (s *Server) Start() error {
	return s.internal.Start()
}

func (s *Server) Stop() {
	s.internal.Stop()
}
