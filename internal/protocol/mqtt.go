package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

func (t *CType) decode(r *bytes.Buffer) error {
	b, err := r.ReadByte()
	if err != nil {
		return errors.New("Missing packet type in header.")
	}
	*t = CType(b>>4) & 0b0001111
	if *t <= RESERVED || *t >= COUNT {
		return errors.New("Unknowned MQTT Control Packet type!")
	}
	r.UnreadByte()
	return nil
}

type Flag struct {
	dup    bool
	qos    QoS
	retain bool
}

func (f Flag) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	var b byte
	if f.dup {
		b |= 0b1000
	}
	b |= byte(f.qos) << 1 & 0b0110
	if f.dup {
		b |= 0b0001
	}
	w.WriteByte(b)
	return w
}

func (f *Flag) decode(r *bytes.Buffer) error {
	b, err := r.ReadByte()
	if err != nil {
		return errors.New("Missing flag in header.")
	}
	*f = Flag{dup: bool(b&0b1000 != 0), qos: QoS(b & 0b0110 >> 1), retain: bool(b&0b0001 != 0)}
	r.UnreadByte()
	return nil
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

func ParseHeader(r *bytes.Buffer) (RequestHeader, error) {
	h := &MqttHeader{}

	err := h.ctl.decode(r)
	if err != nil {
		return nil, errors.New("Malformed Fixed Header, err:" + err.Error())
	}
	err = h.flag.decode(r)
	if err != nil {
		return nil, errors.New("Malformed Fixed Header, err:" + err.Error())
	}
	r.Next(1)

	if h.len.decode(r) != nil {
		return nil, err
	}

	return h, nil
}

func (p *MqttHeader) BodyLength() int {
	return int(p.len)
}

func (p *MqttHeader) ParseBody(r *bytes.Buffer) (Request, error) {
	switch p.ctl {
	case CONNECT:
		return ParseConnect(p, r)
	case PUBLISH:
		return ParsePublish(p, r)
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
	TopicAlias                                    = 0x23
	SubscriptionIdentifier                        = 0x0B
)

func (p MqttProperty) encode() *bytes.Buffer {
	w := bytes.NewBuffer(make([]byte, 0))
	w.WriteByte(byte(p))
	return w
}

type ConnectProperties struct {
	PacketProperties
	sessionExpiryInterval time.Duration
	receiveMaximum        TwoByteInteger
	maximumPacketSize     FourByteInteger
	topicAliasMaximum     TwoByteInteger
	requestResponseInfo   ByteInteger
	requestProblemInfo    ByteInteger
	userProperty          UTF8StringPair
	authenticationMethod  UTF8String
	authenticationData    BinaryData
}

func (p *ConnectProperties) decode(r *bytes.Buffer) error {
	p.fields = make(map[MqttProperty]bool)
	p.sessionExpiryInterval = 0
	p.receiveMaximum = math.MaxUint16
	p.maximumPacketSize = math.MaxUint32
	p.topicAliasMaximum = 0
	p.requestResponseInfo = false
	p.requestProblemInfo = true

	var propLen VarByteInt
	err := propLen.decode(r)
	if err != nil {
		return err
	} else if r.Len() < int(propLen) {
		return errors.New("Invalid Conenct Property Length.")
	} else if propLen == 0 {
		return nil
	}

	remain := r.Len()
	for remain-r.Len() < int(propLen) {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		mProp := MqttProperty(b)
		if p.fields[mProp] {
			return errors.New("Duplicate connect property")
		}
		p.fields[mProp] = true

		switch mProp {
		case SessionExpiryInterval:
			var d FourByteInteger
			if err = d.decode(r); err != nil {
				return errors.New("Invalid Session Expiry Interval, err:" + err.Error())
			}
			p.sessionExpiryInterval = time.Duration(d) * time.Second
		case ReceiveMaximum:
			if err = p.receiveMaximum.decode(r); err != nil {
				return errors.New("Invalid Receive Maximum, err:" + err.Error())
			}
		case MaximumPacketSize:
			if err = p.maximumPacketSize.decode(r); err != nil {
				return errors.New("Invalid Maximum Packet Size, err:" + err.Error())
			}
		case TopicAliasMaximum:
			if err = p.topicAliasMaximum.decode(r); err != nil {
				return errors.New("Invalid Topic Alias Maximum")
			}
		case RequestResponseInformation:
			if err = p.requestProblemInfo.decode(r); err != nil {
				return errors.New("Invalid Request Response Information, err:" + err.Error())
			}
		case RequestProblemInformation:
			if err = p.requestProblemInfo.decode(r); err != nil {
				return errors.New("Invalid Request Problem Information, err:" + err.Error())
			}
		case UserProperty:
			if err = p.userProperty.decode(r); err != nil {
				return errors.New("Invalid User Property, err:" + err.Error())
			}
		case AuthenticationMethod:
			if err = p.authenticationMethod.decode(r); err != nil {
				return errors.New("Invalid Authentication Method" + err.Error())
			}
		case AuthenticationData:
			if err = p.authenticationData.decode(r); err != nil {
				return errors.New("Invalid Authentication Data" + err.Error())
			}
		default:
			return errors.New("Unknown connect packet property")
		}
	}
	return nil
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

func (p *WillProperties) decode(r *bytes.Buffer) error {
	p.delayInterval = 0
	p.payloadFormatIndicator = false

	var propLen VarByteInt
	err := propLen.decode(r)
	if err != nil {
		return errors.New("Unable to decode will property length.")
	} else if r.Len() < int(propLen) {
		return errors.New("Invalid Will Properties Length.")
	} else if propLen == 0 {
		return nil
	}

	remain := r.Len()
	for remain-r.Len() < int(propLen) {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		mProp := MqttProperty(b)
		switch mProp {
		case WillDelayInterval:
			var d FourByteInteger
			if err = d.decode(r); err != nil {
				return errors.New("Invalid Will Delay Interval, err:" + err.Error())
			}
			p.delayInterval = time.Duration(d) * time.Second
		case PayloadFormatIndicator:
			if err = p.payloadFormatIndicator.decode(r); err != nil {
				return errors.New("Invalid Payload Format Indicator, err:" + err.Error())
			}
		case MessageExpiryInterval:
			var d FourByteInteger
			if err = d.decode(r); err != nil {
				return errors.New("Invalid Will Message Expiration Interval, err:" + err.Error())
			}
			p.messageExpiryInterval = time.Duration(d) * time.Second
		case ContentType:
			if err = p.contentType.decode(r); err != nil {
				return errors.New("Invalid Will Content Type, err:" + err.Error())
			}
		case ResponseTopic:
			if err = p.responseTopic.decode(r); err != nil {
				return errors.New("Invalid Response Topic, err:" + err.Error())
			}
		case CorrelationData:
			if err = p.correlationData.decode(r); err != nil {
				return errors.New("Invalid Correlation Data, err:" + err.Error())
			}
		case UserProperty:
			if err = p.userProperty.decode(r); err != nil {
				return errors.New("Invalid User Property, err:" + err.Error())
			}
		default:
			return errors.New("Unknown connect will property")
		}
	}
	return nil
}

type ConnectPayload struct {
	clientIdentifier UTF8String
	willProperties   WillProperties
	willTopic        UTF8String
	willPayload      BinaryData
	username         UTF8String
	password         BinaryData
}

func (pl *ConnectPayload) decode(f *ConnectFlag, r *bytes.Buffer) error {
	if err := pl.clientIdentifier.decode(r); err != nil {
		return err
	}

	if f.will() {
		if err := pl.willProperties.decode(r); err != nil {
			return err
		}

		if err := pl.willTopic.decode(r); err != nil {
			return err
		}

		if err := pl.willPayload.decode(r); err != nil {
			return err
		}
	}

	if f.username() {
		if err := pl.username.decode(r); err != nil {
			return err
		}
	}

	if f.password() {
		if err := pl.password.decode(r); err != nil {
			return err
		}
	}

	return nil
}

type ConnectRequest struct {
	flag      ConnectFlag
	keepAlive time.Duration
	prop      ConnectProperties
	payload   ConnectPayload
}

func ParseConnect(p *MqttHeader, r *bytes.Buffer) (Request, error) {
	var b []byte

	name := []byte{0, 4, 'M', 'Q', 'T', 'T'}
	b = make([]byte, len(name))
	if _, err := r.Read(b); err != nil || !bytes.Equal(name, b) {
		return nil, errors.New("Unsupported protocol!")
	}

	ver := []byte{5}
	b = make([]byte, len(ver))
	if _, err := r.Read(b); err != nil || !bytes.Equal(ver, b) {
		return nil, errors.New("Unsupported protocol!")
	}

	b = make([]byte, 1)
	if _, err := r.Read(b); err != nil {
		return nil, errors.New("Missing flag value.")
	}
	flag := ConnectFlag(b[0])
	if err := flag.valid(); err != nil {
		return nil, err
	}

	b = make([]byte, 2)
	if _, err := r.Read(b); err != nil {
		return nil, errors.New("Missing keep alive value.")
	}
	keepAlive := time.Duration(binary.BigEndian.Uint16(b)) * time.Second

	var prop ConnectProperties
	if err := prop.decode(r); err != nil {
		return nil, err
	}

	var pl ConnectPayload
	if err := pl.decode(&flag, r); err != nil {
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

func (req *ConnectRequest) ToString() string {
	buf := bytes.NewBuffer(make([]byte, 0))

	buf.WriteString(fmt.Sprintf("keepAlive: %s", req.keepAlive.String()))
	buf.WriteString(fmt.Sprintf("clientId: %s", req.payload.clientIdentifier))
	buf.WriteString(fmt.Sprintf("username: %s", req.payload.username))

	return buf.String()
}

func (r *ConnectRequest) ResponseTo(w io.Writer) (int64, error) {
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

type PublishRequest struct {
	topic    UTF8String
	packetId TwoByteInteger
	prop     PublishProperties
	pl       []byte
}

type PublishProperties struct {
	PacketProperties
	payloadFormatIndicator ByteInteger
	messageExpiryInterval  time.Duration
	topicAlias             TwoByteInteger
	responseTopic          UTF8String
	correlationData        BinaryData
	userProperty           UTF8StringPair
	subscriptionIdentifier VarByteInt
	contentType            UTF8String
}

func (p *PublishProperties) decode(r *bytes.Buffer) error {
	p.fields = make(map[MqttProperty]bool)
	p.payloadFormatIndicator = false

	var propLen VarByteInt
	err := propLen.decode(r)
	if err != nil {
		return errors.New("Unable to decode publish property length.")
	} else if r.Len() < int(propLen) {
		return errors.New("Publish property must match set length.")
	} else if propLen == 0 {
		return nil
	}

	remain := r.Len()
	for remain-r.Len() < int(propLen) {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		mProp := MqttProperty(b)
		if p.fields[mProp] {
			return errors.New("Duplicate connect property")
		}
		p.fields[mProp] = true

		switch mProp {
		case PayloadFormatIndicator:
			if err = p.payloadFormatIndicator.decode(r); err != nil {
				return errors.New("Invalid Payload Format Indicator, err:" + err.Error())
			}
		case MessageExpiryInterval:
			var d FourByteInteger
			if err = d.decode(r); err != nil {
				return errors.New("Invalid Will Message Expiration Interval, err:" + err.Error())
			}
			p.messageExpiryInterval = time.Duration(d) * time.Second
		case TopicAlias:
			if err = p.topicAlias.decode(r); err != nil {
				return errors.New("Invalid Topic Alias, err:" + err.Error())
			}
		case ResponseTopic:
			if err = p.responseTopic.decode(r); err != nil {
				return errors.New("Invalid Response Topic, err:" + err.Error())
			}
		case CorrelationData:
			if err = p.correlationData.decode(r); err != nil {
				return errors.New("Invalid Correlation Data, err:" + err.Error())
			}
		case UserProperty:
			if err = p.userProperty.decode(r); err != nil {
				return errors.New("Invalid User Property, err:" + err.Error())
			}
		case SubscriptionIdentifier:
			if err = p.subscriptionIdentifier.decode(r); err != nil || p.subscriptionIdentifier == 0 {
				return errors.New("Invalid Subscription Identifier, err:" + err.Error())
			}
		case ContentType:
			if err = p.contentType.decode(r); err != nil {
				return errors.New("Invalid Will Content Type, err:" + err.Error())
			}
		default:
			return errors.New("Unknown publish property")
		}
	}
	return nil
}

func ParsePublish(h *MqttHeader, r *bytes.Buffer) (Request, error) {
	req := &PublishRequest{}

	if err := req.topic.decode(r); err != nil {
		return nil, errors.New("Unable to parse public topic name, err:" + err.Error())
	}

	// TODO: Wildcard and Subscription's Topic Filter checking

	if h.flag.qos > QoS0 {
		if err := req.packetId.decode(r); err != nil {
			return nil, err
		}
	}

	if err := req.prop.decode(r); err != nil {
		return nil, err
	}

	req.pl = make([]byte, r.Len())
	if _, err := r.Read(req.pl); err != nil {
		return nil, errors.New("Error reading publish payload")
	}

	return req, nil
}

func (req *PublishRequest) ToString() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	buf.WriteString(fmt.Sprintf("packet: PUBLISH, "))
	buf.WriteString(fmt.Sprintf("topic: %s, ", req.topic))
	buf.WriteString(fmt.Sprintf("packId: %d, ", req.packetId))
	buf.WriteString(fmt.Sprintf("payload: "))
	buf.Write(req.pl)

	return buf.String()
}

func (req *PublishRequest) ResponseTo(w io.Writer) (int64, error) {
	return 0, nil
}
