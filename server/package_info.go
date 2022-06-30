package main

import (
	"crypto/sha256"
	"github.com/lucas-clemente/quic-go"
	"gobuilder/quicpkg"
	"io"
	"os"
)

func GetPackageInformation(executable string, w io.Writer) (*quicpkg.PacketPackageInfo, error) {
	o, err := os.Open(executable)
	if err != nil {
		return nil, err
	}
	defer o.Close()

	length, err := o.Seek(io.SeekEnd, 0)
	if err != nil {
		return nil, err
	}

	if _, err := o.Seek(io.SeekStart, 0); err != nil {
		return nil, err
	}

	hashChunk := make([]byte, 1024)
	hashFunc := sha256.New()
	for {
		chunkLen, err := o.Read(hashChunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		chunk := hashChunk[:chunkLen]
		if w != nil {
			if _, err := w.Write(chunk); err != nil {
				return nil, err
			}
		}
		hashFunc.Write(chunk)
	}
	hashSum := hashFunc.Sum(nil)

	response := quicpkg.PacketPackageInfo{
		BinarySize: uint64(length),
		Signature: quicpkg.Data[uint8, []byte]{
			Size: uint8(len(hashSum)),
			Data: hashSum,
		},
	}

	return &response, nil
}

func HandlePackageInfoCommand(stream quic.Stream) error {
	request := quicpkg.PacketPackageName{}
	if err := request.Read(stream); err != nil {
		return err
	}

	pkg, ok := ServerConfig.Packages[request.Package.Data]
	if !ok {
		resp, err := quicpkg.NewErrorPacket(quicpkg.ErrorCodeNotFoundPackage,
			"package `"+request.Package.Data+"` invalid")
		if err != nil {
			return err
		}
		if err := resp.WriteWithOp(stream); err != nil {
			return err
		}
		return nil
	}

	response, err := GetPackageInformation(pkg.Executable, nil)
	if err != nil {
		return err
	}

	if err := response.WriteWithOp(stream); err != nil {
		return err
	}

	return nil
}
