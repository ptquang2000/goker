package test

import (
	"bufio"
	"bytes"
	"goker/internal/protocol"
	"testing"

	"github.com/eclipse/paho.golang/paho"
)

func TestConnectPacket(t *testing.T) {
	protocol.ParseHeader([]byte{})

	cp := &paho.Connect{
		KeepAlive:    30,
		ClientID:     "testClient",
		UsernameFlag: true,
		Username:     "testUser",
		Properties:   &paho.ConnectProperties{SessionExpiryInterval: paho.Uint32(30)},
	}
	cpp := cp.Packet()
	cpp.ProtocolName = "MQTT"
	cpp.ProtocolVersion = 5

	var ws bytes.Buffer
	w := bufio.NewWriter(&ws)
	cpp.WriteTo(w)
	w.Flush()

	b := ws.Bytes()
	if err := parsePacket(b); err != nil {
		t.Error(err)
	}

	b = []byte{16, 38, 0, 4, 77, 81, 84, 84, 5, 128, 0, 30, 5, 17, 0, 0, 0, 30, 0, 10, 116, 101, 115, 116, 67, 108, 105, 101, 110, 116, 0, 8, 116, 101, 115, 116, 85, 115, 101, 114}
	if err := parsePacket(b); err != nil {
		t.Error(err)
	}
}

func parsePacket(b []byte) error {
	p, err := protocol.ParseHeader(b[:2])
	if err != nil {
		return err
	}

	_, err = p.Parse(b[2:])
	if err != nil {
		return err
	}

	return nil
}
