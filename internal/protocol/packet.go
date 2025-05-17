package protocol

import (
	"bytes"
	"io"
)

type Request interface {
	ResponseTo(io.Writer) (int64, error)
}
type RequestHeader interface {
	ParseBody(*bytes.Buffer) (Request, error)
	BodyLength() int
}
