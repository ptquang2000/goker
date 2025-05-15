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

type CType byte

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

func (t CType) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	w.WriteByte(byte(t << 4))
	return w
}

func (t *CType) decode(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, errors.New("Missing packet type in header.")
	}
	*t = CType(b[0]>>4) & 0b0001111
	if *t <= RESERVED || *t >= COUNT {
		return 0, errors.New("Unknowned MQTT Control Packet type!")
	}
	return 1, nil
}

type Flag struct {
	dup    bool
	qos    bool
	retain bool
}

func (f Flag) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	var b byte
	if f.dup {
		b |= 0b0100
	}
	if f.qos {
		b |= 0b0010
	}
	if f.dup {
		b |= 0b0001
	}
	w.WriteByte(b)
	return w
}

func (f *Flag) decode(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, errors.New("Missing flag in header.")
	}
	*f = Flag{dup: bool(b[0]&0b0100 != 0), qos: bool(b[0]&0b0010 != 0), retain: bool(b[0]&0b0001 != 0)}
	return 1, nil
}

type MqttHeader struct {
	ctl  CType
	flag Flag
	len  VarByteInt
}

func (h MqttHeader) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	ctl, _ := h.ctl.encode().ReadByte()
	flag, _ := h.flag.encode().ReadByte()
	w.WriteByte(ctl | flag)
	h.len.encode().WriteTo(w)
	return w
}

func ParseHeader(b []byte) (RequestHeader, error) {
	h := &MqttHeader{}

	n, err := h.ctl.decode(b)
	if err != nil {
		return nil, err
	}
	n, err = h.flag.decode(b)
	if err != nil {
		return nil, err
	}
	b = b[n:]

	n, err = h.len.decode(b)
	if err != nil {
		return nil, err
	}

	return h, nil
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

func (qos QoS) maxQos() QoS {
	return QoS0
}

func (qos QoS) isSupported() bool {
	return qos <= qos.maxQos()
}

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
	SessionExpiryInterval            MqttProperty = 0x11
	ReceiveMaximum                                = 0x21
	MaximumPacketSize                             = 0x27
	TopicAliasMaximum                             = 0x22
	RequestResponseInformation                    = 0x19
	RequestProblemInformation                     = 0x17
	UserProperty                                  = 0x26
	AuthenticationMethod                          = 0x15
	AuthenticationData                            = 0x16
	WillDelayInterval                             = 0x18
	PayloadFormatIndicator                        = 0x01
	MessageExpiryInterval                         = 0x02
	ContentType                                   = 0x03
	ResponseTopic                                 = 0x08
	CorrelationData                               = 0x09
	MaximumQoS                                    = 0x24
	RetainAvailable                               = 0x25
	AssignedClientIdentifier                      = 0x12
	ReasonString                                  = 0x1F
	WildcardSubscriptionAvailable                 = 0x28
	SubscriptionIdentifiersAvailable              = 0x29
	SharedSubscriptionAvailable                   = 0x2A
	ServerKeepAlive                               = 0x13
	ResponseInformation                           = 0x1A
	ServerReference                               = 0x1C
)

func (p MqttProperty) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	w.WriteByte(byte(p))
	return w
}

type ConnectProperties struct {
	sessionExpiryInterval time.Duration
	receiveMaximum        TwoByteInteger
	maximumPacketSize     FourByteInteger
	topicAliasMaximum     TwoByteInteger
	requestResponseInfo   ByteInteger
	requestProblemInfo    ByteInteger
	userProperty          UTF8StringPair
	authenticationMethod  UTF8String
	authenticationData    BinaryData

	fields map[MqttProperty]bool
}

func (p *ConnectProperties) decode(b []byte) (int, error) {
	p.fields = make(map[MqttProperty]bool)
	p.sessionExpiryInterval = 0
	p.receiveMaximum = math.MaxUint16
	p.maximumPacketSize = math.MaxUint32
	p.topicAliasMaximum = 0
	p.requestResponseInfo = false
	p.requestProblemInfo = true

	var propLen VarByteInt
	rBytes, err := propLen.decode(b)
	if err != nil {
		return 0, err
	} else if len(b) < rBytes+int(propLen) {
		return rBytes, errors.New("Invalid Conenct Property Length.")
	} else if propLen == 0 {
		return rBytes, nil
	}

	b = b[rBytes : rBytes+int(propLen)]
	var n int
	for len(b) > 0 {
		mProp := MqttProperty(b[0])
		b = b[1:]
		if p.fields[mProp] {
			return 0, errors.New("Duplicate connect property")
		}
		p.fields[mProp] = true

		switch mProp {
		case SessionExpiryInterval:
			var d FourByteInteger
			if n, err = d.decode(b); err != nil {
				return 0, errors.New("Invalid Session Expiry Interval, err:" + err.Error())
			}
			p.sessionExpiryInterval = time.Duration(d) * time.Second
		case ReceiveMaximum:
			if n, err = p.receiveMaximum.decode(b); err != nil {
				return 0, errors.New("Invalid Receive Maximum, err:" + err.Error())
			}
		case MaximumPacketSize:
			if n, err = p.maximumPacketSize.decode(b); err != nil {
				return 0, errors.New("Invalid Maximum Packet Size, err:" + err.Error())
			}
		case TopicAliasMaximum:
			if n, err = p.topicAliasMaximum.decode(b); err != nil {
				return 0, errors.New("Invalid Topic Alias Maximum")
			}
		case RequestResponseInformation:
			if n, err = p.requestProblemInfo.decode(b); err != nil {
				return 0, errors.New("Invalid Request Response Information, err:" + err.Error())
			}
		case RequestProblemInformation:
			if n, err = p.requestProblemInfo.decode(b); err != nil {
				return 0, errors.New("Invalid Request Problem Information, err:" + err.Error())
			}
		case UserProperty:
			if n, err = p.userProperty.decode(b); err != nil {
				return 0, errors.New("Invalid User Property, err:" + err.Error())
			}
		case AuthenticationMethod:
			if n, err = p.authenticationMethod.decode(b); err != nil {
				return 0, errors.New("Invalid Authentication Method" + err.Error())
			}
		case AuthenticationData:
			if n, err = p.authenticationData.decode(b); err != nil {
				return 0, errors.New("Invalid Authentication Data" + err.Error())
			}
		default:
			return 0, errors.New("Unknown connect packet property")
		}
		b = b[n:]
		rBytes += 1 + n
	}
	return rBytes, nil
}

type WillProperties struct {
	delayInterval          time.Duration
	payloadFormatIndicator ByteInteger
	messageExpiryInterval  time.Duration
	contentType            UTF8String
	responseTopic          UTF8String
	correlationData        BinaryData
	userProperty           UTF8StringPair
}

func (p *WillProperties) decode(b []byte) (int, error) {
	p.delayInterval = 0
	p.payloadFormatIndicator = false

	var pLen VarByteInt
	rBytes, err := pLen.decode(b)
	if err != nil {
		return 0, errors.New("Unable to decode will property length.")
	} else if len(b) < rBytes+int(pLen) {
		return rBytes, errors.New("Invalid Conenct Property Length.")
	} else if pLen == 0 {
		return rBytes, nil
	}
	b = b[rBytes : rBytes+int(pLen)]

	var n int
	for len(b) > 0 {
		mProp := MqttProperty(b[0])
		b = b[1:]
		switch mProp {
		case WillDelayInterval:
			var d FourByteInteger
			if n, err = d.decode(b); err != nil {
				return 0, errors.New("Invalid Will Delay Interval, err:" + err.Error())
			}
			p.delayInterval = time.Duration(d) * time.Second
		case PayloadFormatIndicator:
			if n, err = p.payloadFormatIndicator.decode(b); err != nil {
				return 0, errors.New("Invalid Payload Format Indicator, err:" + err.Error())
			}
		case MessageExpiryInterval:
			var d FourByteInteger
			if n, err = d.decode(b); err != nil {
				return 0, errors.New("Invalid Will Message Expiration Interval, err:" + err.Error())
			}
			p.messageExpiryInterval = time.Duration(d) * time.Second
		case ContentType:
			if n, err = p.contentType.decode(b); err != nil {
				return 0, errors.New("Invalid Will Content Type, err:" + err.Error())
			}
		case ResponseTopic:
			if n, err = p.responseTopic.decode(b); err != nil {
				return 0, errors.New("Invalid Response Topic, err:" + err.Error())
			}
		case CorrelationData:
			if n, err = p.correlationData.decode(b); err != nil {
				return 0, errors.New("Invalid Correlation Data, err:" + err.Error())
			}
		case UserProperty:
			if n, err = p.userProperty.decode(b); err != nil {
				return 0, errors.New("Invalid User Property, err:" + err.Error())
			}
		default:
			return 0, errors.New("Unknown connect will property")
		}
		b = b[n:]
		rBytes += 1 + n
	}
	return rBytes, nil
}

type ConnectPayload struct {
	clientIdentifier UTF8String
	willProperties   WillProperties
	willTopic        UTF8String
	willPayload      BinaryData
	username         UTF8String
	password         BinaryData
}

func (pl *ConnectPayload) decode(f *ConnectFlag, b []byte) error {
	n, err := pl.clientIdentifier.decode(b)
	if err != nil {
		return err
	}
	b = b[n:]

	if f.will() {
		if n, err = pl.willProperties.decode(b); err != nil {
			return err
		}
		b = b[n:]

		if n, err = pl.willTopic.decode(b); err != nil {
			return err
		}
		b = b[n:]

		if n, err = pl.willPayload.decode(b); err != nil {
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
	payload   ConnectPayload
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
	if err = pl.decode(&flag, b); err != nil {
		return nil, err
	}

	return &ConnectRequest{flag: flag, keepAlive: keepAlive, prop: prop, payload: pl}, nil
}

type ReasonCode byte

const (
	Success                    ReasonCode = 0
	Unspecified                           = 0x80
	MalformedPacket                       = 0x81
	ProtocolError                         = 0x82
	ImplementationSpecific                = 0x83
	UnsupportedProtocolVersion            = 0x84
	InvalidClientIdentifier               = 0x85
	BadUsernamePassword                   = 0x86
	NotAuthorized                         = 0x87
	ServerUnavailable                     = 0x88
	ServerBusy                            = 0x89
	Banned                                = 0x8A
	BadAuthenticationMethod               = 0x8C
	InvalidTopicName                      = 0x90
	PacketTooLarge                        = 0x95
	ExceedQuota                           = 0x97
	InvalidPayloadFormat                  = 0x99
	RetainNotSupported                    = 0x9A
	QoSNotSupported                       = 0x9B
	UseAnotherServer                      = 0x9C
	ServerMoved                           = 0x9D
	ExceededConnectionRate                = 0x9F
)

func (p ReasonCode) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	w.WriteByte(byte(p))
	return w
}

type ConnackProperties struct {
	sessionExpiryInterval           time.Duration
	receiveMaximum                  TwoByteInteger
	maximumQoS                      QoS
	retainAvailable                 ByteInteger
	maximumPacketSize               FourByteInteger
	assignedClientIdentifier        UTF8String
	topicAliasMinimum               TwoByteInteger
	reasonString                    UTF8String
	userProperty                    UTF8StringPair
	wildcardSubscriptionAvailable   ByteInteger
	subscriptionIdentifiersAvaiable ByteInteger
	sharedSubscriptionAvaiable      ByteInteger
	serverKeepAlive                 TwoByteInteger
	responseInformation             UTF8String
	serverReference                 UTF8String
	authenticationMethod            UTF8String
	authenticationData              BinaryData
}

func (p *ConnackProperties) encode(flag *ConnectFlag, prop *ConnectProperties) (w *bytes.Buffer, rc ReasonCode) {
	rc = Success
	w = bytes.NewBuffer(make([]byte, 0))

	// TODO: Session Expiry Interval

	// TODO: Received Maximum

	if !flag.qos().isSupported() {
		MqttProperty(MaximumQoS).encode().WriteTo(w)
		ByteInteger(flag.qos().maxQos() >= QoS1).encode().WriteTo(w)

		rc = QoSNotSupported
		return
	}

	// WARNING: Should get retail available from server configuration
	if true {
		MqttProperty(RetainAvailable).encode().WriteTo(w)
		ByteInteger(false).encode().WriteTo(w)

		if flag.retain() {
			rc = RetainNotSupported
			return
		}
	}

	// TODO: Maximum Packet Size

	// TODO: Assigned Client Identifier

	// TODO: Topic Alias Maximum

	// TODO: Reason String

	// TODO: User Property

	// WARNING: Should get wildcard subscription available from server configuration
	if true {
		MqttProperty(WildcardSubscriptionAvailable).encode().WriteTo(w)
		ByteInteger(false).encode().WriteTo(w)
	}

	// WARNING: Should get wildcard subscription available from server configuration
	if true {
		MqttProperty(SubscriptionIdentifiersAvailable).encode().WriteTo(w)
		ByteInteger(false).encode().WriteTo(w)
	}

	// WARNING: Should get wildcard subscription available from server configuration
	if true {
		MqttProperty(SharedSubscriptionAvailable).encode().WriteTo(w)
		ByteInteger(false).encode().WriteTo(w)
	}

	// TODO: Keep Alive

	// TODO: Response Information

	// TODO: Server Reference

	// TODO: Authentication Method

	// TODO: Authentication Data

	return
}

func (r *ConnectRequest) Response() (w *bytes.Buffer, err error) {
	w = bytes.NewBuffer(make([]byte, 0))

	ackFlag := make([]byte, 1)
	if !r.flag.cleanstart() == false /*&& hasSession(r.payload.clientIdentifier)*/ {
		ackFlag[0] = 0b1
	}
	w.Write(ackFlag)

	var prop ConnackProperties
	buf, rc := prop.encode(&r.flag, &r.prop)
	rc.encode().WriteTo(w)

	blen := VarByteInt(buf.Len())
	blen.encode().WriteTo(w)
	buf.WriteTo(w)

	if rc != Success {
		err = errors.New("Unsupported Connect Propeties")
		return
	}

	return
}

func (r *ConnectRequest) WriteTo(w io.Writer) (int64, error) {
	wBytes := int64(0)

	body, err := r.Response()
	if err != nil {
		return 0, err
	}
	header := MqttHeader{ctl: CONNACK, flag: Flag{}, len: VarByteInt(body.Len())}

	n, err := header.encode().WriteTo(w)
	if err != nil {
		return 0, err
	}
	wBytes += n

	n, err = body.WriteTo(w)
	if err != nil {
		return 0, err
	}
	wBytes += n

	return int64(wBytes), nil
}
