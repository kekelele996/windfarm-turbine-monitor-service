package scada

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// Frame is one SCADA protocol frame.
type Frame struct {
	Address uint16
	Command byte
	Payload []byte
}

const (
	frameMagic = 0xA5
	maxPayload = 255
)

// WriteFrame serializes a frame with a simple magic/len/checksum framing.
func WriteFrame(w io.Writer, f Frame) error {
	if len(f.Payload) > maxPayload {
		return fmt.Errorf("payload too large: %d", len(f.Payload))
	}
	header := make([]byte, 6)
	header[0] = frameMagic
	binary.BigEndian.PutUint16(header[1:3], f.Address)
	header[3] = f.Command
	header[4] = byte(len(f.Payload))
	header[5] = checksum(header[:5])
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := w.Write(f.Payload); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}
	return nil
}

// ReadFrame parses one frame from a buffered reader.
func ReadFrame(r *bufio.Reader) (Frame, error) {
	header := make([]byte, 6)
	if _, err := io.ReadFull(r, header); err != nil {
		return Frame{}, fmt.Errorf("read header: %w", err)
	}
	if header[0] != frameMagic {
		return Frame{}, fmt.Errorf("bad magic byte 0x%x", header[0])
	}
	if header[5] != checksum(header[:5]) {
		return Frame{}, fmt.Errorf("header checksum mismatch")
	}
	f := Frame{
		Address: binary.BigEndian.Uint16(header[1:3]),
		Command: header[3],
	}
	n := int(header[4])
	f.Payload = make([]byte, n)
	if _, err := io.ReadFull(r, f.Payload); err != nil {
		return Frame{}, fmt.Errorf("read payload: %w", err)
	}
	return f, nil
}

func checksum(b []byte) byte {
	var sum byte
	for _, c := range b {
		sum ^= c
	}
	return sum
}
