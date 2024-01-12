package radiowave

import (
	"log"
	"net"
)

type Listener[Request Message, Response Message] struct {
	factory MessageFactory[Response]
	network net.Listener
	logger  *log.Logger
}

func Listen[Request Message, Response Message](factory MessageFactory[Response], source string, logger *log.Logger) (*Listener[Request, Response], error) {
	network, listenError := net.Listen("tcp", source)
	if listenError != nil {
		return nil, listenError
	}

	listener := Listener[Request, Response]{factory, network, logger}
	return &listener, nil
}

func (l Listener[Request, Response]) Accept() (*Conn[Request, Response], error) {
	network, acceptError := l.network.Accept()
	if acceptError != nil {
		return nil, acceptError
	}

	conn := newConn[Request, Response](l.factory, network, l.logger)
	return &conn, nil
}
