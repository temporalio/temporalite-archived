package liteconfig

import (
	"fmt"
	"net"
)

// Modified from https://github.com/phayes/freeport/blob/95f893ade6f232a5f1511d61735d89b1ae2df543/freeport.go

type portProvider struct {
	listeners []*net.TCPListener
}

// getFreePort asks the kernel for a free open port that is ready to use.
func (p *portProvider) getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		if addr, err = net.ResolveTCPAddr("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("temporalite: failed to get free port: %v", err))
		}
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	p.listeners = append(p.listeners, l)

	return l.Addr().(*net.TCPAddr).Port, nil
}

func (p *portProvider) mustGetFreePort() int {
	port, err := p.getFreePort()
	if err != nil {
		panic(err)
	}
	return port
}

func (p *portProvider) close() error {
	for _, l := range p.listeners {
		if err := l.Close(); err != nil {
			return err
		}
	}
	return nil
}
