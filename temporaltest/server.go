// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/log"
	"go.uber.org/zap"

	"github.com/DataDog/temporalite"
)

// A TestServer is a Temporal server listening on a system-chosen port on the
// local loopback interface, for use in end-to-end tests.
type TestServer struct {
	server               *temporalite.Server
	defaultTestNamespace string
	defaultClient        client.Client
	clients              []client.Client
}

// Client returns a Temporal client configured for making requests to the server.
// It is configured to use a pre-registered test namespace and will
// be closed on TestServer.Stop.
func (ts *TestServer) Client() client.Client {
	if ts.defaultClient == nil {
		ts.defaultClient = ts.NewClientWithOptions(client.Options{})
	}
	return ts.defaultClient
}

// NewClientWithOptions returns a Temporal client configured for making requests to the server.
// If no namespace option is set it will use a pre-registered test namespace.
// The returned client will be closed on TestServer.Stop.
func (ts *TestServer) NewClientWithOptions(opts client.Options) client.Client {
	if opts.Namespace == "" {
		opts.Namespace = ts.defaultTestNamespace
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := ts.server.NewClientWithOptions(ctx, opts)
	if err != nil {
		panic(fmt.Errorf("error creating client: %w", err))
	}

	ts.clients = append(ts.clients, c)

	return c
}

// Stop closes test clients and shuts down the server.
func (ts *TestServer) Stop() {
	for _, c := range ts.clients {
		c.Close()
	}
	ts.server.Stop()
}

// NewServer starts and returns a new TestServer. The caller should call Stop
// when finished, to shut it down.
func NewServer() *TestServer {
	rand.Seed(time.Now().UnixNano())
	testNamespace := fmt.Sprintf("temporaltest-%d", rand.Intn(999999))

	s, err := temporalite.NewServer(
		temporalite.WithNamespaces(testNamespace),
		temporalite.WithPersistenceDisabled(),
		temporalite.WithDynamicPorts(),
		temporalite.WithLogger(log.NewZapLogger(zap.NewNop())),
	)
	if err != nil {
		panic(fmt.Errorf("error creating server: %w", err))
	}

	go func() {
		if err := s.Start(); err != nil {
			panic(fmt.Errorf("error starting server: %w", err))
		}
	}()

	return &TestServer{
		server:               s,
		defaultTestNamespace: testNamespace,
	}
}
