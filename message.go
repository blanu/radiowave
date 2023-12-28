package radiowave

type Message interface {
	ToBytes() []byte
}

type MessageFactory interface {
	FromBytes(bytes []byte) (*Message, error)
}
