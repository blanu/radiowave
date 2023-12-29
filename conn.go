package radiowave

import (
	"encoding/binary"
	"net"
)

type Conn[M Message] struct {
	factory MessageFactory[M]
	network net.Conn

	InputChannel  chan M
	OutputChannel chan M
	CloseChannel  chan bool
}

func Dial[M Message](factory MessageFactory[M], destination string) (*Conn[M], error) {
	network, dialError := net.Dial("tcp", destination)
	if dialError != nil {
		return nil, dialError
	}

	wrapped := newConn[M](factory, network)
	return &wrapped, nil
}

func newConn[M Message](factory MessageFactory[M], network net.Conn) Conn[M] {
	inputChannel := make(chan M)
	outputChannel := make(chan M)
	closeChannel := make(chan bool)

	conn := Conn[M]{factory, network, inputChannel, outputChannel, closeChannel}

	go conn.pumpInputChannel()
	go conn.pumpNetwork()
	go conn.cleanup()

	return conn
}

func (c Conn[M]) ReadMessage() (*M, error) {
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

func (c Conn[M]) WriteMessage(message Message) error {
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
func (c Conn[M]) fullRead(conn net.Conn, size int) ([]byte, error) {
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
func (c Conn[M]) fullWrite(conn net.Conn, message []byte) error {
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

func (c Conn[M]) pumpInputChannel() {
	// Read all the messages from the outside world.
	for wave := range c.InputChannel {
		// We have a message from the outside world.
		// Write it to the network.
		writeError := c.WriteMessage(wave)
		if writeError != nil {
			break
		}
	}
}

func (c Conn[M]) pumpNetwork() {
	for {
		wave, readError := c.ReadMessage()
		if readError != nil {
			break
		}

		c.OutputChannel <- *wave
	}
}

func (c Conn[M]) cleanup() {
	<-c.CloseChannel

	_ = c.network.Close()

	close(c.InputChannel)
	close(c.OutputChannel)
	close(c.CloseChannel)
}
