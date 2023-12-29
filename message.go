package radiowave

type Message interface {
	ToBytes() []byte
}

type MessageFactory[Response Message] interface {
	FromBytes(bytes []byte) (*Response, error)
}
