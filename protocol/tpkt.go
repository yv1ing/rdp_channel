package protocol

import (
	"encoding/binary"
	"errors"
	"io"
)

/*
	定义协议相关的常量值
*/

const (
	TPKT_VERSION           = 0x03
	TPKT_RESERVED          = 0x00
	TPKT_HEADER_LENGTH     = 0x04
	TPKT_MAX_PACKET_LENGTH = 0xffff
)

/*
	TPKT结构体封装
*/

type TPKT struct {
	conn io.ReadWriter
}

func NewTPKT(conn io.ReadWriter) *TPKT {
	return &TPKT{
		conn: conn,
	}
}

/*
	TPKT封包
*/

func (t *TPKT) Write(payload []byte) error {
	pduLength := TPKT_HEADER_LENGTH + len(payload)
	if pduLength > TPKT_MAX_PACKET_LENGTH {
		return errors.New("[TPKT] packet length too long")
	}

	// 构造TPKT头（4 bytes）：
	pdu := make([]byte, pduLength)
	pdu[0] = TPKT_VERSION
	pdu[1] = TPKT_RESERVED
	binary.BigEndian.PutUint16(pdu[2:4], uint16(pduLength))

	// 装入TPKT载荷
	copy(pdu[4:], payload)

	_, err := t.conn.Write(pdu)
	if err != nil {
		return errors.New("[TPKT] write error: " + err.Error())
	}

	return nil
}

/*
	TPKT解包
*/

func (t *TPKT) Read() ([]byte, error) {
	// 验证TPKT头
	pduHeader := make([]byte, TPKT_HEADER_LENGTH)
	_, err := io.ReadFull(t.conn, pduHeader)
	if err != nil {
		return nil, errors.New("[TPKT] read pdu header error: " + err.Error())
	}

	if pduHeader[0] != TPKT_VERSION {
		return nil, errors.New("[TPKT] version mismatch")
	}

	pduLength := binary.BigEndian.Uint16(pduHeader[2:4])
	if pduLength > TPKT_MAX_PACKET_LENGTH {
		return nil, errors.New("[TPKT] packet length too long")
	}

	// 读取TPKT载荷
	payloadLength := pduLength - TPKT_HEADER_LENGTH
	payload := make([]byte, payloadLength)
	_, err = io.ReadFull(t.conn, payload)
	if err != nil {
		return nil, errors.New("[TPKT] read pdu payload error: " + err.Error())
	}

	return payload, nil
}
