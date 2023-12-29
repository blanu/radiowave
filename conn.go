package radiowave

import (
	"encoding/binary"
	"net"
)

type Conn[Request Message, Response Message] struct {
	factory MessageFactory[Response]
	network net.Conn

	InputChannel  chan Request
	OutputChannel chan Response
	CloseChannel  chan bool
}

func Dial[Request Message, Response Message](factory MessageFactory[Response], destination string) (*Conn[Request, Response], error) {
	network, dialError := net.Dial("tcp", destination)
	if dialError != nil {
		return nil, dialError
	}

	wrapped := newConn[Request, Response](factory, network)
	return &wrapped, nil
}

func newConn[Request Message, Response Message](factory MessageFactory[Response], network net.Conn) Conn[Request, Response] {
	inputChannel := make(chan Request)
	outputChannel := make(chan Response)
	closeChannel := make(chan bool)

	conn := Conn[Request, Response]{factory, network, inputChannel, outputChannel, closeChannel}

	go conn.pumpInputChannel()
	go conn.pumpNetwork()
	go conn.cleanup()

	return conn
}

func (c Conn[Request, Response]) Call(request Request) Response {
	c.InputChannel <- request
	response := <-c.OutputChannel

	return response
}

func (c Conn[Request, Response]) Close() {
	c.CloseChannel <- true
}

func (c Conn[Request, Response]) readMessage() (*Response, error) {
	prefix, prefixReadError := c.fullRead(c.network, 1)
	if prefixReadError != nil {
		return nil, prefixReadError
	}
	varintCount := int(prefix[0])

	compressedBuffer, compressedReadError := c.fullRead(c.network, varintCount)
	if compressedReadError != nil {
		return nil, compressedReadError
	}

	uncompressedBuffer, unpackError := unpackVarintData(compressedBuffer)
	if unpackError != nil {
		return nil, unpackError
	}

	payloadCount := dataToInt(uncompressedBuffer)
	payload, payloadReadError := c.fullRead(c.network, payloadCount)
	if payloadReadError != nil {
		return nil, payloadReadError
	}

	completeMessage := make([]byte, 0)
	completeMessage = append(completeMessage, prefix...)
	completeMessage = append(completeMessage, compressedBuffer...)
	completeMessage = append(completeMessage, payload...)

	return c.factory.FromBytes(completeMessage)
}

func (c Conn[Request, Response]) writeMessage(message Request) error {
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

	return c.fullWrite(c.network, completeMessage)
}

// We need this to ensure that there are no short reads from the connection.
func (c Conn[Request, Response]) fullRead(conn net.Conn, size int) ([]byte, error) {
	buffer := make([]byte, size)

	totalRead := 0
	for totalRead < size {
		numRead, readError := conn.Read(buffer)
		if readError != nil {
			c.CloseChannel <- true
			return nil, readError
		}

		totalRead += numRead
	}

	return buffer, nil
}

// We need this to ensure that there are no short writes to the connection.
func (c Conn[Request, Response]) fullWrite(conn net.Conn, message []byte) error {
	totalWritten := 0
	for totalWritten < len(message) {
		numWritten, writeError := conn.Write(message[totalWritten:])
		if writeError != nil {
			c.CloseChannel <- true
			return writeError
		}

		totalWritten += numWritten
	}

	return nil
}

func (c Conn[Request, Response]) pumpInputChannel() {
	// Read all the messages from the outside world.
	for wave := range c.InputChannel {
		// We have a message from the outside world.
		// Write it to the network.
		writeError := c.writeMessage(wave)
		if writeError != nil {
			break
		}
	}
}

func (c Conn[Request, Response]) pumpNetwork() {
	for {
		wave, readError := c.readMessage()
		if readError != nil {
			break
		}

		c.OutputChannel <- *wave
	}
}

func (c Conn[Request, Response]) cleanup() {
	<-c.CloseChannel

	_ = c.network.Close()

	close(c.InputChannel)
	close(c.OutputChannel)
	close(c.CloseChannel)
}
