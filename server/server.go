package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/temporalite/internal/liteconfig"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/temporal"
	"google.golang.org/grpc"
)

type Server struct {
	internal         *temporal.Server
	frontendHostPort string
	config           *liteconfig.Config
	setupWaitGroup   sync.WaitGroup
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

	s := &Server{
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
		config:           c,
	}
	s.setupWaitGroup.Add(1)

	return s, nil
}

func (s *Server) Start() error {
	if len(s.config.Namespaces) > 0 {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
			defer cancel()
			nsClient, err := s.newNamespaceClient(ctx)
			if err != nil {
				panic(err)
			}
			defer nsClient.Close()

			// Create namespaces
			var errNamespaceExists *serviceerror.NamespaceAlreadyExists
			for _, ns := range s.config.Namespaces {
				if err := nsClient.Register(ctx, &workflowservice.RegisterNamespaceRequest{
					Namespace:                        ns,
					WorkflowExecutionRetentionPeriod: &s.config.DefaultNamespaceRetentionPeriod,
				}); err != nil && !errors.As(err, &errNamespaceExists) {
					panic(err)
				}
			}

			// Wait for each namespace to be ready
			for _, ns := range s.config.Namespaces {
				c, err := s.newClient(context.Background(), ns)
				if err != nil {
					panic(err)
				}

				// Wait up to 1 minute (20ms backoff x 3000 attempts)
				var (
					maxAttempts = 3000
					backoff     = 20 * time.Millisecond
				)
				for i := 0; i < maxAttempts; i++ {
					_, err = c.ListOpenWorkflow(context.Background(), &workflowservice.ListOpenWorkflowExecutionsRequest{
						Namespace: ns,
					})
					if err == nil {
						if _, err := c.DescribeTaskQueue(context.Background(), "_404", enumspb.TASK_QUEUE_TYPE_UNSPECIFIED); err == nil {
							fmt.Println(err)
							break
						}
					}
					time.Sleep(backoff)
				}
				if err != nil {
					panic(fmt.Sprintf("could not connect to namespace %q: %s", ns, err))
				}

				c.Close()
			}

			s.setupWaitGroup.Done()
		}()
	} else {
		s.setupWaitGroup.Done()
	}

	return s.internal.Start()
}

func (s *Server) Stop() {
	s.internal.Stop()
}

func (s *Server) NewClient(ctx context.Context, namespace string) (client.Client, error) {
	s.setupWaitGroup.Wait()
	return s.newClient(ctx, namespace)
}

func (s *Server) newClient(ctx context.Context, namespace string) (client.Client, error) {
	return client.NewClient(client.Options{
		Namespace: namespace,
		HostPort:  s.frontendHostPort,
		ConnectionOptions: client.ConnectionOptions{
			DisableHealthCheck: false,
			HealthCheckTimeout: timeoutFromContext(ctx, time.Minute),
		},
	})
}

func (s *Server) newNamespaceClient(ctx context.Context) (client.NamespaceClient, error) {
	if err := s.healthCheckFrontend(ctx); err != nil {
		return nil, err
	}
	return client.NewNamespaceClient(client.Options{
		HostPort: s.frontendHostPort,
		ConnectionOptions: client.ConnectionOptions{
			DisableHealthCheck: false,
			HealthCheckTimeout: timeoutFromContext(ctx, time.Minute),
		},
	})
}

func timeoutFromContext(ctx context.Context, defaultTimeout time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline.Sub(time.Now())
	}
	return defaultTimeout
}

func (s *Server) healthCheckFrontend(ctx context.Context) error {
	if _, err := grpc.DialContext(ctx, s.frontendHostPort, grpc.WithInsecure(), grpc.WithBlock()); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}
