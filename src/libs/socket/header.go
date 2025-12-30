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

// ToBytes converts SocketHeader to byte slice
func (h *SocketHeader) ToBytes() []byte {
	b := make([]byte, SocketHeaderLength)

	binary.LittleEndian.PutUint32(b[0:4], h.ID)
	binary.LittleEndian.PutUint32(b[4:8], h.Size)
	binary.LittleEndian.PutUint64(b[8:16], h.Timestamp)

	offset := 16
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint32(b[offset:offset+4], h.Reserved[i])
		offset += 4
	}

	return b
}
