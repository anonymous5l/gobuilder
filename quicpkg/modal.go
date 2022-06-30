package quicpkg

import (
	"errors"
	"io"
)

type Operation byte

const (
	OperationPacketError = iota
	OperationPackageInfo
	OperationPackageGet
	OperationPacketReplace
)

func (o Operation) String() string {
	switch o {
	case OperationPacketError:
		return "OperationError"
	case OperationPackageInfo:
		return "PackageInfo"
	case OperationPackageGet:
		return "Package"
	case OperationPacketReplace:
		return "PackageReplace"
	}

	return "Unknown"
}

type ErrorCode byte

const (
	ErrorCodeNotFoundPackage ErrorCode = iota + 1
	ErrorCodeSystem
)

type PacketErrorResponse struct {
	ErrCode    ErrorCode
	ErrMessage Data[uint16, string]
}

func NewErrorPacket(code ErrorCode, message string) (*PacketErrorResponse, error) {
	messageLen := len(message)
	if messageLen > 0xFFFF {
		return nil, errors.New("message length overflow")
	}

	return &PacketErrorResponse{
		ErrCode: code,
		ErrMessage: Data[uint16, string]{
			Size: uint16(messageLen),
			Data: message,
		},
	}, nil
}

func (p *PacketErrorResponse) Read(stream io.Reader) error {
	var errCode byte
	if err := Read[byte](stream, &errCode); err != nil {
		return err
	}
	if err := ReadData[uint16, string](stream, &p.ErrMessage); err != nil {
		return err
	}
	p.ErrCode = ErrorCode(errCode)
	return nil
}

func (p *PacketErrorResponse) Write(stream io.Writer) error {
	if err := Write[byte](stream, byte(p.ErrCode)); err != nil {
		return err
	}
	if err := WriteData(stream, p.ErrMessage); err != nil {
		return err
	}
	return nil
}

func (p *PacketErrorResponse) WriteWithOp(stream io.Writer) error {
	if err := Write[byte](stream, OperationPacketError); err != nil {
		return err
	}
	return p.Write(stream)
}

type PacketReplaceRequest struct {
	Package   Data[uint16, string]
	Signature Data[uint8, []byte]
	Binary    Data[uint64, []byte]
}
