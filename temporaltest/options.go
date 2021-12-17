// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest

import (
	"testing"

	"go.temporal.io/api/enums/v1"
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

// WithSA will ensure that the test server is initialized with the passed search attributes
func WithSA(searchAttributes map[string]enums.IndexedValueType) TestServerOption {
	return newApplyFuncContainer(func(server *TestServer) {
		server.searchAttributes = searchAttributes
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
