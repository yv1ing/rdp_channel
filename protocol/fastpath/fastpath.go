package fastpath

import (
	"encoding/binary"
	"errors"
	"rdp_channel/protocol/core/transport"
)

// 协议常量
const (
	FASTPATH_PDU_HEADER_LENGTH  = 0x4 // updateHeader(1 bytes) + compressionFlags(1 bytes) + size(2 bytes)
	FASTPATH__MAX_PACKET_LENGTH = 0xffff

	FASTPATH_UPDATETYPE_ORDERS uint32 = 0x0
	FASTPATH_UPDATETYPE_BITMAP uint32 = 0x1
)

const (
	// update header: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/a1c4caa8-00ed-45bb-a06e-5177473766d3
	FASTPATH_UPDATE_HEADER     uint8 = 0x8
	FASTPATH_COMPRESSION_FLAGS uint8 = 0x0
)

var (
	FASTPATH_INVALID_PACKET_LENGTH = errors.New("[FASTPATH] invalid packet length")
)

type FastPath struct {
	transport transport.Transport
}

func New(transport transport.Transport) *FastPath {
	return &FastPath{
		transport: transport,
	}
}

func (fp *FastPath) Write(data []byte) (int, error) {
	/*
		构造PDU
		格式参考：https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/0ae3c114-1439-4465-8d3f-6585227eff7d
	*/
	dataLen := len(data)
	if dataLen > FASTPATH__MAX_PACKET_LENGTH {
		return dataLen, FASTPATH_INVALID_PACKET_LENGTH
	}

	size := uint16(FASTPATH_PDU_HEADER_LENGTH + dataLen)
	pdu := make([]byte, size)

	pdu[0] = FASTPATH_UPDATE_HEADER
	pdu[1] = FASTPATH_COMPRESSION_FLAGS
	binary.LittleEndian.PutUint16(pdu[2:4], size)

	copy(pdu[4:], data)

	return fp.transport.Write(pdu)
}

func (fp *FastPath) Read() (int, []byte, error) {
	_, packet, err := fp.transport.Read()
	if err != nil {
		return 0, nil, err
	}

	data := packet[FASTPATH_PDU_HEADER_LENGTH:]
	return len(data), data, nil
}
