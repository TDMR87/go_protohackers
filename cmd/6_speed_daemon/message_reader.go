package main

import (
	"fmt"
	"io"
)

type MessageDecoderFunc func(*MessageReader) (any, bool, error)

type MessageReader struct {
	reader          io.Reader
	buf             []byte
	messageDecoders map[byte]MessageDecoderFunc
}

func NewMessageReader(r io.Reader) *MessageReader {
	reader := MessageReader{
		reader:          r,
		buf:             make([]byte, 0, 4096),
		messageDecoders: make(map[byte]MessageDecoderFunc),
	}

	reader.messageDecoders[WantHeartBeat{}.Type()] =
		func(reader *MessageReader) (any, bool, error) {
			return decodeMessageFromBytes(WantHeartBeat{}.Decode, WantHeartBeat{}.Size(), reader)
		}

	return &reader
}

// NextMessage reads and returns the next message from the underlying reader (e.g. net.Conn).
func (f *MessageReader) NextMessage() (msg any, err error) {
	for {
		if len(f.buf) < 1 {
			err = f.fillBuffer()
			if err != nil {
				return nil, err
			}
		}

		msgType := f.buf[0]
		decodeMsg, ok := f.messageDecoders[msgType]

		if !ok {
			return nil, fmt.Errorf("unknown message type %#x", msgType)
		}

		msg, ok, err := decodeMsg(f)
		if err != nil {
			return nil, err
		}

		if !ok {
			f.fillBuffer()
			continue
		}

		return msg, nil
	}
}

func decodeMessageFromBytes[T any](
	decoderFunc func(b []byte) (T, error),
	bytesLen int,
	reader *MessageReader,
) (msg T, ok bool, err error) {

	var zero T
	if len(reader.buf) < bytesLen {
		err = reader.fillBuffer()
		if err != nil {
			return zero, false, err
		}
		return zero, false, nil
	}
	msg, err = decoderFunc(reader.buf[:bytesLen])
	if err != nil {
		return zero, false, err
	}
	reader.buf = reader.buf[bytesLen:]
	return msg, true, nil
}

func (f *MessageReader) fillBuffer() error {
	tempBuf := make([]byte, 512)
	n, err := f.reader.Read(tempBuf)
	if err != nil {
		return err
	}
	f.buf = append(f.buf, tempBuf[:n]...)
	return nil
}
