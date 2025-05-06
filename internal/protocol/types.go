package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unicode/utf8"
)

type VarByteInt uint32

func (v VarByteInt) encode() []byte {
	encodedByte := uint32(0)
	for {
		encodedByte = uint32(v % 128)
		v /= 128
		if v > 0 {
			encodedByte |= 128
		} else {
			break
		}
	}
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodedByte)
	return bytes.TrimRight(b, "\x00")
}

func (v *VarByteInt) decode(b []byte) (int, error) {
	multiplier := uint32(1)
	x := uint32(0)
	n := 0
	var encodedByte uint32
	for {
		if multiplier > 128*128*128 || n >= len(b) {
			return 0, errors.New("Unable to decode Variable Byte Integer")
		}
		encodedByte = uint32(b[n])
		x += (encodedByte & 127) * multiplier

		multiplier *= 128
		n += 1
		if (encodedByte & 128) == 0 {
			break
		}
	}
	*v = VarByteInt(x)
	return n, nil
}

type UTF8String string

func (v *UTF8String) decode(b []byte) (int, error) {
	*v = ""
	if len(b) < 2 {
		return 0, errors.New("Unable to decode UTF-8 string.")
	}

	slen := binary.BigEndian.Uint16(b[:2])
	if slen == 0 {
		return 2, nil
	}
	b = b[2:]

	if len(b) < int(slen) {
		return 0, errors.New("Unable to decode UTF-8 string.")
	}
	b = b[:slen]

	if !utf8.Valid(b) {
		return 0, errors.New("Unable to decode UTF-8 string.")
	}

	n := 0
	for n < len(b) {
		r, size := utf8.DecodeRune(b[n:])
		if r == utf8.RuneError {
			continue
		}
		*v += UTF8String(r)
		n += size
	}
	return 2 + n, nil
}

type UTF8StringPair []string

func (v *UTF8StringPair) decode(b []byte) (int, error) {
	*v = []string{}
	if len(b) < 4 {
		return 0, errors.New("Unable to decode UTF-8 string pair.")
	}

	rBytes := 0
	for len(*v) <= 2 && len(b[rBytes:]) >= 2 {
		var s UTF8String
		n, err := s.decode(b)
		if err != nil {
			return 0, err
		}

		*v = append(*v, string(s))
		rBytes += n
	}

	if len(*v) != 2 {
		return 0, errors.New("Unable to decode UTF-8 string pair.")
	}
	return rBytes, nil
}

type BinaryData []byte

func (v *BinaryData) decode(b []byte) (int, error) {
	if len(b) < 2 {
		return 0, errors.New("Unable to decode BinaryData.")
	}
	vLen := binary.BigEndian.Uint16(b[:2])
	*v = bytes.Clone(b[2 : 2+vLen])
	return int(vLen) + 2, nil
}

type ByteInteger bool

func (v *ByteInteger) decode(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, errors.New("Unable to decode Byte Integer.")
	}
	b = b[:1]
	if b[0] > 1 {
		return 0, errors.New("Invalid Byte Integer value.")
	}
	*v = ByteInteger(b[0] == 1)
	return len(b), nil
}

type TwoByteInteger uint16

func (v *TwoByteInteger) decode(b []byte) (int, error) {
	if len(b) < 2 {
		return 0, errors.New("Unable to decode Two Byte Integer.")
	}
	b = b[:2]
	*v = TwoByteInteger(binary.BigEndian.Uint16(b))
	return len(b), nil
}

type FourByteInteger uint32

func (v *FourByteInteger) decode(b []byte) (int, error) {
	if len(b) < 4 {
		return 0, errors.New("Unable to decode Two Byte Integer.")
	}
	b = b[:4]
	*v = FourByteInteger(binary.BigEndian.Uint32(b))
	return len(b), nil
}
