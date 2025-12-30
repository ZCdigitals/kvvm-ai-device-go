package socket

import "encoding/binary"

// socket header
//
// every socket message will send this header first
//
// size is body length, it cloud be 0
//
// reserved cloud be used for static data, like format or type
type SocketHeader struct {
	ID        uint32
	Size      uint32
	Timestamp uint64
	Reserved  [8]uint32
}

// 48
const SocketHeaderLength int = 4 + 4 + 8 + 4*8

// parse socket header from binary data
//
// this function do NOT check if input is enough
func ParseSocketHeader(b []byte) SocketHeader {
	if len(b) < SocketHeaderLength {
		return SocketHeader{}
	}

	reserved := [8]uint32{}

	offset := 16
	for i := range len(reserved) {
		reserved[i] = binary.LittleEndian.Uint32(b[offset : offset+4])
		offset += 4
	}

	return SocketHeader{
		ID:        binary.LittleEndian.Uint32(b[0:4]),
		Size:      binary.LittleEndian.Uint32(b[4:8]),
		Timestamp: binary.LittleEndian.Uint64(b[8:16]),
		Reserved:  reserved,
	}
}
