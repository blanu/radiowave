package radiowave

import (
	"net"
)

type Listener[M Message] struct {
	factory MessageFactory[M]
	network net.Listener
}

func Listen[M Message](factory MessageFactory[M], source string) (*Listener[M], error) {
	network, listenError := net.Listen("tcp", source)
	if listenError != nil {
		return nil, listenError
	}

	listener := Listener[M]{factory, network}
	return &listener, nil
}

func (l Listener[M]) Accept() (*Conn[M], error) {
	network, acceptError := l.network.Accept()
	if acceptError != nil {
		return nil, acceptError
	}

	conn := newConn(l.factory, network)
	return &conn, nil
}
