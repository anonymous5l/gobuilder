package quicpkg

import (
	"encoding/binary"
	"errors"
	"io"
)

var binaryOrder = binary.BigEndian

type DataType interface {
	~string | []byte
}

type DataSize interface {
	~uint8 | uint16 | uint32 | uint64
}

type Data[S DataSize, T DataType] struct {
	Size S
	Data T
}

func WriteData[S DataSize, T DataType](w io.Writer, data Data[S, T]) error {
	dataLen := data.Size
	if err := binary.Write(w, binaryOrder, dataLen); err != nil {
		return err
	}

	if dataLen > 0 {
		n, err := w.Write([]byte(data.Data))
		if err != nil {
			return err
		}
		if uint64(n) < uint64(dataLen) {
			return errors.New("buffer overflow")
		}
	}

	return nil
}

func ReadData[S DataSize, T DataType](r io.Reader, data *Data[S, T]) error {
	if err := binary.Read(r, binaryOrder, &data.Size); err != nil {
		return err
	}

	if data.Size > 0 {
		strBytes := make([]byte, data.Size)
		n, err := io.ReadFull(r, strBytes)
		if err != nil {
			return err
		}
		if uint64(n) < uint64(data.Size) {
			return errors.New("data corrupt")
		}

		data.Data = T(strBytes)
	}

	return nil
}

type BasicType interface {
	~bool | int8 | uint8 |
		int16 | uint16 |
		int32 | uint32 |
		int64 | uint64 |
		float32 | float64
}

func Read[T BasicType](r io.Reader, data *T) error {
	if err := binary.Read(r, binaryOrder, data); err != nil {
		return err
	}
	return nil
}

func Write[T BasicType](w io.Writer, data T) error {
	if err := binary.Write(w, binaryOrder, data); err != nil {
		return err
	}
	return nil
}
