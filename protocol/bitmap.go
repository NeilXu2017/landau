package protocol

import (
	"strings"
)

const (
	zero = byte('0')
	one  = byte('1')
)

var (
	uint8arr [8]uint8
)

func init() {
	uint8arr[0] = 128
	uint8arr[1] = 64
	uint8arr[2] = 32
	uint8arr[3] = 16
	uint8arr[4] = 8
	uint8arr[5] = 4
	uint8arr[6] = 2
	uint8arr[7] = 1
}

//ConvertBimapByte2String 权限位Byte转成Sting
func ConvertBimapByte2String(bitmaps []byte) string {
	l := len(bitmaps)
	bl := l * 8
	buf := make([]byte, 0, bl)
	for _, b := range bitmaps {
		buf = appendBinaryString(buf, b)
	}
	return string(buf)
}

//ConvertBitmapString2Byte 权限位Sting转成Byte
func ConvertBitmapString2Byte(bitmaps string) []byte {
	l := len(bitmaps)
	if l == 0 {
		return []byte{}
	}
	mo := l % 8
	if mo != 0 {
		bitmaps += strings.Repeat("0", 8-mo)
	}
	l = len(bitmaps)
	l /= 8
	bs := make([]byte, 0, l)
	var n uint8
	for i, b := range []byte(bitmaps) {
		m := (i + 8) % 8
		if b == one {
			n += uint8arr[m]
		}
		if m == 7 {
			bs = append(bs, n)
			n = 0
		}
	}
	return bs
}

func appendBinaryString(bs []byte, b byte) []byte {
	var a byte
	for i := 0; i < 8; i++ {
		a = b
		b <<= 1
		b >>= 1
		switch a {
		case b:
			bs = append(bs, zero)
		default:
			bs = append(bs, one)
		}
		b <<= 1
	}
	return bs
}
