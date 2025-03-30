package x224

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"rdp_channel/protocol/core/transport"
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
	transport   transport.Transport
	reqProtocol uint32
	selProtocol uint32

	dataHandlers  []func([]byte)
	errorHandlers []func(error)
}

type X224PDU struct {
	Len     uint8
	Type    byte
	DstRef  uint16 // 大端序
	SrcRef  uint16 // 大端序
	ClsOpt  uint8
	Cookie  []byte
	Payload []byte
	NegMsg  *Negotiation
}

func New(transport transport.Transport) *X224 {
	return &X224{
		transport:   transport,
		reqProtocol: PROTOCOL_SSL,
	}
}

/* 注册事件回调函数 */

// OnData 数据回调
func (x *X224) OnData(handler func([]byte)) {
	x.dataHandlers = append(x.dataHandlers, handler)
}

// OnError 错误回调
func (x *X224) OnError(handler func(error)) {
	x.errorHandlers = append(x.errorHandlers, handler)
}

// 从字节流中解析PDU头部
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
	switch pdu.Type {
	case X224_CONNECTION_REQUEST, X224_CONNECTION_CONFIRM:
		if len(pdu.Cookie) > 0 {
			_ = binary.Write(buff, binary.LittleEndian, pdu.Cookie)
			_ = binary.Write(buff, binary.LittleEndian, []byte{0x0D, 0x0A})
		}

		if pdu.NegMsg != nil {
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegMsg.Type)
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegMsg.Flags)
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegMsg.Length)
			_ = binary.Write(buff, binary.LittleEndian, pdu.NegMsg.Payload)
		}
	case X224_DATA:
		buff.Write(pdu.Payload)
	}

	return buff.Bytes()
}

// 处理数据消息
func (x *X224) handleData(reader *bytes.Reader) {
	buff, err := io.ReadAll(reader)
	if err != nil {
		x.handleError(err)
		return
	}

	for _, handler := range x.dataHandlers {
		handler(buff)
	}
}

// 处理错误消息
func (x *X224) handleError(err error) {
	log.Println(err.Error())

	for _, handler := range x.errorHandlers {
		handler(err)
	}
}

// Write 发送数据消息
func (x *X224) Write(data []byte) {
	// 构造pdu
	reqPdu := &X224PDU{
		Len:     X224_HEADER_LENGTH + uint8(len(data)), // 头部长度 + 数据长度
		Type:    X224_DATA,
		DstRef:  0x00,
		SrcRef:  0x00,
		ClsOpt:  0x0,
		Payload: data,
	}

	// 序列化pdu
	payload := x.serialize(reqPdu)

	// 写入传输层
	_, err := x.transport.Write(payload)
	if err != nil {
		x.handleError(err)
	}
}

/*
	X224客户端相关实现
*/

// ConnectToServer 客户端向服务端发起连接请求
func (x *X224) ConnectToServer() {
	cookie := []byte("Cookie: mstshash=yv1ing")

	/* 构造pdu */
	reqPdu := &X224PDU{
		Len:    X224_HEADER_LENGTH + uint8(len(cookie)+0x02+RDP_NEG_LENGTH), // 头部长度 + Cookie长度 + CRLF + Neg字段
		Type:   X224_CONNECTION_REQUEST,
		DstRef: 0x00,
		SrcRef: 0x00,
		ClsOpt: 0x0,
		Cookie: cookie,
		NegMsg: &Negotiation{
			Type:    RDP_NEG_RSP,
			Flags:   0x00,
			Length:  RDP_NEG_LENGTH,
			Payload: x.reqProtocol,
		},
	}

	/* 序列化pdu */
	payload := x.serialize(reqPdu)

	/* 写入传输层 */
	_, err := x.transport.Write(payload)
	if err != nil {
		x.handleError(errors.New("[X224] failed to write pdu: " + err.Error()))
		return
	}

	/* 等待处理服务端对连接请求的响应 */
	go x.clientHandleServerMessage()
}

// 客户端处理服务端的消息
func (x *X224) clientHandleServerMessage() {
	for {
		li, packet, err := x.transport.Read()
		if err != nil {
			continue
		}

		if li < 0x07 {
			x.handleError(errors.New("[X224] invalid packet"))
			return
		}

		resPdu := &X224PDU{}
		reader := bytes.NewReader(packet)

		err = x.parsePduHeader(reader, resPdu)
		if err != nil {
			x.handleError(errors.New("[X224] failed to parse pdu header: " + err.Error()))
			return
		}

		switch resPdu.Type {
		case X224_CONNECTION_CONFIRM:
			x.clientHandleConnectionConfirm(resPdu, reader)
		case X224_DATA:
			x.handleData(reader)
		}
	}
}

// handleConnectionConfirm 客户端处理服务端对连接请求的响应
func (x *X224) clientHandleConnectionConfirm(resPdu *X224PDU, reader *bytes.Reader) {
	// 读取安全协议协商结果
	neg := &Negotiation{}
	err := neg.parseNegotiation(reader)
	if err != nil {
		x.handleError(errors.New("[X224] failed to parse negotiation: " + err.Error()))
	}

	resPdu.NegMsg = neg
}

/*
	X224服务端相关实现
*/

// 服务端向客户端发送响应
func (x *X224) serverResponseToClient(reqPdu *X224PDU) {
	var err error

	// 构造协商响应
	resPdu := &X224PDU{
		Len:    X224_HEADER_LENGTH + RDP_NEG_LENGTH, // 头部长度 + Neg字段
		Type:   X224_CONNECTION_CONFIRM,
		DstRef: reqPdu.SrcRef,
		SrcRef: reqPdu.DstRef,
		ClsOpt: reqPdu.ClsOpt,
		NegMsg: &Negotiation{
			Type:    RDP_NEG_RSP,
			Flags:   0x00,
			Length:  RDP_NEG_LENGTH,
			Payload: x.selProtocol,
		},
	}

	payload := x.serialize(resPdu)
	_, err = x.transport.Write(payload)
	if err != nil {
		x.handleError(errors.New("[X224] failed to write response: " + err.Error()))
	}
}

// 服务端处理客户端消息
func (x *X224) serverHandleClientMessage() {
	for {
		_, packet, err := x.transport.Read()
		if err != nil {
			continue
		}

		reqPdu := &X224PDU{}
		reader := bytes.NewReader(packet)

		err = x.parsePduHeader(reader, reqPdu)
		if err != nil {
			x.handleError(errors.New("[X224] failed to parse pdu header: " + err.Error()))
			return
		}

		switch reqPdu.Type {
		case X224_CONNECTION_REQUEST:
			x.serverHandleConnectionRequest(reqPdu, reader)
		case X224_DATA:
			x.handleData(reader)
		}
	}
}

// 服务端处理客户端发来的连接请求
func (x *X224) serverHandleConnectionRequest(reqPdu *X224PDU, reader *bytes.Reader) {

	// 解析Cookie
	cookieBuff := make([]byte, 0, 32)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			x.handleError(errors.New("[X224] failed to read cookie: " + err.Error()))
			return
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
		x.handleError(errors.New("[X224] failed to parse negotiation: " + err.Error()))
		return
	}
	reqPdu.NegMsg = reqNeg

	// 确定使用协议
	x.selProtocol = PROTOCOL_SSL

	// 响应请求
	x.serverResponseToClient(reqPdu)
}
