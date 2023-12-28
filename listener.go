package radiowave

import (
	"net"
)

type Listener struct {
	factory MessageFactory
	network net.Listener
}

func Listen(factory MessageFactory, source string) (*Listener, error) {
	network, listenError := net.Listen("tcp", source)
	if listenError != nil {
		return nil, listenError
	}

	listener := Listener{factory, network}
	return &listener, nil
}

func (l Listener) Accept() (*Conn, error) {
	network, acceptError := l.network.Accept()
	if acceptError != nil {
		return nil, acceptError
	}

	conn := NewConn(l.factory, network)
	return &conn, nil
}
