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

package main_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

func TestMTLSConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run ephemerally in the mtls example folder using the config there. This
	// does expect port 7233 to be free.
	_, thisFile, _, _ := runtime.Caller(0)
	args := []string{
		"run", filepath.Dir(thisFile),
		"start",
		"--ephemeral",
		"--config", ".",
		"--namespace", "default",
		"--log-format", "pretty",
	}
	if !testing.Verbose() {
		args = append(args, "--log-level", "warn")
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	mtlsDir := filepath.Join(thisFile, "../../../internal/examples/mtls")
	cmd.Dir = mtlsDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	t.Logf("Running go with args %v in %v", args, cmd.Dir)
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := kill(cmd); err != nil {
			t.Logf("Process kill failed: %v", err)
		}
	}()

	// Load client cert/key for auth
	clientCert, err := tls.LoadX509KeyPair(
		filepath.Join(mtlsDir, "client-cert.pem"),
		filepath.Join(mtlsDir, "client-key.pem"),
	)
	if err != nil {
		log.Fatal(err)
	}
	// Load server cert for CA check
	serverCAPEM, err := os.ReadFile(filepath.Join(mtlsDir, "server-ca-cert.pem"))
	if err != nil {
		log.Fatal(err)
	}
	serverCAPool := x509.NewCertPool()
	serverCAPool.AppendCertsFromPEM(serverCAPEM)

	// Build client options and try to connect client every 200ms for 10s
	var options client.Options
	options.ConnectionOptions.TLS = &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      serverCAPool,
	}
	var c client.Client
	for i := 0; i < 50; i++ {
		if c, err = client.Dial(options); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Make a call
	resp, err := c.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
		Namespace: "default",
	})
	if err != nil {
		log.Fatal(err)
	} else if resp.NamespaceInfo.State != enums.NAMESPACE_STATE_REGISTERED {
		log.Fatalf("Bad state: %v", resp.NamespaceInfo.State)
	}
}

func kill(cmd *exec.Cmd) error {
	// Have to use taskkill on Windows to avoid custom ctrl+c code
	if runtime.GOOS == "windows" {
		k := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		k.Stdout, k.Stderr = os.Stdout, os.Stderr
		if err := k.Run(); err != nil {
			return fmt.Errorf("taskkill failed: %w", err)
		}
	} else if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("kill failed: %w", err)
	}
	if _, err := cmd.Process.Wait(); err != nil {
		return fmt.Errorf("wait failed: %w", err)
	}
	return nil
}
