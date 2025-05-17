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
	defer c.Close()

	for {
		var h protocol.MqttHeader

		p, err := protocol.ParseHeader()
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
