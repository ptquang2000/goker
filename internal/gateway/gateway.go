package gateway

import (
	"bytes"
	"goker/internal/protocol"
	"goker/internal/utils"
	"net"
)

func ListenAndServe() {
	l, err := net.Listen("tcp", ":8883")
	utils.AssertFail(err != nil)
	defer l.Close()

	for {
		c, err := l.Accept()
		utils.AssertFail(err != nil)

		go clientHandle(c)
	}
}

func clientHandle(c net.Conn) {
	defer c.Close()

	var b []byte
	for {
		b = make([]byte, protocol.FixedHeaderLen)
		if _, err := c.Read(b); err != nil {
			utils.LogError("Failed to read header, err:", err)
			return
		}

		h, err := protocol.ParseHeader(bytes.NewBuffer(b))
		if err != nil {
			utils.LogError(err != nil, "Failed to parse header, err:", err)
			return
		}

		b = make([]byte, h.BodyLength())
		if _, err = c.Read(b); err != nil {
			utils.LogError("Failed to read body, err:", err)
			return
		}

		req, err := h.ParseBody(bytes.NewBuffer(b))
		if err != nil {
			utils.LogError("Close connection with reason, err:", err)
			return
		}

		req.ResponseTo(c)
	}
}
