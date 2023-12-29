package radiowave

type Message interface {
	ToBytes() []byte
}

type MessageFactory[M Message] interface {
	FromBytes(bytes []byte) (*M, error)
}
