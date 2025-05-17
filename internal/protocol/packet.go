package protocol

import (
	"bytes"
	"io"
)

type Request interface {
	WriteTo(io.Writer) (int64, error)
}
type RequestHeader interface {
	Parse(*bytes.Buffer) (Request, error)
	BodyLength() int
}
