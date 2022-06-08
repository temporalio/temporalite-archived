// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest

import (
	"github.com/DataDog/temporalite"
	"go.temporal.io/sdk/client"
	"testing"
)

type TestServerOption interface {
	apply(*TestServer)
}

// WithT directs all worker and client logs to the test logger.
//
// If this option is specified, then server will automatically be stopped when the
// test completes.
func WithT(t *testing.T) TestServerOption {
	return newApplyFuncContainer(func(server *TestServer) {
		server.t = t
	})
}

func WithClientOptions(o client.Options) TestServerOption {
	return newApplyFuncContainer(func(server *TestServer) {
		server.defaultClientOptions = o
	})
}

func WithTls(caCertificates []string, certificate, key string, useMtls bool) TestServerOption {
	return newApplyFuncContainer(func(server *TestServer) {
		server.serverOptions = append(server.serverOptions, temporalite.WithTLSOptions(caCertificates, certificate, key, useMtls))
	})
}

type applyFuncContainer struct {
	applyInternal func(*TestServer)
}

func (fso *applyFuncContainer) apply(ts *TestServer) {
	fso.applyInternal(ts)
}

func newApplyFuncContainer(apply func(*TestServer)) *applyFuncContainer {
	return &applyFuncContainer{
		applyInternal: apply,
	}
}
