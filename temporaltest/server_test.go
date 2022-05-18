// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

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

func TestNewServerWithMutalTls(t *testing.T) {
	testNewServerWithTlsEnabled(t, true)
}

func TestNewServerWithTls(t *testing.T) {
	testNewServerWithTlsEnabled(t, false)
}

func testNewServerWithTlsEnabled(t *testing.T, useMutualTls bool) {
	b, err := temporaltest.GenerateCertificates()
	if err != nil {
		t.Fatal(err)
	}

	caCert, caKey, err := writeCertificate(b.Ca)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(caCert)
	defer os.Remove(caKey)

	clientCert, clientKey, err := writeCertificate(b.Client)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(clientCert)
	defer os.Remove(clientKey)

	serverCert, serverKey, err := writeCertificate(b.Server)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(serverCert)
	defer os.Remove(serverKey)

	kp, err := tls.X509KeyPair(b.Client.CertPem, b.Client.KeyPem)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(b.Ca.CertPem)

	ts := temporaltest.NewServerWithTls(caCert, serverCert, serverKey, useMutualTls, client.Options{
		ConnectionOptions: client.ConnectionOptions{
			TLS: &tls.Config{
				Certificates: []tls.Certificate{kp},
				RootCAs:      pool,
			},
		},
	}, temporaltest.WithT(t))

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

func writeCertificate(cert *temporaltest.Certificate) (string, string, error) {
	file, err := ioutil.TempFile("", "certificate")
	if err != nil {
		return "", "", err
	}

	if _, err := file.Write(cert.CertPem); err != nil {
		defer os.Remove(file.Name())
		return "", "", err
	}

	key, err := ioutil.TempFile("", "key")
	if err != nil {
		defer os.Remove(file.Name())
		return "", "", err
	}

	if _, err := key.Write(cert.KeyPem); err != nil {
		defer os.Remove(file.Name())
		defer os.Remove(key.Name())
		return "", "", err
	}

	return file.Name(), key.Name(), nil
}
