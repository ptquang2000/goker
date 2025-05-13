package test

import (
	"bytes"
	"goker/internal/protocol"
	"testing"

	"github.com/eclipse/paho.golang/packets"
	"github.com/eclipse/paho.golang/paho"
)

func TestConnectPacket(t *testing.T) {
	var buf bytes.Buffer

	_, err := protocol.ParseHeader(buf.Bytes())
	if err == nil {
		t.Error("Missing empty buffer case")
		t.FailNow()
	}

	buf.Reset()
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
	cpp.WriteTo(&buf)
	req, err := parsePacket(buf.Bytes())
	if err != nil {
		t.Error(err)
	}

	buf.Reset()
	req.WriteTo(&buf)
	ack, err := packets.ReadPacket(&buf)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if ack.Type != packets.CONNACK {
		t.Error("Expected CONNACK got", ack.PacketType())
	}

	buf.Reset()
	buf.Write([]byte{16, 38, 0, 4, 77, 81, 84, 84, 5, 128, 0, 30, 5, 17, 0, 0, 0, 30, 0, 10, 116, 101, 115, 116, 67, 108, 105, 101, 110, 116, 0, 8, 116, 101, 115, 116, 85, 115, 101, 114})
	req, err = parsePacket(buf.Bytes())
	if err != nil {
		t.Error(err)
	}

	buf.Reset()
	req.WriteTo(&buf)
	ack, err = packets.ReadPacket(&buf)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if ack.Type != packets.CONNACK {
		t.Error("Expected CONNACK got", ack.PacketType())
	}
}

func parsePacket(b []byte) (protocol.Request, error) {
	p, err := protocol.ParseHeader(b[:2])
	if err != nil {
		return nil, err
	}

	r, err := p.Parse(b[2:])
	if err != nil {
		return nil, err
	}

	return r, nil
}
