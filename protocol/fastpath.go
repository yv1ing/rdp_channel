package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

/*
	定义协议相关的常量值
*/

// FASTPATH_UPDATE_HEADER
// UpdateHeader(1 byte)
// updateCode(4 bits) | fragmentation(2 bits) | compression(2 bits)
// 参考：https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/a1c4caa8-00ed-45bb-a06e-5177473766d3
const FASTPATH_UPDATE_HEADER uint8 = 0b00010000

/*
	FastPath PDU定义
	参考：https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/0ae3c114-1439-4465-8d3f-6585227eff7d
*/

type FastPathPDU struct {
	UpdateHeader     uint8
	CompressionFlags uint8
	Size             uint16
	Payload          BitmapUpdateData
}

/*
	Bitmap Data 定义
	参考：https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/d681bb11-f3b5-4add-b092-19fe7075f9e3
		 https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/84a3d4d2-5523-4e49-9a48-33952c559485
*/

type BitmapUpdateData struct {
	UpdateType       uint16
	NumberRectangles uint16
	Payload          BitmapData
}

type BitmapData struct {
	DestLeft     uint16
	DestTop      uint16
	DestRight    uint16
	DestBottom   uint16
	Width        uint16
	Height       uint16
	BitsPerPixel uint16
	Flags        uint16
	BitmapLength uint16
	Payload      []byte
}

/*
	FastPath协议封装
*/

type FastPath struct {
	transport *TPKT
}

func NewFastPath(conn io.ReadWriter) *FastPath {
	return &FastPath{transport: NewTPKT(conn)}
}

/*
	FastPath封包
*/

func (f *FastPath) Write(payload []byte) error {
	payloadLen := len(payload)

	// 构造BitmapData
	bitmapData := BitmapData{
		DestLeft:     0x0000,
		DestTop:      0x0000,
		DestRight:    0x000f, // 15 = 0 + 16 -1
		DestBottom:   0x000f, // 15
		Width:        0x0010, // 16
		Height:       0x0010, // 16
		BitsPerPixel: 0x0010, // 16位每像素
		Flags:        0x0000, // 无压缩
		BitmapLength: uint16(payloadLen),
		Payload:      payload,
	}

	// 序列化BitmapData
	var bitmapDataBuff bytes.Buffer
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.DestLeft)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.DestTop)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.DestRight)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.DestBottom)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.Width)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.Height)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.BitsPerPixel)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.Flags)
	binary.Write(&bitmapDataBuff, binary.LittleEndian, bitmapData.BitmapLength)
	bitmapDataBuff.Write(bitmapData.Payload)

	// 构造BitmapUpdateData
	bitmapUpdateData := BitmapUpdateData{
		UpdateType:       0x0001,
		NumberRectangles: 0x0001,
		Payload:          bitmapData,
	}

	// 序列化BitmapUpdateData
	var updateDataBuff bytes.Buffer
	binary.Write(&updateDataBuff, binary.LittleEndian, bitmapUpdateData.UpdateType)
	binary.Write(&updateDataBuff, binary.LittleEndian, bitmapUpdateData.NumberRectangles)
	updateDataBuff.Write(bitmapDataBuff.Bytes())

	updateDataBytes := updateDataBuff.Bytes()
	updateDataLength := len(updateDataBytes)

	// 构造FastPathPDU
	var fastPathPDUBuff bytes.Buffer
	fastPathPDUBuff.WriteByte(FASTPATH_UPDATE_HEADER)
	fastPathPDUBuff.WriteByte(0x00)

	size := uint16(updateDataLength)
	if size <= 0x7F {
		fastPathPDUBuff.WriteByte(byte(size))
	} else {
		fastPathPDUBuff.WriteByte(byte((size >> 8) | 0x80))
		fastPathPDUBuff.WriteByte(byte(size & 0xFF))
	}

	fastPathPDUBuff.Write(updateDataBytes)
	return f.transport.Write(fastPathPDUBuff.Bytes())
}

/*
	FastParh解包
*/

func (f *FastPath) Read() (payload []byte, err error) {
	packet, err := f.transport.Read()
	if err != nil {
		return nil, errors.New("[FASTPATH] read packet error: " + err.Error())
	}

	reader := bytes.NewReader(packet)

	// 解析FastPathPDU
	fastPathPDU := FastPathPDU{}

	fastPathPDU.UpdateHeader, err = reader.ReadByte()
	if err != nil {
		return nil, errors.New("[FASTPATH] read fastpathpdu's update header error: " + err.Error())
	}

	fastPathPDU.CompressionFlags, err = reader.ReadByte()
	if err != nil {
		return nil, errors.New("[FASTPATH] read fastpathpdu's compression flags error: " + err.Error())
	}

	// 解析Size字段
	lengthByte, err := reader.ReadByte()
	if err != nil {
		return nil, errors.New("[FASTPATH] read fastpathpdu's size first byte error: " + err.Error())
	}

	var size uint16
	if (lengthByte & 0x80) != 0 {
		lengthByte2, err := reader.ReadByte()
		if err != nil {
			return nil, errors.New("[FASTPATH] read fastpathpdu's size second byte error: " + err.Error())
		}
		size = (uint16(lengthByte&0x7F) << 8) | uint16(lengthByte2)
	} else {
		size = uint16(lengthByte)
	}

	fastPathPDU.Size = size

	// 提取FastPathPDU的Payload
	fastPathPDUPayload := make([]byte, fastPathPDU.Size)
	n, err := reader.Read(fastPathPDUPayload)
	if err != nil || n != int(size) {
		return nil, errors.New("[FASTPATH] read fastpathpdu's payload error: " + err.Error())
	}

	// 解析BitmapUpdateData
	updateReader := bytes.NewReader(fastPathPDUPayload)
	var updateType uint16
	if err := binary.Read(updateReader, binary.LittleEndian, &updateType); err != nil {
		return nil, errors.New("[FASTPATH] read bitmapupdatedata's update type error: " + err.Error())
	}

	var numRects uint16
	if err := binary.Read(updateReader, binary.LittleEndian, &numRects); err != nil {
		return nil, errors.New("[FASTPATH] read bitmapupdatedata's number rectangles error: " + err.Error())
	}

	// 解析BitmapData
	var bitmapData BitmapData
	fields := []interface{}{
		&bitmapData.DestLeft,
		&bitmapData.DestTop,
		&bitmapData.DestRight,
		&bitmapData.DestBottom,
		&bitmapData.Width,
		&bitmapData.Height,
		&bitmapData.BitsPerPixel,
		&bitmapData.Flags,
		&bitmapData.BitmapLength,
	}

	for _, field := range fields {
		if err := binary.Read(updateReader, binary.LittleEndian, field); err != nil {
			return nil, errors.New("[FASTPATH] read bitmapdata field error: " + err.Error())
		}
	}

	if bitmapData.BitmapLength > uint16(updateReader.Len()) {
		return nil, errors.New("[FASTPATH] invalid bitmaplength")
	}

	// 提取真实的Payload
	bitmapPayload := make([]byte, bitmapData.BitmapLength)
	if _, err := updateReader.Read(bitmapPayload); err != nil {
		return nil, errors.New("[FASTPATH] read bitmappayload error: " + err.Error())
	}

	return bitmapPayload, nil
}
