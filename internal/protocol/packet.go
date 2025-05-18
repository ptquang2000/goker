package protocol

import (
	"bytes"
	"io"
)

type Request interface {
	ResponseTo(io.Writer) (int64, error)
	ToString() string
}
type RequestHeader interface {
	ParseBody(*bytes.Buffer) (Request, error)
	BodyLength() int
}
type PacketProperties struct {
	fields map[MqttProperty]bool
}
