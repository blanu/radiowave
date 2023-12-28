package radiowave

import (
	"encoding/binary"
	"errors"
)

// This is one phase of the varint decoder. It takes a compressed length (1-8 bytes) and makes an 8-byte length.
func unpackVarintData(buffer []byte) ([]byte, error) {
	if buffer == nil {
		return nil, errors.New("buffer was nil")
	}

	if len(buffer) == 0 {
		return nil, errors.New("buffer was empty")
	}

	count := len(buffer)
	gap := 8 - count

	uint64Bytes := make([]byte, 8)
	copy(uint64Bytes[gap:], buffer[:count])

	return uint64Bytes, nil
}

// This is one phase of the varint decoder. It takes an 8-byte length and returns an int.
func dataToInt(buffer []byte) int {
	uintValue := binary.BigEndian.Uint64(buffer)
	intValue := int(uintValue)
	return intValue
}
