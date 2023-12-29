package radiowave

import (
	"context"
	"errors"
	"os/exec"
)

type Process[M Message] struct {
	factory MessageFactory[M]
	cancel  context.CancelFunc
	file    File[M]

	InputChannel  chan M
	OutputChannel chan M
	CloseChannel  chan bool
}

// Exec attempts to start the resource as a separate process connected to us through stdin/stdout
func Exec[M Message](factory MessageFactory[M], path string) (*Process[M], error) {
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

	file := NewFile(factory, resourceOutput, resourceInput)
	closeChannel := make(chan bool)
	process := Process[M]{factory, cancel, file, file.InputChannel, file.OutputChannel, closeChannel}
	go process.wait(resource)
	go process.cleanup()

	return &process, nil
}

func (p Process[M]) wait(resource *exec.Cmd) {
	_ = resource.Wait()

	p.CloseChannel <- true
}

func (p Process[M]) cleanup() {
	<-p.CloseChannel

	p.cancel()

	p.file.CloseChannel <- true

	close(p.InputChannel)
	close(p.OutputChannel)
	close(p.CloseChannel)
}
