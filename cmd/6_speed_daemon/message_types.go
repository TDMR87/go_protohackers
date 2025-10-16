package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Message interface {
	Type() byte
	Size() int
}

func (h WantHeartBeat) Type() byte    { return 0x40 }
func (h WantHeartBeat) Size() int     { return 5 }
func (e Error) Type() byte            { return 0x10 }
func (e Error) Size() int             { return 1 + 1 + 255 }
func (p Plate) Type() byte            { return 0x20 }
func (p Plate) Size() int             { return 2 + len(p.Plate) + 4 }
func (t Ticket) Type() byte           { return 0x21 }
func (t Ticket) Size() int            { return 1 + len(t.Plate) + 2 + 2 + 4 + 2 + 4 + 2 }
func (h HeartBeat) Type() byte        { return 0x41 }
func (h HeartBeat) Size() int         { return 1 }
func (cam IAmCamera) Type() byte      { return 0x80 }
func (cam IAmCamera) Size() int       { return 1 + 2 + 2 + 2 }
func (disp IAmDispatcher) Type() byte { return 0x81 }
func (disp IAmDispatcher) Size() int  { return 2 + len(disp.Roads)*2 }

type Error struct {
	Msg string
}

type Plate struct {
	Plate     string
	Timestamp uint32
}

type WantHeartBeat struct {
	Interval uint32
}

type Ticket struct {
	Plate      string
	Road       uint16
	Mile1      uint16
	Timestamp1 uint32
	Mile2      uint16
	Timestamp2 uint32
	Speed      uint16
}

type HeartBeat struct{}

type IAmCamera struct {
	Road  uint16
	Mile  uint16
	Limit uint16
}

type IAmDispatcher struct {
	Numroads uint8
	Roads    []uint16
}

func (e Error) Encode() (bytes []byte, err error) {
	if len(e.Msg) > (Error{}).Size() {
		return nil, errors.New("error message cannot exceed 255 bytes")
	}
	payload := []byte(e.Msg)
	result := []byte{byte(Error{}.Type()), byte(len(payload))}
	result = append(result, payload...)
	return result, nil
}

func (p Plate) Encode() (bytes []byte, err error) {
	if len(p.Plate) > (Plate{}).Size() {
		return nil, errors.New("plate cannot exceed 255 bytes")
	}
	messageType := Plate{}.Type()
	plate := []byte(p.Plate)
	timestamp := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp, p.Timestamp)
	result := []byte{messageType}
	result = append(result, plate...)
	result = append(result, timestamp...)
	return result, nil
}

func (p Ticket) Encode() (bytes []byte, err error) {
	if len(p.Plate) > 255 {
		return nil, errors.New("plate cannot exceed 255 bytes")
	}

	plate := []byte(p.Plate)
	road := make([]byte, 2)
	binary.BigEndian.PutUint16(road, p.Road)
	mile1 := make([]byte, 2)
	binary.BigEndian.PutUint16(mile1, p.Mile1)
	mile2 := make([]byte, 2)
	binary.BigEndian.PutUint16(mile2, p.Mile2)
	speed := make([]byte, 2)
	binary.BigEndian.PutUint16(speed, p.Speed)
	timestamp1 := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp1, p.Timestamp1)
	timestamp2 := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp2, p.Timestamp2)

	result := []byte{Ticket{}.Type()}
	result = append(result, plate...)
	result = append(result, road...)
	result = append(result, mile1...)
	result = append(result, timestamp1...)
	result = append(result, mile2...)
	result = append(result, timestamp2...)
	result = append(result, speed...)
	return result, nil
}

func (h WantHeartBeat) Encode() []byte {
	interval := make([]byte, WantHeartBeat{}.Size()-1)
	binary.BigEndian.PutUint32(interval, h.Interval)
	result := []byte{WantHeartBeat{}.Type()}
	result = append(result, interval...)
	return result
}

func (h HeartBeat) Encode() []byte {
	return []byte{HeartBeat{}.Type()}
}

func (cam IAmCamera) Encode() []byte {
	road := make([]byte, 2)
	binary.BigEndian.PutUint16(road, cam.Road)
	mile := make([]byte, 2)
	binary.BigEndian.PutUint16(mile, cam.Mile)
	limit := make([]byte, 2)
	binary.BigEndian.PutUint16(limit, cam.Limit)

	result := []byte{IAmCamera{}.Type()}
	result = append(result, road...)
	result = append(result, mile...)
	result = append(result, limit...)
	return result
}

func (disp IAmDispatcher) Encode() []byte {
	numroads := byte(disp.Numroads)
	roads := make([]byte, len(disp.Roads)*2)
	for i, road := range disp.Roads {
		binary.BigEndian.PutUint16(roads[i*2:], road)
	}
	result := []byte{IAmDispatcher{}.Type(), numroads}
	result = append(result, roads...)
	return result
}

func (WantHeartBeat) Decode(data []byte) (WantHeartBeat, error) {
	if len(data) != (WantHeartBeat{}).Size() || data[0] != (WantHeartBeat{}).Type() {
		return WantHeartBeat{}, errors.New("invalid WantHeartBeat message")
	}
	interval := binary.BigEndian.Uint32(data[1:])
	return WantHeartBeat{Interval: interval}, nil
}

func (Error) Decode(data []byte) (Error, error) {
	if len(data) > (Error{}).Size() {
		return Error{}, fmt.Errorf("the given data (%#x bytes) is too large to be Error", len(data))
	}
	if data[0] != (Error{}).Type() {
		return Error{}, fmt.Errorf("the given payload's type (%#x) is not Error", data[0])
	}
	length := int(data[1])
	if len(data[2:]) != length {
		return Error{}, errors.New("invalid Error payload length")
	}
	return Error{Msg: string(data[2 : 2+length])}, nil
}

func (p Plate) Decode(data []byte) (Plate, error) {
	if len(data) < (Plate{}).Size() || data[0] != (Plate{}).Type() {
		return Plate{}, errors.New("invalid data for a Plate")
	}
	plateLen := int(data[1])
	if len(data) < 2+plateLen+4 {
		return Plate{}, errors.New("not enough data for a Plate")
	}
	plate := string(data[2 : 2+plateLen])
	timestamp := binary.BigEndian.Uint32(data[2+plateLen:])
	return Plate{Plate: plate, Timestamp: timestamp}, nil
}

func DecodeTicket(data []byte) (Ticket, error) {
	if len(data) < (Ticket{}).Size() || data[0] != (Ticket{}).Type() {
		return Ticket{}, errors.New("invalid Ticket message")
	}
	data = data[1:] // skip type byte

	if len(data) < 2+2+4+2+4+2 {
		return Ticket{}, errors.New("data too short for Ticket")
	}

	// Plate is the remaining length minus fixed fields
	plateLen := len(data) - (2 + 2 + 4 + 2 + 4 + 2)
	if plateLen < 0 {
		return Ticket{}, errors.New("invalid Ticket length")
	}

	plate := string(data[:plateLen])
	offset := plateLen

	road := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	mile1 := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	timestamp1 := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	mile2 := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	timestamp2 := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	speed := binary.BigEndian.Uint16(data[offset : offset+2])

	return Ticket{
		Plate:      plate,
		Road:       road,
		Mile1:      mile1,
		Timestamp1: timestamp1,
		Mile2:      mile2,
		Timestamp2: timestamp2,
		Speed:      speed,
	}, nil
}

func (HeartBeat) Decode(data []byte) (HeartBeat, error) {
	if len(data) != (HeartBeat{}).Size() || data[0] != (HeartBeat{}).Type() {
		return HeartBeat{}, errors.New("invalid HeartBeat message")
	}
	return HeartBeat{}, nil
}

func DecodeIAmCamera(data []byte) (IAmCamera, error) {
	if len(data) != (IAmCamera{}).Size() || data[0] != (IAmCamera{}).Type() {
		return IAmCamera{}, errors.New("invalid IAmCamera message")
	}
	road := binary.BigEndian.Uint16(data[1:3])
	mile := binary.BigEndian.Uint16(data[3:5])
	limit := binary.BigEndian.Uint16(data[5:7])
	return IAmCamera{Road: road, Mile: mile, Limit: limit}, nil
}

func DecodeIAmDispatcher(data []byte) (IAmDispatcher, error) {
	if len(data) < (IAmDispatcher{}).Size() || data[0] != (IAmDispatcher{}).Type() {
		return IAmDispatcher{}, errors.New("invalid IAmDispatcher message")
	}
	numroads := data[1]
	if len(data) != int(2+numroads*2) {
		return IAmDispatcher{}, errors.New("invalid IAmDispatcher length")
	}
	roads := make([]uint16, numroads)
	for i := 0; i < int(numroads); i++ {
		roads[i] = binary.BigEndian.Uint16(data[2+i*2 : 2+i*2+2])
	}
	return IAmDispatcher{Numroads: numroads, Roads: roads}, nil
}
