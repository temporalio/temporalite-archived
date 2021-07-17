package temporaltest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DataDog/temporalite/server"
	"github.com/DataDog/temporalite/server/temporaltest"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func Greet(ctx workflow.Context, subject string) (string, error) {
	var greeting string
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Second,
	}), PickGreeting).Get(ctx, &greeting); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s", greeting, subject), nil
}

func PickGreeting(ctx context.Context) (string, error) {
	return "Hello", nil
}

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
	w.RegisterWorkflow(Greet)
	w.RegisterActivity(PickGreeting)

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	wfr, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{TaskQueue: "example"}, Greet, "world")
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
