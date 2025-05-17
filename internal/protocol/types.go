package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unicode/utf8"
)

type VarByteInt uint32

func (v *VarByteInt) Add(n int) {
	*v += VarByteInt(n)
}

func (v VarByteInt) encode() *bytes.Buffer {
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
	b = bytes.Trim(b, "\x00")

	return bytes.NewBuffer(b)
}

func (v *VarByteInt) decode(r *bytes.Buffer) error {
	multiplier := uint32(1)
	x := uint32(0)
	var encodedByte uint32
	for {
		b, err := r.ReadByte()
		if multiplier > 128*128*128 || err != nil {
			return errors.New("Unable to decode Variable Byte Integer")
		}
		encodedByte = uint32(b)
		x += (encodedByte & 127) * multiplier

		multiplier *= 128
		if (encodedByte & 128) == 0 {
			break
		}
	}
	*v = VarByteInt(x)
	return nil
}

type UTF8String string

func (v *UTF8String) decode(r *bytes.Buffer) error {
	*v = ""
	b := make([]byte, 2)
	if _, err := r.Read(b); err != nil {
		return errors.New("Unable to decode UTF-8 string.")
	}
	slen := binary.BigEndian.Uint16(b)
	if slen == 0 {
		return nil
	}

	if r.Len() < int(slen) {
		return errors.New("UTF-8 string doesn't match set length.")
	} else if !utf8.Valid(r.Bytes()) {
		return errors.New("UTF-8 string is not valid utf-8.")
	}

	remain := r.Len()
	for remain-r.Len() < int(slen) {
		rune, size := utf8.DecodeRune(r.Bytes())
		if rune == utf8.RuneError {
			continue
		}
		*v += UTF8String(rune)
		r.Next(size)
	}
	return nil
}

type UTF8StringPair struct {
	key   UTF8String
	value UTF8String
}

func (v *UTF8StringPair) decode(r *bytes.Buffer) error {
	if err := v.key.decode(r); err != nil {
		return errors.New("Unable to decode key in UTF-8 string pair, err:" + err.Error())
	}
	if err := v.value.decode(r); err != nil {
		return errors.New("Unable to decode value in UTF-8 string pair, err:" + err.Error())
	}
	return nil
}

type BinaryData []byte

func (v *BinaryData) decode(r *bytes.Buffer) error {
	b := make([]byte, 2)
	if _, err := r.Read(b); err != nil {
		return errors.New("Unable to decode BinaryData.")
	}
	vLen := binary.BigEndian.Uint16(b)
	*v = make([]byte, vLen)
	if _, err := r.Read(*v); err != nil {
		return errors.New("Unable to decode BinaryData.")
	}
	return nil
}

type ByteInteger bool

func (v ByteInteger) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	if v {
		w.WriteByte(0b1)
	} else {
		w.WriteByte(0b0)
	}
	return w
}

func (v *ByteInteger) decode(r *bytes.Buffer) error {
	b := make([]byte, 1)
	if _, err := r.Read(b); err != nil || b[0] >= 1 {
		return errors.New("Unable to decode Byte Integer.")
	}
	*v = ByteInteger(b[0] == 1)
	return nil
}

type TwoByteInteger uint16

func (v *TwoByteInteger) decode(r *bytes.Buffer) error {
	b := make([]byte, 2)
	if _, err := r.Read(b); err != nil {
		return errors.New("Unable to decode Two Byte Integer.")
	}
	*v = TwoByteInteger(binary.BigEndian.Uint16(b))
	return nil
}

type FourByteInteger uint32

func (v *FourByteInteger) decode(r *bytes.Buffer) error {
	b := make([]byte, 4)
	if _, err := r.Read(b); err != nil {
		return errors.New("Unable to decode Two Byte Integer.")
	}
	*v = FourByteInteger(binary.BigEndian.Uint32(b))
	return nil
}
