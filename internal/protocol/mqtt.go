package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"time"
)

type FixedHeader [2]byte

const FixedHeaderLen = len(FixedHeader{})

type CType uint8

const (
	RESERVED CType = iota
	CONNECT
	CONNACK
	PUBLISH
	PUBACK
	PUBREC
	PUBREL
	PUBCOMP
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ
	PINGRESP
	DISCONNECT
	AUTH
	COUNT
)

type Flag struct {
	dup    bool
	qos    bool
	retain bool
}
type MqttHeader struct {
	ctl  CType
	flag Flag
	len  VarByteInt
}

func ParseHeader(b []byte) (RequestHeader, error) {
	if len(b) < FixedHeaderLen {
		return nil, errors.New("Invalid header size!")
	}
	h := b[:FixedHeaderLen]

	t := CType(h[0]>>4) & 0b0001111
	if t <= RESERVED || t >= COUNT {
		return nil, errors.New("Unknowned MQTT Control Packet type!")
	}

	f := Flag{dup: bool(h[0]&0b0100 != 0), qos: bool(h[0]&0b0010 != 0), retain: bool(h[0]&0b0001 != 0)}

	var l VarByteInt
	_, err := l.decode(h[1:])
	if err != nil {
		return nil, err
	}

	return &MqttHeader{ctl: t, flag: f, len: l}, nil
}

func (p *MqttHeader) BodyLength() int {
	return int(p.len)
}

func (p *MqttHeader) Parse(b []byte) (Request, error) {
	switch p.ctl {
	case CONNECT:
		return ParseConnect(p, b)
	default:
		return nil, errors.New("Unsupported MQTT packet control")
	}
}

type ConnectFlag byte
type QoS int

const (
	QoS0 QoS = iota
	QoS1
	QoS2
	QoS3
)

func (f ConnectFlag) username() bool {
	return byte(f)&0b10000000 != 0
}
func (f ConnectFlag) password() bool {
	return byte(f)&0b00100000 != 0
}
func (f ConnectFlag) retain() bool {
	return byte(f)&0b00010000 != 0
}
func (f ConnectFlag) qos() QoS {
	return QoS(byte(f) & 0b00011000)
}
func (f ConnectFlag) will() bool {
	return byte(f)&0b00000100 != 0
}
func (f ConnectFlag) cleanstart() bool {
	return byte(f)&0b00000010 != 0
}
func (f ConnectFlag) reservered() bool {
	return byte(f)&0b00000001 != 0
}
func (f ConnectFlag) size() int {
	return 1
}
func (f ConnectFlag) valid() error {
	if f.reservered() {
		return errors.New("Reserved flag must be 0.")
	} else if f.qos() >= QoS3 {
		return errors.New("Invalid QoS")
	} else if f.will() && f.qos() != QoS0 {
		return errors.New("Invalid QoS while will flag is set.")
	}
	return nil
}

type MqttProperty byte

const (
	SessionExpInt MqttProperty = 0x11
	ReceiveMax                 = 0x21
	MaxPacketSize              = 0x27
	TopicAliasMax              = 0x22
	ReqRespInfo                = 0x19
	ReqProbInfo                = 0x17
	UserProp                   = 0x26
	AuthMethod                 = 0x15
	AuthData                   = 0x16
	WillDelayInt               = 0x18
	PlFormatInd                = 0x01
	MsgExpInt                  = 0x02
	ContentType                = 0x03
	RespTopic                  = 0x08
	CorrData                   = 0x09
)

type ConnectProperties struct {
	sessionExpInt time.Duration
	receiveMax    uint16
	maxPacketSize uint32
	topicAliasMax uint16
	reqRespInfo   bool
	reqProbInfo   bool
	userProp      UTF8StringPair
	authMethod    UTF8String
	authData      BinaryData
}

func (p *ConnectProperties) decode(b []byte) (int, error) {
	p.sessionExpInt = 0
	p.receiveMax = math.MaxUint16
	p.maxPacketSize = math.MaxUint32
	p.topicAliasMax = 0
	p.reqRespInfo = false
	p.reqProbInfo = true

	var propLen VarByteInt
	rBytes, err := propLen.decode(b)
	if err != nil {
		return 0, err
	} else if propLen == 0 {
		return rBytes, nil
	}

	b = b[rBytes : rBytes+int(propLen)]
	for len(b) > 0 {
		switch MqttProperty(b[0]) {
		case SessionExpInt:
			if len(b[1:]) < 4 {
				return 0, errors.New("Invalid Session Expiration Interval")
			}
			p.sessionExpInt = time.Duration(binary.BigEndian.Uint32(b[1:5]))
			p.sessionExpInt *= time.Second
			b = b[5:]
		case ReceiveMax:
			if len(b[1:]) < 2 {
				return 0, errors.New("Invalid Receive Maximum")
			}
			p.receiveMax = binary.BigEndian.Uint16(b[1:3])
			b = b[3:]
		case MaxPacketSize:
			if len(b[1:]) < 4 {
				return 0, errors.New("Invalid Maximum Packet Size")
			}
			p.maxPacketSize = binary.BigEndian.Uint32(b[1:5])
			b = b[5:]
		case TopicAliasMax:
			if len(b[1:]) < 2 {
				return 0, errors.New("Invalid Topic Alias Maximum")
			}
			p.topicAliasMax = binary.BigEndian.Uint16(b[1:3])
			b = b[3:]
		case ReqRespInfo:
			if b[1] > 1 {
				return 0, errors.New("Invalid Request Response Info")
			}
			p.reqRespInfo = b[1] == 1
			b = b[2:]
		case ReqProbInfo:
			if b[1] > 1 {
				return 0, errors.New("Invalid Request Problem Info")
			}
			p.reqProbInfo = b[1] == 1
			b = b[2:]
		case UserProp:
			n, err := p.userProp.decode(b)
			if err != nil {
				return 0, errors.New("Invalid User Property")
			}
			b = b[1+n:]
			break
		case AuthMethod:
			n, err := p.authMethod.decode(b)
			if err != nil {
				return 0, errors.New("Invalid Authentication Method")
			}
			b = b[1+n:]
		case AuthData:
			n, err := p.authData.decode(b)
			if err != nil {
				return 0, errors.New("Invalid Authentication Data")
			}
			b = b[1+n:]
		default:
			return 0, errors.New("Unknown connect packet property")
		}
	}
	return rBytes + int(propLen), nil
}

type WillProperties struct {
	delayInt    time.Duration
	fmtInd      byte
	msgExpInt   time.Duration
	contentType UTF8String
	respTopic   UTF8String
	corrData    byte
	userProp    UTF8StringPair
}

func (p *WillProperties) decode(b []byte) (int, error) {
	p.delayInt = 0
	p.fmtInd = 0
	p.corrData = 0

	var pLen VarByteInt
	rBytes, err := pLen.decode(b)
	if err != nil {
		return 0, errors.New("Unable to decode will property length.")
	} else if pLen == 0 {
		return rBytes, nil
	}
	b = b[rBytes : rBytes+int(pLen)]

	for len(b) > 0 {
		switch MqttProperty(b[0]) {
		case WillDelayInt:
			if len(b[1:]) < 4 {
				return 0, errors.New("Invalid Will Delay Interval.")
			}
			p.delayInt = time.Duration(binary.BigEndian.Uint32(b[1:5]))
			p.delayInt *= time.Second
			b = b[5:]
		case PlFormatInd:
			if len(b[:1]) < 1 {
				return 0, errors.New("Invalid Payload Format Indicator.")
			}
			p.fmtInd = b[1]
			b = b[2:]
		case MsgExpInt:
			if len(b[1:]) < 4 {
				return 0, errors.New("Invalid Will Message Expiration Interval.")
			}
			p.msgExpInt = time.Duration(binary.BigEndian.Uint32(b[1:5]))
			p.msgExpInt *= time.Second
			b = b[5:]
		case ContentType:
			n, err := p.contentType.decode(b[1:])
			if err != nil {
				return 0, errors.New("Invalid Will Content Type.")
			}
			b = b[1+n:]
		case RespTopic:
			n, err := p.respTopic.decode(b[1:])
			if err != nil {
				return 0, errors.New("Invalid Response Topic.")
			}
			b = b[1+n:]
		case CorrData:
			if len(b[:1]) < 1 {
				return 0, errors.New("Invalid Correlation Data.")
			}
			p.corrData = b[1]
			b = b[2:]
		case UserProp:
			n, err := p.userProp.decode(b)
			if err != nil {
				return 0, errors.New("Invalid User Property")
			}
			b = b[1+n:]
		}
	}

	return rBytes + int(pLen), nil
}

type ConnectPayload struct {
	clientId UTF8String
	wProp    WillProperties
	wTopic   UTF8String
	wPayload BinaryData
	username UTF8String
	password BinaryData
}

func (pl *ConnectPayload) decode(f *ConnectFlag, b []byte) error {
	n, err := pl.clientId.decode(b)
	if err != nil {
		return err
	}
	b = b[n:]

	if f.will() {
		n, err = pl.wProp.decode(b)
		if err != nil {
			return err
		}
		b = b[n:]

		if n, err = pl.wTopic.decode(b); err != nil {
			return err
		}
		b = b[n:]

		if n, err = pl.wPayload.decode(b); err != nil {
			return err
		}
		b = b[n:]
	}

	if f.username() {
		if n, err = pl.username.decode(b); err != nil {
			return err
		}
		b = b[n:]
	}

	if f.password() {
		if n, err = pl.password.decode(b); err != nil {
			return err
		}
		b = b[n:]
	}

	return nil
}

type ConnectRequest struct {
	flag      ConnectFlag
	keepAlive time.Duration
	prop      ConnectProperties
}

func ParseConnect(p *MqttHeader, b []byte) (Request, error) {
	name := []byte{0, 4, 'M', 'Q', 'T', 'T'}
	if !bytes.Equal(name, b[:len(name)]) {
		return nil, errors.New("Unsupported protocol!")
	}
	b = b[len(name):]

	ver := []byte{5}
	if !bytes.Equal(ver, b[:len(ver)]) {
		return nil, errors.New("Unsupported protocol!")
	}
	b = b[len(ver):]

	flag := ConnectFlag(b[0])
	if err := flag.valid(); err != nil {
		return nil, err
	}
	b = b[flag.size():]

	keepAlive := time.Duration(binary.BigEndian.Uint16(b[:2])) * time.Second
	b = b[2:]

	var prop ConnectProperties
	n, err := prop.decode(b)
	if err != nil {
		return nil, err
	}
	b = b[n:]

	var pl ConnectPayload
	err = pl.decode(&flag, b)
	if err != nil {
		return nil, err
	}

	return &ConnectRequest{flag: flag, keepAlive: keepAlive, prop: prop}, nil
}

func (r *ConnectRequest) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

type ReasonCode byte

const (
	Success      ReasonCode = 0
	Unspecified             = 0x80
	Malformed               = 0x81
	ProtocolErr             = 0x82
	ImplSpecific            = 0x83
	Unsupported             = 0x84
	InvClientId             = 0x85
	BadUsrPass              = 0x86
	NotAuth                 = 0x87
	Unavailable             = 0x88
	Busy                    = 0x89
	Banned                  = 0x8A
	BadAuth                 = 0x8C
	InvalidTopic            = 0x90
	LargePacket             = 0x95
	ExceedQuota             = 0x97
	InvPlFormat             = 0x99
	NoSupRetain             = 0x9A
	NoSpQoS                 = 0x9B
	UseAnother              = 0x9C
	ServerMoved             = 0x9D
	ExceedRate              = 0x9F
)

func (r *ConnectRequest) Response() ([]byte, error) {
	// ackFlag := byte(0)
	// rc := Success

	return []byte{}, nil
}
