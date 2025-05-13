package gateway

import (
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
	h := make([]byte, protocol.FixedHeaderLen)
	defer c.Close()

	for {
		n, err := c.Read(h)
		if n != protocol.FixedHeaderLen || err != nil {
			utils.LogError("Failed to read header, read: ", n, ", err:", err)
			return
		}

		p, err := protocol.ParseHeader(h)
		if err != nil {
			utils.LogError(err != nil, "Failed to parse header, err:", err)
			return
		}

		b := make([]byte, p.BodyLength())
		n, err = c.Read(b)
		if n != p.BodyLength() || err != nil {
			utils.LogError("Failed to read body, read: ", n, ", err:", err)
			return
		}

		req, err := p.Parse(b)
		if err != nil {
			utils.LogError("Close connection with reason, err:", err)
			return
		}

		req.WriteTo(c)
	}
}
