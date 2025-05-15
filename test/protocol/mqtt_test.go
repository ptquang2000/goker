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
	recv, err := packets.ReadPacket(&buf)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	ack, ok := recv.Content.(*packets.Connack)
	if recv.Type != packets.CONNACK || !ok {
		t.Error("Expected CONNACK got", recv.PacketType())
		t.FailNow()
	}
	testConnackProp(ack, t)

	buf.Reset()
	buf.Write([]byte{16, 38, 0, 4, 77, 81, 84, 84, 5, 128, 0, 30, 5, 17, 0, 0, 0, 30, 0, 10, 116, 101, 115, 116, 67, 108, 105, 101, 110, 116, 0, 8, 116, 101, 115, 116, 85, 115, 101, 114})
	req, err = parsePacket(buf.Bytes())
	if err != nil {
		t.Error(err)
	}

	buf.Reset()
	req.WriteTo(&buf)
	recv, err = packets.ReadPacket(&buf)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	ack, ok = recv.Content.(*packets.Connack)
	if recv.Type != packets.CONNACK || !ok {
		t.Error("Expected CONNACK got", recv.PacketType())
		t.FailNow()
	}
	testConnackProp(ack, t)
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

func testConnackProp(pkt *packets.Connack, t *testing.T) {
	if *pkt.Properties.RetainAvailable == 1 {
		t.Error("Expected retain should be unvailable")
	}
	if *pkt.Properties.SubIDAvailable == 1 {
		t.Error("Expected subscription identifiers should be unvailable")
	}
	if *pkt.Properties.WildcardSubAvailable == 1 {
		t.Error("Expected wildcard subscriptions should be unvailable")
	}
	if *pkt.Properties.SharedSubAvailable == 1 {
		t.Error("Expected shared subscriptions should be unvailable")
	}
}
