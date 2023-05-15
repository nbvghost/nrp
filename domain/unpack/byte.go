package unpack

import (
	"bytes"
	"encoding/binary"
)

type HeadType int32

const (
	HeadTypeData      HeadType = 50
	HeadTypeTCP       HeadType = 100
	HeadTypeHTTP      HeadType = 200
	HeadTypeHeartbeat HeadType = 300
)

type Head struct {
	ID     uint64   //8
	Length int32    //4
	Type   HeadType //4
}

func HeadLen() int {
	return 8 + 4 + 4
}

func NewHead(id uint64, length int32, headType HeadType) Head {
	return Head{
		ID:     id,
		Length: length,
		Type:   headType,
	}
}

func ToBytes(head Head, body []byte) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	err := binary.Write(buffer, binary.LittleEndian, &head.ID)
	if err != nil {
		return nil, err
	} //8
	err = binary.Write(buffer, binary.LittleEndian, &head.Length)
	if err != nil {
		return nil, err
	} //4
	err = binary.Write(buffer, binary.LittleEndian, &head.Type)
	if err != nil {
		return nil, err
	} //4
	err = binary.Write(buffer, binary.LittleEndian, &body)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func FromBytes(pack []byte) (head Head, err error) {
	readBuffer := bytes.NewBuffer(pack)
	err = binary.Read(readBuffer, binary.LittleEndian, &head.ID)
	if err != nil {
		return
	}
	err = binary.Read(readBuffer, binary.LittleEndian, &head.Length)
	if err != nil {
		return
	}
	err = binary.Read(readBuffer, binary.LittleEndian, &head.Type)
	if err != nil {
		return
	}
	return
}
