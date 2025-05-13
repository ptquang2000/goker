package protocol

import "io"

type Request interface {
	WriteTo(io.Writer) (int64, error)
}
type RequestHeader interface {
	Parse([]byte) (Request, error)
	BodyLength() int
}
