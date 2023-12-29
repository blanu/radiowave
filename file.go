package radiowave

import (
	"encoding/binary"
	"io"
)

type File struct {
	factory       MessageFactory
	inputStream   io.ReadCloser
	outputStream  io.WriteCloser
	InputChannel  chan Message
	OutputChannel chan Message
	CloseChannel  chan bool
}

func NewFile(factory MessageFactory, input io.ReadCloser, output io.WriteCloser) File {
	inputChannel := make(chan Message)
	outputChannel := make(chan Message)
	closeChannel := make(chan bool)

	file := File{factory, input, output, inputChannel, outputChannel, closeChannel}
	go file.pumpInputChannel()
	go file.pumpOutputStream()
	go file.cleanup()

	return file
}

// Messages are in a format consisting of a payload prefixed by a varint-encoded length.
// Please note that there are multiple known formats for varint encoding.
// The format used here is not the one from the Go standard library.
// This function reads messages from the resource's stdout
func (f File) ReadMessage() (Message, error) {
	prefix, prefixReadError := fullReadFile(f.inputStream, 1)
	if prefixReadError != nil {
		return nil, prefixReadError
	}
	varintCount := int(prefix[0])

	compressedBuffer, compressedReadError := fullReadFile(f.inputStream, varintCount)
	if compressedReadError != nil {
		return nil, compressedReadError
	}

	uncompressedBuffer, unpackError := unpackVarintData(compressedBuffer)
	if unpackError != nil {
		return nil, unpackError
	}

	payloadCount := dataToInt(uncompressedBuffer)
	payload, payloadReadError := fullReadFile(f.inputStream, payloadCount)
	if payloadReadError != nil {
		return nil, payloadReadError
	}

	completeMessage := make([]byte, 0)
	completeMessage = append(completeMessage, prefix...)
	completeMessage = append(completeMessage, compressedBuffer...)
	completeMessage = append(completeMessage, payload...)

	return f.factory.FromBytes(completeMessage)
}

func (f File) WriteMessage(message Message) error {
	payload := message.ToBytes()

	length := uint64(len(payload))
	compressedBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(compressedBuffer, length)

	for len(compressedBuffer) > 0 {
		if compressedBuffer[0] == 0 {
			compressedBuffer = compressedBuffer[1:]
		} else {
			break
		}
	}

	prefix := byte(len(compressedBuffer))
	completeMessage := make([]byte, 0)
	completeMessage = append(completeMessage, prefix)
	completeMessage = append(completeMessage, compressedBuffer...)
	completeMessage = append(completeMessage, payload...)

	return fullWriteFile(f.outputStream, completeMessage)
}

// We need this to ensure that there are no short reads from the file.
func fullReadFile(conn io.ReadCloser, size int) ([]byte, error) {
	buffer := make([]byte, size)

	totalRead := 0
	for totalRead < size {
		numRead, readError := conn.Read(buffer)
		if readError != nil {
			return nil, readError
		}

		totalRead += numRead
	}

	return buffer, nil
}

// We need this to ensure that there are no short writes to the file.
func fullWriteFile(conn io.WriteCloser, message []byte) error {
	totalWritten := 0
	for totalWritten < len(message) {
		numWritten, writeError := conn.Write(message[totalWritten:])
		if writeError != nil {
			return writeError
		}

		totalWritten += numWritten
	}

	return nil
}

func (f File) pumpInputChannel() {
	// Read all of the messages from the outside world
	for wave := range f.InputChannel {
		// We have a message from the outside world.
		// Write it to the network.
		f.WriteMessage(wave)
	}
}

func (f File) pumpOutputStream() {
	for {
		wave, readError := f.ReadMessage()
		if readError != nil {
			break
		}

		f.OutputChannel <- wave
	}
}

func (f File) cleanup() {
	<-f.CloseChannel

	f.inputStream.Close()
	f.outputStream.Close()
	close(f.InputChannel)
	close(f.OutputChannel)
	close(f.CloseChannel)
}
