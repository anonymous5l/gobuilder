package main

import (
	"bytes"
	"github.com/lucas-clemente/quic-go"
	"gobuilder/quicpkg"
)

func HandlePackageGetCommand(stream quic.Stream) error {
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

	buf := bytes.NewBuffer([]byte{})

	infoPackage, err := GetPackageInformation(pkg.Executable, buf)
	if err != nil {
		return err
	}

	responsePacket := quicpkg.PacketPackage{
		Signature: infoPackage.Signature,
		Data: quicpkg.Data[uint64, []byte]{
			Size: infoPackage.BinarySize,
			Data: buf.Bytes(),
		},
	}

	return responsePacket.WriteWithOp(stream)
}
