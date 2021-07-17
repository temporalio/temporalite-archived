package server_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DataDog/temporalite/server"
	"github.com/DataDog/temporalite/server/temporaltest"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

var ts *server.Server

func TestMain(m *testing.M) {
	s := temporaltest.NewServer(server.WithNamespaces("default"))
	defer s.Stop()

	ts = s

	code := m.Run()
	os.Exit(code)
}

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

func BenchmarkRunWorkflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		func(b *testing.B) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			c, err := ts.NewClient(ctx, "default")
			if err != nil {
				b.Fatal(err)
			}
			defer c.Close()

			w := worker.New(c, "example", worker.Options{})
			w.RegisterWorkflow(Greet)
			w.RegisterActivity(PickGreeting)

			if err := w.Start(); err != nil {
				panic(err)
			}
			defer w.Stop()

			wfr, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{TaskQueue: "example"}, Greet, "world")
			if err != nil {
				b.Fatal(err)
			}

			var resp string
			if err := wfr.Get(ctx, &resp); err != nil {
				b.Fatal(err)
			}

			if resp != "Hello world" {
				b.Fatalf("unexpected result: %q", resp)
			}
		}(b)
	}
}
