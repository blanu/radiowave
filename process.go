package radiowave

import (
	"context"
	"errors"
	"os/exec"
)

type Process struct {
	factory MessageFactory
	cancel  context.CancelFunc
	file    File

	InputChannel  chan Message
	OutputChannel chan Message
	CloseChannel  chan bool
}

// Exec attempts to start the resource as a separate process connected to us through stdin/stdout
func Exec(factory MessageFactory, path string) (*Process, error) {
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
	process := Process{factory, cancel, file, file.InputChannel, file.OutputChannel, closeChannel}
	go process.cleanup()

	return &process, nil
}

func (p Process) cleanup() {
	<-p.CloseChannel

	p.cancel()

	p.file.CloseChannel <- true

	close(p.InputChannel)
	close(p.OutputChannel)
	close(p.CloseChannel)
}
