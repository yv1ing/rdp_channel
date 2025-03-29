package tpkt

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

// 协议常量
const (
	TPKT_VERSION           = 0x03
	TPKT_RESERVED          = 0x00
	TPKT_HEADER_LENGTH     = 0x04
	TPKT_MAX_PACKET_LENGTH = 0xffff
)

var (
	TPKT_INVALID_VERSION       = errors.New("[TPKT] invalid version")
	TPKT_INVALID_PACKET_LENGTH = errors.New("[TPKT] invalid packet length")
)

// TPKT 协议封装
type TPKT struct {
	connection net.Conn
	readBuff   *bufio.Reader
}

func New(conn net.Conn) *TPKT {
	tpkt := &TPKT{
		connection: conn,
		readBuff:   bufio.NewReader(conn),
	}

	return tpkt
}

// Write 发送TPKT包
func (tpkt *TPKT) Write(data []byte) (int, error) {
	dataLen := len(data)
	if dataLen > (TPKT_MAX_PACKET_LENGTH - TPKT_HEADER_LENGTH) {
		return 0, TPKT_INVALID_PACKET_LENGTH
	}

	// TPKT封包
	packet := make([]byte, TPKT_HEADER_LENGTH+dataLen)
	packet[0] = TPKT_VERSION
	packet[1] = TPKT_RESERVED
	binary.BigEndian.PutUint16(packet[2:4], uint16(TPKT_HEADER_LENGTH+dataLen)) // 第2、3字节为tpkt包的长度

	// 装填载荷
	copy(packet[TPKT_HEADER_LENGTH:], data)

	// 发送TPKT包
	return tpkt.connection.Write(packet)
}

func (tpkt *TPKT) Read() (int, []byte, error) {
	// 验证TPKT头
	header := make([]byte, TPKT_HEADER_LENGTH)
	_, err := io.ReadFull(tpkt.readBuff, header)
	if err != nil {
		return 0, nil, err
	}

	// 验证TPKT版本
	if header[0] != TPKT_VERSION {
		return 0, nil, TPKT_INVALID_VERSION
	}

	// 验证载荷长度
	length := binary.BigEndian.Uint16(header[2:4])
	if length > TPKT_MAX_PACKET_LENGTH {
		return 0, nil, TPKT_INVALID_PACKET_LENGTH
	}

	// 读取载荷数据
	dataLen := length - TPKT_HEADER_LENGTH
	data := make([]byte, dataLen)
	if _, err := io.ReadFull(tpkt.readBuff, data); err != nil {
		return 0, nil, err
	}

	return int(dataLen), data, nil
}
