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
