package utility

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
)

type BeforeInitHandler func(int)
type BeforeStopHandler func()

var moduleIDSource uint32 = 10

//AllocModuleID allocate one unique module id
func AllocModuleID() uint32 {
	return atomic.AddUint32(&moduleIDSource, 1)
}

func GetTokens(src, sep string) []string {
	return strings.Split(src, sep)
}

func Format(myfmt string, v ...interface{}) string {
	return fmt.Sprintf(myfmt, v...)
}

func StrCmp(src, dest string) int {
	return strings.Compare(src, dest)
}

func atoi(s string, bitSize int) (uint64, error) {
	u, err := strconv.ParseUint(s, 10, bitSize)
	if err != nil {
		return 0, err
	}

	return u, nil
}

func ReadUint32(data []byte) ([]byte, uint32) {
	var size uint32
	for j, i := 0, 3; i >= 0; i, j = i-1, j+1 {
		size |= uint32(data[j]) << (uint)((i * 8))
	}

	return data[4:], size
}

func WriteInt32(dst []byte, size int32) {

	for i, j := 0, 3; i < 4; i, j = i+1, j-1 {
		dst[i] = byte((size >> uint(j*8)) & 0xff)
	}
}

func WriteUint32(dst []byte, size uint32) {
	for i, j := 0, 3; i < 4; i, j = i+1, j-1 {
		dst[i] = byte((size >> uint(j*8)) & 0xff)
	}
}

func ReadUint64(data []byte) ([]byte, uint64) {
	var size uint64
	for j, i := 0, 7; i >= 0; i, j = i-1, j+1 {
		size |= uint64(data[j]) << (uint)((i * 8))
	}
	return data[8:], size
}

func WriteUint64(dst []byte, size uint64) {
	for i, j := 0, 7; i < 8; i, j = i+1, j-1 {
		dst[i] = byte((size >> uint((j * 8))) & 0xff)
	}
}

func WriteUint16(dst []byte, size uint16) []byte {
	dst[0] = byte(size>>8) & 0xff
	dst[1] = byte(size & 0xff)
	return dst[2:]
}

func ReadUint16(data []byte) ([]byte, uint16) {
	size := uint16(data[0]) << 8
	size |= uint16(data[1])
	return data[2:], size
}

func StrToBytes(src string) []byte {
	reader := strings.NewReader(src)
	size := reader.Len()
	data := make([]byte, size)
	reader.Read(data)
	return data
}

//ParseAddr convert string address to byte slice
func ParseAddr(addr string) []byte {
	s := make([]byte, 4)
	b := 0
	for _, c := range addr {
		if c >= '0' && c <= '9' {
			b += b*10 + int(c-'0')
			continue
		}
		s = append(s, byte(b))
	}

	return s
}
