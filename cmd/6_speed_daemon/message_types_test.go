package main

import (
	"encoding/binary"
	"testing"
)

func TestErrorEncode(t *testing.T) {
	bytes, _ := Error{"This is a message"}.Encode()

	if len(bytes) != 19 {
		t.Fatalf("Expected a total of %d bytes, got %d bytes", 19, len(bytes))
	}
	if bytes[0] != (Error{}).Type() {
		t.Fatalf("Expected message type to be hex %x, got hex %x", Error{}.Type(), bytes[0])
	}
	if bytes[1] != 17 {
		t.Fatalf("Expected payload length to be 17, got %d", bytes[1])
	}
	if len(bytes[2:]) != 17 {
		t.Fatalf("Expected payload to have 17 bytes, got %d", len(bytes[2:]))
	}
	if string(bytes[2:]) != "This is a message" {
		t.Fatalf("Expected 'This is a message', got %s", string(bytes[2:]))
	}

	errorMsg := Error{"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc faucibus vel velit sed congue. Donec tincidunt sed libero pretium dapibus. Sed commodo nunc at varius congue. Nullam a felis non dui interdum hendrerit. Nunc non nisi nisl. Sed rhoncus lobortis efficitur."}
	_, err := errorMsg.Encode()
	if err == nil {
		t.Fatalf("Expected error, got none")
	}
	if err.Error() != "error message cannot exceed 255 bytes" {
		t.Fatalf("Got incorrect error message, got '%s'", err.Error())
	}
}

func TestPlateEncode(t *testing.T) {
	bytes, _ := Plate{Plate: "UN1X", Timestamp: 123456}.Encode()

	if len(bytes) != 9 {
		t.Fatalf("Expected a total of %d bytes, got %d bytes", 19, len(bytes))
	}
	if bytes[0] != (Plate{}).Type() {
		t.Fatalf("Expected message type to be hex %x, got hex %x", Plate{}.Type(), bytes[0])
	}
	if len(bytes[1:]) != 8 {
		t.Fatalf("Expected payload to have 8 bytes, got %d", len(bytes[1:]))
	}
	if string(bytes[1:5]) != "UN1X" {
		t.Fatalf("Expected bytes[1:4] to be 'UN1X', got %s", string(bytes[1:4]))
	}

	timestamp := int(binary.BigEndian.Uint32(bytes[5:]))
	if timestamp != 123456 {
		t.Fatalf("Expected bytes[5:] to be '123456', got %d", timestamp)
	}

	errorMsg := Plate{Plate: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc faucibus vel velit sed congue. Donec tincidunt sed libero pretium dapibus. Sed commodo nunc at varius congue. Nullam a felis non dui interdum hendrerit. Nunc non nisi nisl. Sed rhoncus lobortis efficitur."}
	_, err := errorMsg.Encode()
	if err == nil {
		t.Fatalf("Expected error, got none")
	}
	if err.Error() != "plate cannot exceed 255 bytes" {
		t.Fatalf("Got incorrect error message, got '%s'", err.Error())
	}
}

func TestTicketEncode(t *testing.T) {
	bytes, _ := Ticket{
		Plate:      "UN1X",
		Road:       66,
		Mile1:      100,
		Mile2:      200,
		Speed:      88,
		Timestamp1: 123456,
		Timestamp2: 654321,
	}.Encode()

	expectedLen := 21 // 1 (msg type) + len(plate) + 2+2+4+2+4+2 = 21
	if len(bytes) != expectedLen {
		t.Fatalf("Expected %d bytes, got %d", expectedLen, len(bytes))
	}
	if bytes[0] != (Ticket{}).Type() {
		t.Fatalf("Expected message type %x, got %x", Ticket{}.Type(), bytes[0])
	}
	if string(bytes[1:5]) != "UN1X" { // Plate
		t.Fatalf("Expected plate 'UN1X', got '%s'", string(bytes[1:5]))
	}
	road := binary.BigEndian.Uint16(bytes[5:7])
	if road != 66 {
		t.Fatalf("Expected road 66, got %d", road)
	}
	mile1 := binary.BigEndian.Uint16(bytes[7:9])
	if mile1 != 100 {
		t.Fatalf("Expected mile1 100, got %d", mile1)
	}
	timestamp1 := binary.BigEndian.Uint32(bytes[9:13])
	if timestamp1 != 123456 {
		t.Fatalf("Expected timestamp1 123456, got %d", timestamp1)
	}
	mile2 := binary.BigEndian.Uint16(bytes[13:15])
	if mile2 != 200 {
		t.Fatalf("Expected mile2 200, got %d", mile2)
	}
	timestamp2 := binary.BigEndian.Uint32(bytes[15:19])
	if timestamp2 != 654321 {
		t.Fatalf("Expected timestamp2 654321, got %d", timestamp2)
	}
	speed := binary.BigEndian.Uint16(bytes[19:21])
	if speed != 88 {
		t.Fatalf("Expected speed 88, got %d", speed)
	}
	longPlate := make([]byte, 256)
	for i := range longPlate {
		longPlate[i] = 'A'
	}

	badTicket := Ticket{Plate: string(longPlate)}
	_, err := badTicket.Encode()

	if err == nil {
		t.Fatalf("Expected error for plate exceeding 255 bytes, got none")
	}
	if err.Error() != "plate cannot exceed 255 bytes" {
		t.Fatalf("Expected error message 'plate cannot exceed 255 bytes', got '%s'", err.Error())
	}
}

func TestWantHeartBeatEncode(t *testing.T) {
	bytes := WantHeartBeat{Interval: 30}.Encode()

	expectedLen := 5 // 1 (msg type) + 4 (interval) = 5
	if len(bytes) != expectedLen {
		t.Fatalf("Expected %d bytes, got %d", expectedLen, len(bytes))
	}
	if bytes[0] != (WantHeartBeat{}).Type() {
		t.Fatalf("Expected message type %x, got %x", WantHeartBeat{}.Type(), bytes[0])
	}

	interval := binary.BigEndian.Uint32(bytes[1:5])
	if interval != 30 {
		t.Fatalf("Expected interval 30, got %d", interval)
	}
}

func TestHeartBeatEncode(t *testing.T) {
	bytes := HeartBeat{}.Encode()

	if len(bytes) != 1 { // 1 (just the message type)
		t.Fatalf("Expected 1 byte, got %d", len(bytes))
	}
	if bytes[0] != (HeartBeat{}).Type() {
		t.Fatalf("Expected message type %x, got %x", HeartBeat{}.Type(), bytes[0])
	}
}

func TestIAmCameraEncode(t *testing.T) {
	bytes := IAmCamera{
		Road:  66,
		Mile:  1234,
		Limit: 55,
	}.Encode()

	expectedLen := 7 // 1 (msg type) + 2 + 2 + 2 = 7
	if len(bytes) != expectedLen {
		t.Fatalf("Expected %d bytes, got %d", expectedLen, len(bytes))
	}
	if bytes[0] != (IAmCamera{}).Type() {
		t.Fatalf("Expected message type %x, got %x", IAmCamera{}.Type(), bytes[0])
	}

	road := binary.BigEndian.Uint16(bytes[1:3])
	if road != 66 {
		t.Fatalf("Expected road 66, got %d", road)
	}

	mile := binary.BigEndian.Uint16(bytes[3:5])
	if mile != 1234 {
		t.Fatalf("Expected mile 1234, got %d", mile)
	}

	limit := binary.BigEndian.Uint16(bytes[5:7])
	if limit != 55 {
		t.Fatalf("Expected limit 55, got %d", limit)
	}
}

func TestIAmDispatcherEncode(t *testing.T) {
	tests := map[string]IAmDispatcher{
		"single road": {
			Numroads: 1,
			Roads:    []uint16{42},
		},
		"two roads": {
			Numroads: 2,
			Roads:    []uint16{66, 77},
		},
		"three roads": {
			Numroads: 3,
			Roads:    []uint16{100, 200, 300},
		},
		"four roads small values": {
			Numroads: 4,
			Roads:    []uint16{1, 2, 3, 4},
		},
		"five roads with max uint16": {
			Numroads: 5,
			Roads:    []uint16{0, 123, 9999, 65535, 500},
		},
	}

	for name, dispatcher := range tests {
		bytes := dispatcher.Encode()

		expectedLen := 1 + 1 + 2*len(dispatcher.Roads)
		if len(bytes) != expectedLen {
			t.Fatalf("%s: Expected %d bytes, got %d", name, expectedLen, len(bytes))
		}

		if bytes[0] != (IAmDispatcher{}).Type() {
			t.Fatalf("%s: Expected message type %x, got %x", name, IAmDispatcher{}.Type(), bytes[0])
		}

		if bytes[1] != dispatcher.Numroads {
			t.Fatalf("%s: Expected numroads %d, got %d", name, dispatcher.Numroads, bytes[1])
		}

		for i, road := range dispatcher.Roads {
			offset := 2 + i*2
			gotRoad := binary.BigEndian.Uint16(bytes[offset : offset+2])
			if gotRoad != road {
				t.Fatalf("%s: At index %d, expected road %d, got %d", name, i, road, gotRoad)
			}
		}
	}
}

func TestDecodeError(t *testing.T) {
	tests := map[string]struct {
		data    []byte
		wantMsg string
		wantErr bool
	}{
		"valid error": {
			data:    []byte{Error{}.Type(), 5, 'H', 'e', 'l', 'l', 'o'},
			wantMsg: "Hello",
		},
		"wrong type": {
			data:    []byte{0x99, 5, 'H', 'e', 'l', 'l', 'o'},
			wantErr: true,
		},
		"short payload": {
			data:    []byte{Error{}.Type(), 5, 'H', 'e'},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		got, err := Error{}.Decode(tt.data)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error, got nil", name)
			}
			return
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if got.Msg != tt.wantMsg {
			t.Fatalf("%s: expected '%s', got '%s'", name, tt.wantMsg, got.Msg)
		}
	}
}

func TestDecodePlate(t *testing.T) {
	timestamp := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp, 123456)

	for name, tt := range map[string]struct {
		data    []byte
		want    Plate
		wantErr bool
	}{
		"happy path": {
			data: append(append([]byte{Plate{}.Type(), 4}, []byte("UN1X")...), timestamp...),
			want: Plate{Plate: "UN1X", Timestamp: 123456},
		},
		"wrong type": {
			data:    append([]byte{0x99, 0x00}, timestamp...),
			wantErr: true,
		},
		"too short overall": {
			data:    []byte{Plate{}.Type()},
			wantErr: true,
		},
		"declared length too big": {
			data:    []byte{Plate{}.Type(), 10, 'A', 'B', 'C'},
			wantErr: true,
		},
		"missing bytes in timestamp": {
			data:    append([]byte{Plate{}.Type(), 1, 'A'}, []byte{0x01, 0x02}...),
			wantErr: true,
		},
		"empty plate is valid": {
			data: append([]byte{Plate{}.Type(), 0}, timestamp...),
			want: Plate{Plate: "", Timestamp: 123456},
		},
	} {
		got, err := Plate{}.Decode(tt.data)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("[%s] expected error, got none", name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("[%s] unexpected error: %v", name, err)
		}
		if got != tt.want {
			t.Fatalf("[%s] expected %+v, got %+v", name, tt.want, got)
		}
	}
}

func TestDecodeTicket(t *testing.T) {
	plate := "ABC123"
	plateBytes := []byte(plate)
	road := make([]byte, 2)
	binary.BigEndian.PutUint16(road, 1)
	mile1 := make([]byte, 2)
	binary.BigEndian.PutUint16(mile1, 10)
	timestamp1 := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp1, 1000)
	mile2 := make([]byte, 2)
	binary.BigEndian.PutUint16(mile2, 20)
	timestamp2 := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp2, 2000)
	speed := make([]byte, 2)
	binary.BigEndian.PutUint16(speed, 60)

	data := append([]byte{Ticket{}.Type()}, plateBytes...)
	data = append(data, road...)
	data = append(data, mile1...)
	data = append(data, timestamp1...)
	data = append(data, mile2...)
	data = append(data, timestamp2...)
	data = append(data, speed...)

	got, err := DecodeTicket(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Plate != plate {
		t.Fatalf("expected plate %s, got %s", plate, got.Plate)
	}
	if got.Road != 1 || got.Mile1 != 10 || got.Timestamp1 != 1000 ||
		got.Mile2 != 20 || got.Timestamp2 != 2000 || got.Speed != 60 {
		t.Fatalf("ticket fields incorrect: %+v", got)
	}
}

func TestDecodeWantHeartBeat(t *testing.T) {
	data := make([]byte, 5)
	data[0] = (WantHeartBeat{}).Type()
	binary.BigEndian.PutUint32(data[1:], 30)

	got, err := WantHeartBeat{}.Decode(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Interval != 30 {
		t.Fatalf("expected interval 30, got %d", got.Interval)
	}
}

func TestDecodeHeartBeat(t *testing.T) {
	data := []byte{HeartBeat{}.Type()}
	_, err := HeartBeat{}.Decode(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeIAmCamera(t *testing.T) {
	data := make([]byte, 7)
	data[0] = IAmCamera{}.Type()
	binary.BigEndian.PutUint16(data[1:3], 1)
	binary.BigEndian.PutUint16(data[3:5], 100)
	binary.BigEndian.PutUint16(data[5:7], 80)

	got, err := DecodeIAmCamera(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Road != 1 || got.Mile != 100 || got.Limit != 80 {
		t.Fatalf("fields incorrect: %+v", got)
	}
}

func TestDecodeIAmDispatcher(t *testing.T) {
	data := []byte{IAmDispatcher{}.Type(), 3}
	roads := []uint16{10, 20, 30}
	for _, r := range roads {
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, r)
		data = append(data, b...)
	}

	got, err := DecodeIAmDispatcher(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Numroads != 3 {
		t.Fatalf("expected numroads 3, got %d", got.Numroads)
	}
	for i, r := range roads {
		if got.Roads[i] != r {
			t.Fatalf("expected road %d at index %d, got %d", r, i, got.Roads[i])
		}
	}
}
