// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporalite

import (
	"context"
	"fmt"
	"sync"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/temporal"
	"google.golang.org/grpc"

	"github.com/DataDog/temporalite/internal/liteconfig"
)

// Server wraps a temporal.Server.
type Server struct {
	internal         *temporal.Server
	frontendHostPort string
	config           *liteconfig.Config
	setupWaitGroup   sync.WaitGroup
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

// Start temporal server.
func (s *Server) Start() error {
	if len(s.config.Namespaces) > 0 {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			nsClient, err := s.newNamespaceClient(ctx)
			if err != nil {
				panic(err)
			}
			defer nsClient.Close()

			// Wait for each namespace to be ready
			for _, ns := range s.config.Namespaces {
				c, err := s.newClient(ctx, client.Options{Namespace: ns})
				if err != nil {
					panic(err)
				}

				// Wait up to 1 minute (20ms backoff x 3000 attempts)
				var (
					maxAttempts = 3000
					backoff     = 20 * time.Millisecond
				)
				for i := 0; i < maxAttempts; i++ {
					_, err = c.ListOpenWorkflow(ctx, &workflowservice.ListOpenWorkflowExecutionsRequest{
						Namespace: ns,
					})
					if err == nil {
						if _, err := c.DescribeTaskQueue(ctx, "_404", enumspb.TASK_QUEUE_TYPE_UNSPECIFIED); err == nil {
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

// Stop the server.
func (s *Server) Stop() {
	s.internal.Stop()
}

// NewClient initializes a client ready to communicate with the Temporal
// server in the target namespace.
func (s *Server) NewClient(ctx context.Context, namespace string) (client.Client, error) {
	return s.newClientBlocking(ctx, client.Options{Namespace: namespace})
}

// NewClientWithOptions is the same as NewClient but allows further customization.
//
// To set the client's namespace, use the corresponding field in client.Options.
//
// Note that the HostPort and ConnectionOptions fields of client.Options will always be overridden.
func (s *Server) NewClientWithOptions(ctx context.Context, options client.Options) (client.Client, error) {
	return s.newClientBlocking(ctx, options)
}

func (s *Server) newClientBlocking(ctx context.Context, options client.Options) (client.Client, error) {
	s.setupWaitGroup.Wait()
	return s.newClient(ctx, options)
}

func (s *Server) newClient(ctx context.Context, options client.Options) (client.Client, error) {
	options.HostPort = s.frontendHostPort
	options.ConnectionOptions = client.ConnectionOptions{
		DisableHealthCheck: false,
		HealthCheckTimeout: timeoutFromContext(ctx, time.Minute),
	}
	return client.NewClient(options)
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
