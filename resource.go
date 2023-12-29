package radiowave

type Resource[Request Message, Response Message] interface {
	Call(Request) Response
	Close()
}
