package radiowave

import (
	"net"
)

type Listener[Request Message, Response Message] struct {
	factory MessageFactory[Response]
	network net.Listener
}

func Listen[Request Message, Response Message](factory MessageFactory[Response], source string) (*Listener[Request, Response], error) {
	network, listenError := net.Listen("tcp", source)
	if listenError != nil {
		return nil, listenError
	}

	listener := Listener[Request, Response]{factory, network}
	return &listener, nil
}

func (l Listener[Request, Response]) Accept() (*Conn[Request, Response], error) {
	network, acceptError := l.network.Accept()
	if acceptError != nil {
		return nil, acceptError
	}

	conn := newConn[Request, Response](l.factory, network)
	return &conn, nil
}
