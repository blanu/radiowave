package radiowave

import (
	"context"
	"errors"
	"os/exec"
)

type Process[Request Message, Response Message] struct {
	factory MessageFactory[Response]
	cancel  context.CancelFunc
	file    File[Request, Response]

	InputChannel  chan Request
	OutputChannel chan Response
	CloseChannel  chan bool
}

// Exec attempts to start the resource as a separate process connected to us through stdin/stdout
func Exec[Request Message, Response Message](factory MessageFactory[Response], path string) (*Process[Request, Response], error) {
	ctx, cancel := context.WithCancel(context.Background())
	resource := exec.CommandContext(ctx, path)
	resourceInput, inputError := resource.StdinPipe()
	if inputError != nil {
		return nil, inputError
	}
	resourceOutput, outputError := resource.StdoutPipe()
	if outputError != nil {
		return nil, outputError
	}

	startError := resource.Start()
	if startError != nil {
		cancel()
		return nil, errors.New("resource could not be started")
	}

	file := NewFile[Request, Response](factory, resourceOutput, resourceInput)
	closeChannel := make(chan bool)
	process := Process[Request, Response]{factory, cancel, file, file.InputChannel, file.OutputChannel, closeChannel}
	go process.wait(resource)
	go process.cleanup()

	return &process, nil
}

func (p Process[Request, Response]) Write(request Request) {
	p.InputChannel <- request
}

func (p Process[Request, Response]) Read() Response {
	response := <-p.OutputChannel

	return response
}

func (p Process[Request, Response]) Call(request Request) Response {
	p.Write(request)
	return p.Read()
}

func (p Process[Request, Response]) Close() {
	p.CloseChannel <- true
}

func (p Process[Request, Response]) wait(resource *exec.Cmd) {
	_ = resource.Wait()

	p.CloseChannel <- true
}

func (p Process[Request, Response]) cleanup() {
	<-p.CloseChannel

	p.cancel()

	p.file.CloseChannel <- true

	close(p.InputChannel)
	close(p.OutputChannel)
	close(p.CloseChannel)
}
