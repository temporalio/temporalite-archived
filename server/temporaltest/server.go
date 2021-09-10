// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest

import (
	"github.com/DataDog/temporalite/server"
	tlog "go.temporal.io/server/common/log"
	"go.uber.org/zap"
)

func NewServer(opts ...server.Option) *server.Server {
	opts = append(opts, 
		server.WithPersistenceDisabled(),
                server.WithFrontendPort(0),
                server.WithDynamicPorts(),
                server.WithLogger(tlog.NewZapLogger(zap.NewNop())),	
	)
	s, err := server.New(opts...)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := s.Start(); err != nil {
			panic(err)
		}
	}()

	return s
}
