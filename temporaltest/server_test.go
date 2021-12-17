// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/DataDog/temporalite/internal/examples/helloworld"
	"github.com/DataDog/temporalite/temporaltest"
)

// to be used in example code
var t *testing.T

func ExampleNewServer_testWorker() {
	// Create test Temporal server and client
	ts := temporaltest.NewServer(temporaltest.WithT(t))
	c := ts.Client()

	// Register a new worker on the `hello_world` task queue
	ts.Worker("hello_world", func(registry worker.Registry) {
		helloworld.RegisterWorkflowsAndActivities(registry)
	})

	// Start test workflow
	wfr, err := c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{TaskQueue: "hello_world"},
		helloworld.Greet,
		"world",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Get workflow result
	var result string
	if err := wfr.Get(context.Background(), &result); err != nil {
		t.Fatal(err)
	}

	// Print result
	fmt.Println(result)
	// Output: Hello world
}

func TestNewServer(t *testing.T) {
	ts := temporaltest.NewServer(temporaltest.WithT(t))

	ts.Worker("hello_world", func(registry worker.Registry) {
		helloworld.RegisterWorkflowsAndActivities(registry)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wfr, err := ts.Client().ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{TaskQueue: "hello_world"},
		helloworld.Greet,
		"world",
	)
	if err != nil {
		t.Fatal(err)
	}

	var result string
	if err := wfr.Get(ctx, &result); err != nil {
		t.Fatal(err)
	}

	if result != "Hello world" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func BenchmarkRunWorkflow(b *testing.B) {
	ts := temporaltest.NewServer()
	defer ts.Stop()

	ts.Worker("hello_world", func(registry worker.Registry) {
		helloworld.RegisterWorkflowsAndActivities(registry)
	})
	c := ts.Client()

	for i := 0; i < b.N; i++ {
		func(b *testing.B) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			wfr, err := c.ExecuteWorkflow(
				ctx,
				client.StartWorkflowOptions{TaskQueue: "hello_world"},
				helloworld.Greet,
				"world",
			)
			if err != nil {
				b.Fatal(err)
			}

			if err := wfr.Get(ctx, nil); err != nil {
				b.Fatal(err)
			}
		}(b)
	}
}

func TestWithSearchAttributes(t *testing.T) {
	// Create test Temporal server and client
	ts := temporaltest.NewServer(temporaltest.WithT(t),
		temporaltest.WithSA(map[string]enums.IndexedValueType{
			"test": enums.INDEXED_VALUE_TYPE_TEXT,
		}))
	c := ts.Client()

	// Register a new worker on the `hello_world` task queue
	ts.Worker("hello_world", func(registry worker.Registry) {
		helloworld.RegisterWorkflowsAndActivities(registry)
	})

	// Start test workflow with search attributes
	wfr, err := c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			TaskQueue:        "hello_world",
			SearchAttributes: map[string]interface{}{"test": "test-value"},
		},
		helloworld.Greet,
		"world",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Get workflow result
	var result string
	if err := wfr.Get(context.Background(), &result); err != nil {
		t.Fatal(err)
	}

	// Print result
	fmt.Println(result)
	// Output: Hello world
	searchAttrs, err := ts.Client().GetSearchAttributes(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the search attributes returned by the server
	// have the search attribute defined in the test
	value, ok := searchAttrs.Keys["test"]
	if ok {
		if value != enums.INDEXED_VALUE_TYPE_TEXT {
			t.Fatal(fmt.Sprintf("search attribute was defined and present, but the value did not match. Expected %s, but got %s", enums.INDEXED_VALUE_TYPE_TEXT, value.String()))
		}
	} else {
		t.Fatal("search attribute was defined, but not returned by the server")
	}
}
