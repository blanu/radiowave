package radiowave

type Resource[M Message] interface {
	Call(M) M
	Close()
}
