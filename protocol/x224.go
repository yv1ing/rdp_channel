package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

/* 协议常量 */

// X224消息头部长度
const (
	X224_HEADER_LENGTH = 0x07
)

// X224消息类型字段标识
const (
	X224_CONNECTION_REQUEST byte = 0xE0
	X224_CONNECTION_CONFIRM byte = 0xD0
	X224_DISCONNECT_REQUEST byte = 0x80
	X224_DATA               byte = 0xF0
	X224_ERROR              byte = 0x70
)

// X224 协议封装
type X224 struct {
	transport   *TPKT
	reqProtocol uint32
	selProtocol uint32
}

type X224PDU struct {
	Len     uint8
	Type    byte
	DstRef  uint16 // 大端序
	SrcRef  uint16 // 大端序
	ClsOpt  uint8
	Payload []byte
}

func NewX224(conn io.ReadWriter) *X224 {
	return &X224{
		transport: NewTPKT(conn),
	}
}

// 从字节流中解析PDU头部
func (x *X224) parsePDUHeader(reader *bytes.Reader, pdu *X224PDU) error {
	var err error

	// 读取Len字段
	err = binary.Read(reader, binary.BigEndian, &pdu.Len)
	if err != nil {
		return errors.New("[X224] failed to read pdu length: " + err.Error())
	}

	// 读取Type字段
	err = binary.Read(reader, binary.BigEndian, &pdu.Type)
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
	err = binary.Read(reader, binary.BigEndian, &pdu.ClsOpt)
	if err != nil {
		return errors.New("[X224] failed to read pdu clsopt: " + err.Error())
	}

	return nil
}

// 序列化X224PDU
func (x *X224) serializeX224PDU(pdu *X224PDU) []byte {
	buff := bytes.NewBuffer(nil)
	_ = binary.Write(buff, binary.BigEndian, pdu.Len)
	_ = binary.Write(buff, binary.BigEndian, pdu.Type)
	_ = binary.Write(buff, binary.BigEndian, pdu.DstRef)
	_ = binary.Write(buff, binary.BigEndian, pdu.SrcRef)
	_ = binary.Write(buff, binary.BigEndian, pdu.ClsOpt)

	buff.Write(pdu.Payload)
	return buff.Bytes()
}

// 封包
func (x *X224) Write(payload []byte) error {
	pdu := &X224PDU{
		Len:     uint8(X224_HEADER_LENGTH + len(payload)), // 头部长度 + 载荷字段
		Type:    X224_DATA,
		DstRef:  0xf0,
		SrcRef:  0xf1,
		ClsOpt:  0x0,
		Payload: payload,
	}

	payloadBytes := x.serializeX224PDU(pdu)
	err := x.transport.Write(payloadBytes)
	if err != nil {
		return errors.New("[X224] failed to write: " + err.Error())
	}

	return nil
}

// 解包
func (x *X224) Read() ([]byte, error) {
	packet, err := x.transport.Read()
	if err != nil {
		return nil, errors.New("[X224] failed to read: " + err.Error())
	}

	pdu := &X224PDU{}
	reader := bytes.NewReader(packet)

	err = x.parsePDUHeader(reader, pdu)
	if err != nil {
		return nil, errors.New("[X224] failed to parse pdu header: " + err.Error())
	}

	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.New("[X224] failed to read payload: " + err.Error())
	}

	return payload, nil
}
