package temporaltest_test

import (
	"context"
	"testing"
	"time"

	"github.com/DataDog/temporalite/internal/examples/helloworld"
	"github.com/DataDog/temporalite/server"
	"github.com/DataDog/temporalite/server/temporaltest"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func TestNewServer(t *testing.T) {
	// Create test Temporal server
	srv := temporaltest.NewServer(server.WithNamespaces("default"))
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get a new client
	c, err := srv.NewClient(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	w := worker.New(c, "example", worker.Options{})
	helloworld.RegisterWorkflowsAndActivities(w)

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	wfr, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{TaskQueue: "example"}, helloworld.Greet, "world")
	if err != nil {
		t.Fatal(err)
	}

	var resp string
	if err := wfr.Get(ctx, &resp); err != nil {
		t.Fatal(err)
	}

	if resp != "Hello world" {
		t.Fatalf("unexpected result: %q", resp)
	}
}
