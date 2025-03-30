package x224

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"rdp_channel/protocol/tpkt"
)

/* 协议常量 */
const (
	X224_HEADER_LENGTH = 0x06
	RDP_NEG_LENGTH     = 0x08
)

// X224消息类型字段标识
const (
	X224_CONNECTION_REQUEST byte = 0xE0
	X224_CONNECTION_CONFIRM byte = 0xD0
	X224_DISCONNECT_REQUEST byte = 0x80
	X224_DATA               byte = 0xF0
	X224_ERROR              byte = 0x70
)

// 安全协议协商状态字段标识
const (
	RDP_NEG_REQ  byte = 0x01
	RDP_NEG_RSP  byte = 0x02
	RDP_NEG_FAIL byte = 0x03
)

// 可选协议
const (
	PROTOCOL_RDP uint32 = 0x0000
	PROTOCOL_SSL uint32 = 0x0001
)

// Negotiation 安全协议协商结构体
type Negotiation struct {
	Type    byte
	Flags   uint8
	Length  uint16
	Payload uint32
}

// 格式化输出
func (neg *Negotiation) String() string {
	buff := &bytes.Buffer{}
	buff.WriteString(fmt.Sprintf("    Type: 0x%02X\n", neg.Type))
	buff.WriteString(fmt.Sprintf("    Flags: 0x%02X\n", neg.Flags))
	buff.WriteString(fmt.Sprintf("    Length: 0x%04X (%d bytes)\n", neg.Length, neg.Length))
	buff.WriteString(fmt.Sprintf("    Payload: 0x%08X\n", neg.Payload))

	return buff.String()
}

func (neg *Negotiation) parseNegotiation(reader *bytes.Reader) error {
	var err error

	err = binary.Read(reader, binary.LittleEndian, &neg.Type)
	if err != nil {
		return errors.New("[X224] failed to read pdu neg type: " + err.Error())
	}

	err = binary.Read(reader, binary.LittleEndian, &neg.Flags)
	if err != nil {
		return errors.New("[X224] failed to read pdu neg flags: " + err.Error())
	}

	err = binary.Read(reader, binary.LittleEndian, &neg.Length)
	if err != nil {
		return errors.New("[X224] failed to read pdu neg length: " + err.Error())
	}

	err = binary.Read(reader, binary.LittleEndian, &neg.Payload)
	if err != nil {
		return errors.New("[X224] failed to read pdu neg payload: " + err.Error())
	}

	return nil
}

// X224 协议封装
type X224 struct {
	transport   *tpkt.TPKT
	reqProtocol uint32
	selProtocol uint32
}

type X224PDU struct {
	Len        uint8
	Type       byte
	DstRef     uint16 // 大端序
	SrcRef     uint16 // 大端序
	ClsOpt     uint8
	Cookie     []byte
	NegPayload *Negotiation
}

// 格式化输出
func (pdu *X224PDU) String() string {
	buff := &bytes.Buffer{}
	buff.WriteString("X224PDU {\n")
	buff.WriteString(fmt.Sprintf("  Len: 0x%02X (%d bytes)\n", pdu.Len, pdu.Len))
	buff.WriteString(fmt.Sprintf("  Type: 0x%02X\n", pdu.Type))
	buff.WriteString(fmt.Sprintf("  DstRef: 0x%04X (BigEndian)\n", pdu.DstRef))
	buff.WriteString(fmt.Sprintf("  SrcRef: 0x%04X (BigEndian)\n", pdu.SrcRef))
	buff.WriteString(fmt.Sprintf("  ClsOpt: 0x%02X\n", pdu.ClsOpt))
	buff.WriteString("  Cookie: ")
	if len(pdu.Cookie) > 0 {
		buff.WriteString(fmt.Sprintf("%q\n", string(pdu.Cookie)))
	} else {
		buff.WriteString("(empty)\n")
	}
	if pdu.NegPayload != nil {
		buff.WriteString("  NegPayload: {\n")
		buff.WriteString(pdu.NegPayload.String())
		buff.WriteString("  }\n")
	} else {
		buff.WriteString("  NegPayload: <nil>\n")
	}
	buff.WriteString("}\n")

	return buff.String()
}

func New(transport *tpkt.TPKT) *X224 {
	return &X224{
		transport:   transport,
		reqProtocol: PROTOCOL_SSL,
	}
}

func (x *X224) Read() (int, []byte, error) {
	return x.transport.Read()
}

func (x *X224) Write(data []byte) (int, error) {
	return x.transport.Write(data)
}

func (x *X224) parsePduHeader(reader *bytes.Reader, pdu *X224PDU) error {
	var err error

	// 读取Len字段
	err = binary.Read(reader, binary.LittleEndian, &pdu.Len)
	if err != nil {
		return errors.New("[X224] failed to read pdu length: " + err.Error())
	}

	// 读取Type字段
	err = binary.Read(reader, binary.LittleEndian, &pdu.Type)
	if err != nil {
		return errors.New("[X224] failed to read pdu type: " + err.Error())
	}

	// 读取DstRef（大端序）
	err = binary.Read(reader, binary.BigEndian, &pdu.DstRef)
	if err != nil {
		return errors.New("[X224] failed to read pdu dstref: " + err.Error())
	}

	// 读取SrcRef（大端序）
	err = binary.Read(reader, binary.BigEndian, &pdu.SrcRef)
	if err != nil {
		return errors.New("[X224] failed to read pdu srcref: " + err.Error())
	}

	// 读取ClsOpt
	err = binary.Read(reader, binary.LittleEndian, &pdu.ClsOpt)
	if err != nil {
		return errors.New("[X224] failed to read pdu clsopt: " + err.Error())
	}

	return nil
}

// 序列化X224PDU
func (x *X224) serialize(pdu *X224PDU) []byte {
	buff := bytes.NewBuffer(nil)
	_ = binary.Write(buff, binary.LittleEndian, pdu.Len)
	_ = binary.Write(buff, binary.LittleEndian, pdu.Type)
	_ = binary.Write(buff, binary.BigEndian, pdu.DstRef)
	_ = binary.Write(buff, binary.BigEndian, pdu.SrcRef)
	_ = binary.Write(buff, binary.LittleEndian, pdu.ClsOpt)

	// 仅连接相关PDU包含Cookie和协商负载
	if pdu.Type == X224_CONNECTION_REQUEST || pdu.Type == X224_CONNECTION_CONFIRM {
		if len(pdu.Cookie) > 0 {
			_ = binary.Write(buff, binary.LittleEndian, pdu.Cookie)
			_ = binary.Write(buff, binary.LittleEndian, []byte{0x0D, 0x0A})
		}

		if pdu.NegPayload != nil {
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegPayload.Type)
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegPayload.Flags)
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegPayload.Length)
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegPayload.Payload)
		}
	}

	return buff.Bytes()
}

/*
	X224客户端相关实现
*/

// ConnectToServer 客户端向服务端发起连接请求
func (x *X224) ConnectToServer() error {
	cookie := []byte("Cookie: mstshash=yv1ing")

	/* 构造pdu */
	reqPdu := &X224PDU{
		Len:    X224_HEADER_LENGTH + uint8(len(cookie)+0x02+RDP_NEG_LENGTH), // 头部长度 + Cookie长度 + CRLF + Neg字段
		Type:   X224_CONNECTION_REQUEST,
		DstRef: 0x00,
		SrcRef: 0x00,
		ClsOpt: 0x0,
		Cookie: cookie,
		NegPayload: &Negotiation{
			Type:    RDP_NEG_RSP,
			Flags:   0x00,
			Length:  RDP_NEG_LENGTH,
			Payload: x.reqProtocol,
		},
	}

	/* 序列化pdu */
	payload := x.serialize(reqPdu)

	/* 写入传输层 */
	_, err := x.Write(payload)
	if err != nil {
		return err
	}

	/* 等待处理服务端对连接请求的响应 */
	return x.handleConnectionConfirm()
}

// handleConnectionConfirm 客户端处理服务端对连接请求的响应
func (x *X224) handleConnectionConfirm() error {
	li, packet, err := x.Read()
	if err != nil {
		return err
	}

	if li < 0x07 {
		return errors.New("[X224] invalid packet")
	}

	resPdu := &X224PDU{}
	reader := bytes.NewReader(packet)

	err = x.parsePduHeader(reader, resPdu)
	if err != nil {
		return errors.New("[X224] failed to parse pdu header: " + err.Error())
	}

	// 读取安全协议协商结果
	neg := &Negotiation{}
	err = neg.parseNegotiation(reader)
	if err != nil {
		return err
	}

	resPdu.NegPayload = neg

	/* 完成安全协议协商 */
	fmt.Printf("client received server's confirm: \n%+v\n", resPdu.String())
	return nil
}

/*
	X224服务端相关实现
*/

// 服务端向客户端发送响应
func (x *X224) responseToClient(reqPdu *X224PDU) error {
	var err error

	// 构造协商响应
	resPdu := &X224PDU{
		Len:    X224_HEADER_LENGTH + RDP_NEG_LENGTH, // 头部长度 + Neg字段
		Type:   X224_CONNECTION_CONFIRM,
		DstRef: reqPdu.SrcRef,
		SrcRef: reqPdu.DstRef,
		ClsOpt: reqPdu.ClsOpt,
		NegPayload: &Negotiation{
			Type:    RDP_NEG_RSP,
			Flags:   0x00,
			Length:  RDP_NEG_LENGTH,
			Payload: x.selProtocol,
		},
	}

	payload := x.serialize(resPdu)
	_, err = x.Write(payload)
	if err != nil {
		return errors.New("[X224] failed to write response: " + err.Error())
	}

	return nil
}

// 服务端处理客户端发来的连接请求
func (x *X224) handleConnectionRequest(packet []byte) error {
	var err error

	reqPdu := &X224PDU{}
	reader := bytes.NewReader(packet)

	err = x.parsePduHeader(reader, reqPdu)
	if err != nil {
		return err
	}

	// 解析Cookie
	cookieBuff := make([]byte, 0, 32)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return errors.New("[X224] failed to read cookie: " + err.Error())
		}
		cookieBuff = append(cookieBuff, b)
		if len(cookieBuff) >= 2 && bytes.Equal(cookieBuff[len(cookieBuff)-2:], []byte{0x0D, 0x0A}) {
			break
		}
	}
	reqPdu.Cookie = cookieBuff[:len(cookieBuff)-2] // 去掉结尾CRLF

	// 解析协商请求
	reqNeg := &Negotiation{}
	if err := reqNeg.parseNegotiation(reader); err != nil {
		return errors.New("[X224] failed to parse negotiation: " + err.Error())
	}
	reqPdu.NegPayload = reqNeg

	// 确定使用协议
	x.selProtocol = PROTOCOL_SSL

	/* 完成安全协议协商 */
	fmt.Printf("server received client's request: \n%+v\n", reqPdu.String())

	// 响应请求
	return x.responseToClient(reqPdu)
}
