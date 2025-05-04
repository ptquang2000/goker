package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unicode/utf8"
)

type VarByteInt uint32

func (v VarByteInt) encode(x uint) []byte {
	encodedByte := uint32(0)
	for {
		encodedByte = uint32(x % 128)
		x /= 128
		if x > 0 {
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
			return 0, errors.New("Invalid Variable Byte Integer")
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

func (s *UTF8String) decode(b []byte) (int, error) {
	*s = ""
	if len(b) < 2 {
		return 0, errors.New("Invalid UTF-8 string.")
	}

	slen := binary.BigEndian.Uint16(b[:2])
	if slen == 0 {
		return 2, nil
	}
	b = b[2:]

	if len(b) < int(slen) {
		return 0, errors.New("Invalid UTF-8 string.")
	}
	b = b[:slen]

	if !utf8.Valid(b) {
		return 0, errors.New("Invalid UTF-8 string.")
	}

	n := 0
	for n < len(b) {
		r, size := utf8.DecodeRune(b[n:])
		if r == utf8.RuneError {
			continue
		}
		*s += UTF8String(r)
		n += size
	}
	return 2 + n, nil
}

type UTF8StringPair []string

func (sp *UTF8StringPair) decode(b []byte) (int, error) {
	*sp = []string{}
	if len(b) < 4 {
		return 0, errors.New("Invalid UTF-8 string pair.")
	}

	rBytes := 0
	for len(*sp) <= 2 && len(b[rBytes:]) >= 2 {
		var s UTF8String
		n, err := s.decode(b)
		if err != nil {
			return 0, err
		}

		*sp = append(*sp, string(s))
		rBytes += n
	}

	if len(*sp) != 2 {
		return 0, errors.New("Invalid UTF-8 string pair.")
	}
	return rBytes, nil
}

type BinaryData []byte

func (bd *BinaryData) decode(b []byte) (int, error) {
	if len(b) < 2 {
		return 0, errors.New("Invalid BinaryData.")
	}
	bdLen := binary.BigEndian.Uint16(b[:2])
	*bd = bytes.Clone(b[2 : 2+bdLen])
	return int(bdLen) + 2, nil
}
