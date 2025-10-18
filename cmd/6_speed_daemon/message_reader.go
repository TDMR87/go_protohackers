package main

import (
	"fmt"
	"io"
)

type MessageReader struct {
	reader io.Reader
	buf    []byte
}

func NewMessageReader(r io.Reader) *MessageReader {
	reader := MessageReader{
		buf:    make([]byte, 0, 4096),
		reader: r,
	}

	return &reader
}

// NextMessage reads and returns the next message from the underlying reader (e.g. net.Conn).
func (reader *MessageReader) NextMessage() (msg any, err error) {
	for {
		if len(reader.buf) < 1 {
			err = reader.fillBuffer()
			if err != nil {
				return nil, err
			}
		}

		var msg any
		var needMoreBytes bool
		msgType := reader.buf[0]

		switch msgType {
			case Plate{}.Type():
				strSize := int(reader.buf[1])
				msgSizeInBytes := 2 + strSize + 4 // Type(1 byte) + Length(1 byte) + Plate length (bytes) + Timestamp(4 bytes)
				msg, needMoreBytes, err = extractMessage(reader, msgSizeInBytes, Plate{}.Decode)
			case WantHeartBeat{}.Type():
				msg, needMoreBytes, err = extractMessage(reader, WantHeartBeat{}.Size(), WantHeartBeat{}.Decode)
			case IAmCamera{}.Type():
				msg, needMoreBytes, err = extractMessage(reader, IAmCamera{}.Size(), IAmCamera{}.Decode)
			default:
				return nil, fmt.Errorf("MessageReader does not support message type %#x", msgType)
		}

		if err != nil {
			return nil, err
		}

		if needMoreBytes {
			reader.fillBuffer()
			continue
		}

		return msg, nil
	}
}

// extractMessage extracts messages of various types from the reader's buffer.
func extractMessage[T any](
	reader *MessageReader,
	msgLength int,
	decoderFunc func(b []byte) (T, error),
) (msg T, needMoreBytes bool, err error) {
	var zero T
	if len(reader.buf) < msgLength {
		return zero, true, nil
	}

	// Use the decoder function to extract a message from the buffer
	// Slice the buffer to the required length for the message
	msg, err = decoderFunc(reader.buf[:msgLength])
	if err != nil {
		return zero, false, err
	}

	// Remove the extracted bytes from the reader's buffer
	reader.buf = reader.buf[msgLength:]
	return msg, false, nil
}

// fillBuffer reads more data from the underlying reader into the buffer.
func (f *MessageReader) fillBuffer() error {
	tempBuf := make([]byte, 512)
	n, err := f.reader.Read(tempBuf)
	if err != nil {
		return err
	}
	f.buf = append(f.buf, tempBuf[:n]...)
	return nil
}
