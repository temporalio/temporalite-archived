// MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2021 Datadog, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"github.com/temporalio/temporalite/internal/liteconfig"
)

func TestGetDynamicConfigValues(t *testing.T) {
	assertBadVal := func(v string) {
		if _, err := getDynamicConfigValues([]string{v}); err == nil {
			t.Fatalf("expected error for %v", v)
		}
	}
	type v map[string][]interface{}
	assertGoodVals := func(expected v, in ...string) {
		actualVals, err := getDynamicConfigValues(in)
		if err != nil {
			t.Fatal(err)
		}
		actual := make(v, len(actualVals))
		for k, vals := range actualVals {
			for _, val := range vals {
				actual[string(k)] = append(actual[string(k)], val.Value)
			}
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("not equal, expected - actual: %v - %v", expected, actual)
		}
	}

	assertBadVal("foo")
	assertBadVal("foo=")
	assertBadVal("foo=bar")
	assertBadVal("foo=123a")

	assertGoodVals(v{"foo": {123.0}}, "foo=123")
	assertGoodVals(
		v{"foo": {123.0, []interface{}{"123", false}}, "bar": {"baz"}, "qux": {true}},
		"foo=123", `bar="baz"`, "qux=true", `foo=["123", false]`,
	)
}

func newServerAndClientOpts(port int, customArgs ...string) ([]string, client.Options) {
	args := []string{
		"temporalite",
		"start",
		"--namespace", "default",
		// Use noop logger to avoid fatal logs failing tests on shutdown signal.
		"--log-format", "noop",
		"--headless",
		"--port", strconv.Itoa(port),
	}

	return append(args, customArgs...), client.Options{
		HostPort:  fmt.Sprintf("localhost:%d", port),
		Namespace: "temporal-system",
	}
}

func assertServerHealth(t *testing.T, ctx context.Context, opts client.Options) {
	var (
		c         client.Client
		clientErr error
	)
	for i := 0; i < 50; i++ {
		if c, clientErr = client.Dial(opts); clientErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if clientErr != nil {
		t.Error(clientErr)
	}

	if _, err := c.CheckHealth(ctx, nil); err != nil {
		t.Error(err)
	}

	// Check for pollers on a system task queue to ensure that the worker service is running.
	for {
		if ctx.Err() != nil {
			t.Error(ctx.Err())
			break
		}
		resp, err := c.DescribeTaskQueue(ctx, "temporal-sys-tq-scanner-taskqueue-0", enums.TASK_QUEUE_TYPE_WORKFLOW)
		if err != nil {
			t.Error(err)
		}
		if len(resp.GetPollers()) > 0 {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func TestCreateDataDirectory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testUserHome := filepath.Join(os.TempDir(), "temporalite_test", t.Name())
	t.Cleanup(func() {
		if err := os.RemoveAll(testUserHome); err != nil {
			fmt.Println("error cleaning up temp dir:", err)
		}
	})
	// Set user home for all supported operating systems
	t.Setenv("AppData", testUserHome)         // Windows
	t.Setenv("HOME", testUserHome)            // macOS
	t.Setenv("XDG_CONFIG_HOME", testUserHome) // linux
	// Verify that worked
	configDir, _ := os.UserConfigDir()
	if !strings.HasPrefix(configDir, testUserHome) {
		t.Fatalf("expected config dir %q to be inside user home directory %q", configDir, testUserHome)
	}

	temporaliteCLI := buildCLI()
	// Don't call os.Exit
	temporaliteCLI.ExitErrHandler = func(_ *cli.Context, _ error) {}

	portProvider := liteconfig.NewPortProvider()
	var (
		port1 = portProvider.MustGetFreePort()
		port2 = portProvider.MustGetFreePort()
		port3 = portProvider.MustGetFreePort()
	)
	portProvider.Close()

	t.Run("default db path", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		args, clientOpts := newServerAndClientOpts(port1)

		go func() {
			if err := temporaliteCLI.RunContext(ctx, args); err != nil {
				fmt.Println("Server closed with error:", err)
			}
		}()

		assertServerHealth(t, ctx, clientOpts)

		// If the rest of this test case passes but this assertion fails,
		// there may have been a breaking change in the liteconfig package
		// related to how the default db file path is calculated.
		if _, err := os.Stat(filepath.Join(configDir, "temporalite", "db", "default.db")); err != nil {
			t.Errorf("error checking for default db file: %s", err)
		}
	})

	t.Run("custom db path -- missing directory", func(t *testing.T) {
		customDBPath := filepath.Join(testUserHome, "foo", "bar", "baz.db")
		args, _ := newServerAndClientOpts(
			port2, "-f", customDBPath,
		)
		if err := temporaliteCLI.RunContext(ctx, args); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				t.Errorf("expected error %q, got %q", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), filepath.Dir(customDBPath)) {
				t.Errorf("expected error %q to contain string %q", err, filepath.Dir(customDBPath))
			}
		} else {
			t.Error("no error when directory missing")
		}
	})

	t.Run("custom db path -- existing directory", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		args, clientOpts := newServerAndClientOpts(
			port3, "-f", filepath.Join(testUserHome, "foo.db"),
		)

		go func() {
			if err := temporaliteCLI.RunContext(ctx, args); err != nil {
				fmt.Println("Server closed with error:", err)
			}
		}()

		assertServerHealth(t, ctx, clientOpts)
	})
}
