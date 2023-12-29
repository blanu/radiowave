package radiowave

type Resource[Request Message, Response Message] interface {
	Write(Request)
	Read() Response
	Call(Request) Response
	Close()
}
